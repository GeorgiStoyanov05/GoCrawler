package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"GoCrawler/internal/crawler"
	"GoCrawler/internal/images"
	"GoCrawler/internal/storage"

	"github.com/chromedp/chromedp"
)

/*
go run cmd/crawler/main.go \
  --url=http \
  --depth=1 \
  --js \
	--external true

go run cmd/webserver/main.go
*/

func main() {
	startURL := flag.String("url", "", "Start URL to crawl (required)")
	maxDepth := flag.Int("depth", 2, "Max depth (0 = only seed page)")
	maxWorkers := flag.Int("workers", 10, "Number of crawler workers")
	followExternal := flag.Bool("external", false, "Follow external page links")
	useJS := flag.Bool("js", false, "Use headless browser (chromedp) to render JS pages")

	timeout := flag.Int("timeout", 120, "Global timeout in seconds (default: 120)")
	imgWorkers := flag.Int("img-workers", 4, "Number of image processing workers")
	imgTimeout := flag.Int("img-timeout", 20, "Per-image processing timeout in seconds")

	maxG := flag.Int("max-goroutines", 200, "Hard cap for total goroutines (crawl + image workers)")

	flag.Parse()

	fmt.Println("=== CRAWLER START ===")
	fmt.Println("startURL =", *startURL)
	fmt.Println("maxDepth =", *maxDepth)
	fmt.Println("workers  =", *maxWorkers)
	fmt.Println("external =", *followExternal)
	fmt.Println("js       =", *useJS)
	fmt.Println("timeout  =", *timeout, "seconds")
	fmt.Println("imgWorkers =", *imgWorkers)
	fmt.Println("imgTimeout =", *imgTimeout, "seconds")
	fmt.Println("maxGoroutines =", *maxG)
	fmt.Println("=====================")

	if *startURL == "" {
		log.Fatal("missing required flag: --url")
	}

	if (*maxWorkers + *imgWorkers + 10) > *maxG {
		log.Fatalf("too many goroutines requested: workers=%d img-workers=%d max-goroutines=%d",
			*maxWorkers, *imgWorkers, *maxG)
	}

	baseCtx := context.Background()
	if *useJS {
		fmt.Println("[JS] starting chromedp allocator/context...")
		allocCtx, allocCancel := chromedp.NewExecAllocator(baseCtx, chromedp.DefaultExecAllocatorOptions[:]...)
		defer allocCancel()

		browserCtx, browserCancel := chromedp.NewContext(allocCtx)
		defer browserCancel()

		baseCtx = browserCtx
		fmt.Println("[JS] chromedp ready")
	}

	ctx, cancel := context.WithTimeout(baseCtx, time.Duration(*timeout)*time.Second)
	defer cancel()

	if err := os.MkdirAll("images", 0o755); err != nil {
		log.Fatal("Failed to create images dir:", err)
	}
	if err := os.MkdirAll("thumbnails", 0o755); err != nil {
		log.Fatal("Failed to create thumbnails dir:", err)
	}
	fmt.Println("[DIR] ensured ./images and ./thumbnails")

	store, err := storage.NewMySQLStorage("crawler", "password123", "localhost:3306", "crawlerdb")
	if err != nil {
		log.Fatal("Failed to connect to MySQL:", err)
	}
	fmt.Println("[DB] connected")
	repo := storage.NewImageRepository(store)

	pool := crawler.NewWorkerPool(ctx, *maxWorkers, 200, 200)
	pool.Start(crawler.ProcessJob)
	fmt.Println("[POOL] started crawler worker pool with", *maxWorkers, "workers")

	imageJobs := make(chan string, 256)
	var imgWG sync.WaitGroup

	for i := 0; i < *imgWorkers; i++ {
		imgWG.Add(1)
		go func(id int) {
			defer imgWG.Done()
			fmt.Println("[IMG WORKER START]", id)

			for {
				select {
				case <-ctx.Done():
					fmt.Println("[IMG WORKER EXIT]", id, "ctx done:", ctx.Err())
					return

				case imgURL, ok := <-imageJobs:
					if !ok {
						fmt.Println("[IMG WORKER EXIT]", id, "imageJobs closed")
						return
					}
					if ctx.Err() != nil {
						fmt.Println("[IMG WORKER EXIT]", id, "ctx err:", ctx.Err())
						return
					}

					fmt.Println("[IMG]", imgURL)

					imgCtx, cancel := context.WithTimeout(ctx, time.Duration(*imgTimeout)*time.Second)
					meta, err := images.ProcessImage(imgCtx, imgURL, "./images", "./thumbnails")
					cancel()

					if err != nil {
						if ctx.Err() == nil {
							fmt.Println("[IMG ERR]", imgURL, err)
						}
						continue
					}
					if meta == nil {
						fmt.Println("[IMG SKIP] nil meta for", imgURL)
						continue
					}

					if err := repo.InsertImage(ctx, meta); err != nil && ctx.Err() == nil {
						fmt.Println("[DB ERR]", err)
						continue
					}

					fmt.Println("[IMG OK]", imgURL)
				}
			}
		}(i)
	}

	visited := make(map[string]struct{}, 4096)
	queue := make([]crawler.CrawlJob, 0, 4096)
	inFlight := 0

	seenImages := make(map[string]struct{}, 8192)
	imageBacklog := make([]string, 0, 8192)

	enqueue := func(raw string, depth int) {
		if depth < 0 {
			return
		}
		norm, err := crawler.NormalizeURL(*startURL, raw) // base is seed
		if err != nil || norm == "" {
			return
		}
		if _, ok := visited[norm]; ok {
			return
		}
		visited[norm] = struct{}{}

		queue = append(queue, crawler.CrawlJob{
			URL:            norm,
			Depth:          depth,
			FollowExternal: *followExternal,
			UseJS:          *useJS,
		})

		fmt.Println("[ENQUEUE]", norm, "depth=", depth, "queue=", len(queue), "visited=", len(visited))
	}

	fmt.Println("[SEED] enqueue start URL")
	enqueue(*startURL, *maxDepth)

	for len(queue) > 0 || inFlight > 0 || len(imageBacklog) > 0 {
		var (
			jobCh chan<- crawler.CrawlJob
			next  crawler.CrawlJob

			imgCh   chan<- string
			nextImg string
		)

		if len(queue) > 0 {
			jobCh = pool.Jobs()
			next = queue[0]
		}
		if len(imageBacklog) > 0 {
			imgCh = imageJobs
			nextImg = imageBacklog[0]
		}

		select {
		case <-ctx.Done():
			fmt.Println("[TIMEOUT]", ctx.Err())
			fmt.Println("Global timeout reached.")
			close(imageJobs)
			imgWG.Wait()
			pool.Stop()
			fmt.Println("[EXIT] stopped")
			return

		case jobCh <- next:
			queue = queue[1:]
			inFlight++
			fmt.Println("[DISPATCH]", next.URL, "depth=", next.Depth, "queue=", len(queue), "inFlight=", inFlight)

		case imgCh <- nextImg:
			imageBacklog = imageBacklog[1:]
			fmt.Println("[IMG DISPATCH]", nextImg, "imgBacklog=", len(imageBacklog), "imgChan=", len(imageJobs), "/", cap(imageJobs))

		case result, ok := <-pool.Results():
			if !ok {
				fmt.Println("[POOL] results closed unexpectedly")
				close(imageJobs)
				imgWG.Wait()
				return
			}
			inFlight--

			if result.Err != nil {
				if ctx.Err() == nil {
					fmt.Println("[RESULT ERR]", result.URL, "err=", result.Err)
				}
				continue
			}

			fmt.Println("[RESULT OK ]", result.URL, "links=", len(result.Links), "imgs=", len(result.ImageURLs), "depth=", result.Depth, "inFlight=", inFlight)

			fmt.Println("[IMAGES] from", result.URL, "count=", len(result.ImageURLs))
			for _, imgURL := range crawler.Unique(result.ImageURLs) {
				if imgURL == "" {
					continue
				}
				if _, ok := seenImages[imgURL]; ok {
					continue
				}
				seenImages[imgURL] = struct{}{}
				imageBacklog = append(imageBacklog, imgURL)
			}
			if len(result.ImageURLs) > 0 {
				fmt.Println("[IMG BACKLOG]", len(imageBacklog))
			}

			fmt.Println("[LINKS] from", result.URL, "count=", len(result.Links))
			if result.Depth > 0 {
				for _, link := range crawler.Unique(result.Links) {
					enqueue(link, result.Depth-1)
				}
			}
		}
	}

	fmt.Println("Shutting down workers...")
	close(imageJobs)
	imgWG.Wait()
	pool.Stop()
	fmt.Println("Crawl complete!")
}
