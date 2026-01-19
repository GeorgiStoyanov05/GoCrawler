package crawler

import (
	"context"
)

func ProcessJob(ctx context.Context, job CrawlJob) CrawlResult {
	body, err := FetchPage(ctx, job.URL, job.UseJS)
	if err != nil {
		return CrawlResult{URL: job.URL, Depth: job.Depth, Err: err}
	}

	linksRaw, err := ExtractLinks(body)
	if err != nil {
		return CrawlResult{URL: job.URL, Depth: job.Depth, Err: err}
	}
	imagesRaw, err := ExtractImages(body)
	if err != nil {
		return CrawlResult{URL: job.URL, Depth: job.Depth, Err: err}
	}

	links := make([]string, 0, len(linksRaw))
	for _, l := range linksRaw {
		n, err := NormalizeURL(job.URL, l)
		if err == nil && n != "" {
			links = append(links, n)
		}
	}

	images := make([]string, 0, len(imagesRaw))
	for _, img := range imagesRaw {
		n, err := NormalizeURL(job.URL, img)
		if err == nil && n != "" {
			images = append(images, n)
		}
	}

	if !job.FollowExternal {
		domain, err := ExtractDomain(job.URL) // eTLD+1
		if err == nil && domain != "" {
			links = FilterSameDomain(links, domain)
		}
	}

	return CrawlResult{
		URL:       job.URL,
		Links:     Unique(links),
		ImageURLs: Unique(images),
		Depth:     job.Depth,
		Err:       nil,
	}
}
