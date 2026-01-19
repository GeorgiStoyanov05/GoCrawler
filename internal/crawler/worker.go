package crawler

import (
	"context"
	"sync"
)

type CrawlJob struct {
	URL            string
	Depth          int
	FollowExternal bool
	UseJS          bool
}

type CrawlResult struct {
	URL       string
	Links     []string
	ImageURLs []string
	Depth     int
	Err       error
}

type WorkerPool struct {
	ctx        context.Context
	cancel     context.CancelFunc
	maxWorkers int
	jobs       chan CrawlJob
	results    chan CrawlResult
	wg         sync.WaitGroup
}

func NewWorkerPool(parent context.Context, maxWorkers, jobBuf, resultBuf int) *WorkerPool {
	ctx, cancel := context.WithCancel(parent)
	return &WorkerPool{
		ctx:        ctx,
		cancel:     cancel,
		maxWorkers: maxWorkers,
		jobs:       make(chan CrawlJob, jobBuf),
		results:    make(chan CrawlResult, resultBuf),
	}
}

func (wp *WorkerPool) Jobs() chan<- CrawlJob       { return wp.jobs }
func (wp *WorkerPool) Results() <-chan CrawlResult { return wp.results }

func (wp *WorkerPool) Start(process func(context.Context, CrawlJob) CrawlResult) {
	for i := 0; i < wp.maxWorkers; i++ {
		wp.wg.Add(1)
		go func() {
			defer wp.wg.Done()
			for job := range wp.jobs {
				res := process(wp.ctx, job)

				select {
				case wp.results <- res:
				case <-wp.ctx.Done():
					return
				}
			}
		}()
	}
}

func (wp *WorkerPool) Stop() {
	close(wp.jobs)
	wp.wg.Wait()
	close(wp.results)
	wp.cancel()
}
