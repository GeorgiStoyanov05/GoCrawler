package crawler

import (
	"context"
	"time"

	"github.com/chromedp/chromedp"
)

func FetchHTMLJS(ctx context.Context, url string) ([]byte, error) {
	cctx, cancel := chromedp.NewContext(ctx)
	defer cancel()

	var renderedHTML string

	tasks := chromedp.Tasks{
		chromedp.Navigate(url),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.Sleep(1 * time.Second),
		chromedp.OuterHTML("html", &renderedHTML, chromedp.ByQuery),
	}

	err := chromedp.Run(cctx, tasks)
	if err != nil {
		return nil, err
	}

	return []byte(renderedHTML), nil
}

func FetchPage(ctx context.Context, url string, useJS bool) ([]byte, error) {
	if useJS {
		body, err := FetchHTMLJS(ctx, url)
		if err == nil {
			return body, nil
		}
	}

	return FetchHTML(ctx, url)
}
