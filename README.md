# GoCrawler

A concurrent web crawler + image indexer written in Go.  
It crawls pages starting from a seed URL, extracts image links (including `srcset`/`picture` and SVG `<image>` refs), downloads images, generates 200px-wide thumbnails, stores metadata in MySQL, and provides a small web UI to search/preview the results.

## Features

- Recursive crawling with configurable depth
- Worker-pool crawling (goroutines + channels)
- Optional JS-rendered crawling for SPAs using **chromedp** (`--js`)
- Optional external link traversal (`--external`)
  - Even when external page links are disabled, **image URLs can still be external/CDN** (images are not domain-filtered)
- Downloads **JPEG/PNG/GIF/SVG**
- Saves originals to `./images/` and thumbnails to `./thumbnails/`
- Stores image metadata in MySQL (`original_url`, `saved_path`, `thumb_path`, `filename`, `width`, `height`, `format`)
- Simple HTML search UI (filter by URL/filename/format)

## Requirements

- Go (use the version from `go.mod`)
- MySQL running locally (or update connection settings in the code)
- If using `--js`: Chrome/Chromium available (chromedp drives a local browser)

## Project layout

```txt
cmd/
  crawler/     # CLI crawler
  webserver/   # simple search UI
internal/
  crawler/     # fetch + parse + worker pool
  images/      # downloader + thumbnail generator
  storage/     # MySQL access + repository
  web/         # templates (and web helpers)
images/        # downloaded images (created at runtime)
thumbnails/    # generated thumbnails (created at runtime)
```

## Setup

### 1) MySQL schema

Create a DB + table (adjust names/types if you want):

```sql
CREATE DATABASE IF NOT EXISTS crawlerdb;
USE crawlerdb;

CREATE TABLE IF NOT EXISTS images (
  id INT AUTO_INCREMENT PRIMARY KEY,
  original_url TEXT NOT NULL,
  saved_path TEXT NOT NULL,
  thumb_path TEXT NOT NULL,
  filename VARCHAR(255) NOT NULL,
  width INT NOT NULL,
  height INT NOT NULL,
  format VARCHAR(64) NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### 2) Database credentials

Update your MySQL setup in:
- `cmd/crawler/main.go`
- `cmd/webserver/main.go`

## Run

### Crawl pages and index images

From the repo root:

```bash
go run ./cmd/crawler --url "https://example.com" --depth 2
```

Common options:

```bash
go run ./cmd/crawler   --url "https://example.com"   --depth 2   --workers 10   --img-workers 4   --timeout 120   --img-timeout 20
```

Enable JS rendering (for SPA pages):

```bash
go run ./cmd/crawler --url "https://example.com" --depth 2 --js
```

Follow external page links too:

```bash
go run ./cmd/crawler --url "https://example.com" --depth 2 --external
```

### Start the web UI

After crawling:

```bash
go run ./cmd/webserver
```

Open:

- `http://localhost:8080`

Search using query params (the UI form builds these):
- `?url=<contains>`
- `?filename=<contains>`
- `?format=image/png` (or `image/jpeg`, `image/gif`, `image/svg+xml`)

Example:

```txt
http://localhost:8080/?filename=logo&format=image/png
```

## CLI flags (crawler)

- `--url` (required): seed URL to start from
- `--depth` (default `2`): crawl depth (`0` = only seed)
- `--workers` (default `10`): crawler worker pool size
- `--external` (default `false`): follow external page links
- `--js` (default `false`): render pages with chromedp before parsing
- `--timeout` (default `120`): global crawl timeout in seconds
- `--img-workers` (default `4`): number of image processing workers
- `--img-timeout` (default `20`): per-image processing timeout in seconds
- `--max-goroutines` (default `200`): safety cap (crawl + image workers)

## Notes

- This is a learning project: be nice to websites (small depth/workers, respect robots/terms).
- Only `http`/`https` are crawled; `mailto:`, `javascript:` and fragment-only links are ignored.
- Some image servers send content types with charset or extra parameters; if you hit “unsupported image type”, that’s the check in the downloader.

## License

Add a `LICENSE` file if you plan to publish this as open-source.
