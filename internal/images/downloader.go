package images

import (
	"context"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var supported = map[string]bool{
	"image/jpeg":    true,
	"image/png":     true,
	"image/gif":     true,
	"image/svg+xml": true,
}

func DownloadImage(ctx context.Context, imageURL, saveDir string) (string, string, error) {

	_, err := url.Parse(imageURL)
	if err != nil {
		return "", "", err
	}

	client := &http.Client{Timeout: 15 * time.Second}

	req, err := http.NewRequestWithContext(ctx, "GET", imageURL, nil)
	if err != nil {
		return "", "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	ctype := resp.Header.Get("Content-Type")
	if !supported[ctype] {
		return "", "", errors.New("unsupported image type: " + ctype)
	}

	exts, _ := mime.ExtensionsByType(ctype)
	ext := ".bin"
	if len(exts) > 0 {
		ext = exts[0]
	}

	fname := strings.ReplaceAll(filepath.Base(imageURL), "?", "_")
	if !strings.Contains(fname, ".") {
		fname += ext
	}

	path := filepath.Join(saveDir, fname)

	out, err := os.Create(path)
	if err != nil {
		return "", "", err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", "", err
	}

	return path, ctype, nil
}
