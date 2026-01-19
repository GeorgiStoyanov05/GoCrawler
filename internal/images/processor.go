package images

import (
	"context"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/image/draw"
)

func GenerateThumbnail(srcPath, thumbDir string) (string, int, int, error) {
	file, err := os.Open(srcPath)
	if err != nil {
		return "", 0, 0, err
	}
	defer file.Close()

	img, format, err := image.Decode(file)
	if err != nil {
		return "", 0, 0, err
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	maxW := 200
	newW := maxW
	newH := int(float64(height) * (float64(maxW) / float64(width)))

	thumb := image.NewRGBA(image.Rect(0, 0, newW, newH))
	draw.CatmullRom.Scale(thumb, thumb.Bounds(), img, bounds, draw.Over, nil)

	fname := filepath.Base(srcPath)
	thumbName := "thumb_" + fname
	thumbPath := filepath.Join(thumbDir, thumbName)

	out, err := os.Create(thumbPath)
	if err != nil {
		return "", 0, 0, err
	}
	defer out.Close()

	switch format {
	case "jpeg":
		err = jpeg.Encode(out, thumb, &jpeg.Options{Quality: 85})
	case "png":
		err = png.Encode(out, thumb)
	case "gif":
		err = gif.Encode(out, thumb, nil)
	default:
		return "", 0, 0, io.ErrUnexpectedEOF
	}

	return thumbPath, newW, newH, err
}

func ProcessImage(ctx context.Context, url, saveDir, thumbDir string) (*ImageMetadata, error) {

	savedPath, ctype, err := DownloadImage(ctx, url, saveDir)
	if err != nil {
		return nil, err
	}

	if ctype == "image/svg+xml" {
		return &ImageMetadata{
			OriginalURL: url,
			SavedPath:   savedPath,
			ThumbPath:   savedPath,
			Filename:    filepath.Base(savedPath),
			Width:       0,
			Height:      0,
			Format:      "svg",
		}, nil
	}

	thumbPath, w, h, err := GenerateThumbnail(savedPath, thumbDir)
	if err != nil {
		return nil, err
	}

	return &ImageMetadata{
		OriginalURL: url,
		SavedPath:   savedPath,
		ThumbPath:   thumbPath,
		Filename:    filepath.Base(savedPath),
		Width:       w,
		Height:      h,
		Format:      ctype,
	}, nil
}
