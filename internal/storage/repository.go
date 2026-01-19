package storage

import (
	"GoCrawler/internal/images"
	"context"
)

type ImageRepository struct {
	db *MySQLStorage
}

func NewImageRepository(store *MySQLStorage) *ImageRepository {
	return &ImageRepository{db: store}
}

func (repo *ImageRepository) InsertImage(ctx context.Context, meta *images.ImageMetadata) error {
	query := `
        INSERT INTO images (original_url, saved_path, thumb_path, filename, width, height, format)
        VALUES (?, ?, ?, ?, ?, ?, ?)
    `

	_, err := repo.db.DB.ExecContext(ctx, query,
		meta.OriginalURL,
		meta.SavedPath,
		meta.ThumbPath,
		meta.Filename,
		meta.Width,
		meta.Height,
		meta.Format,
	)

	return err
}

func (repo *ImageRepository) SearchImages(ctx context.Context, params map[string]string) ([]images.ImageMetadata, error) {

	base := "SELECT original_url, saved_path, thumb_path, filename, width, height, format FROM images WHERE 1=1"
	args := []interface{}{}

	if v, ok := params["format"]; ok && v != "" {
		base += " AND format = ?"
		args = append(args, v)
	}
	if v, ok := params["filename"]; ok && v != "" {
		base += " AND filename LIKE ?"
		args = append(args, "%"+v+"%")
	}
	if v, ok := params["url"]; ok && v != "" {
		base += " AND original_url LIKE ?"
		args = append(args, "%"+v+"%")
	}

	rows, err := repo.db.DB.QueryContext(ctx, base, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []images.ImageMetadata

	for rows.Next() {
		var m images.ImageMetadata
		err := rows.Scan(
			&m.OriginalURL,
			&m.SavedPath,
			&m.ThumbPath,
			&m.Filename,
			&m.Width,
			&m.Height,
			&m.Format,
		)
		if err != nil {
			return nil, err
		}
		results = append(results, m)
	}

	return results, nil
}
