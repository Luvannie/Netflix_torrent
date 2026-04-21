package library

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

type MediaItem struct {
	ID        int64     `json:"id"`
	TmdbID   *int      `json:"tmdbId"`
	Title    string    `json:"title"`
	Year     *int      `json:"year"`
	Type     string    `json:"type"`
	Files    []MediaFile `json:"files,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

type MediaFile struct {
	ID         int64   `json:"id"`
	MediaItemID int64  `json:"mediaItemId"`
	FilePath  string  `json:"filePath"`
	Container *string `json:"container"`
	Codec     *string `json:"codec"`
	Duration  *float64 `json:"duration"`
	Width     *int    `json:"width"`
	Height    *int    `json:"height"`
	Size      *int64  `json:"size"`
}

func (r *Repository) ListMediaItems(ctx context.Context, mediaType string, limit, offset int) ([]MediaItem, int64, error) {
	var total int64

	var args []any
	var query string

	if mediaType != "" {
		query = `SELECT COUNT(*) FROM media_items WHERE type = $1`
		args = []any{mediaType}
	} else {
		query = `SELECT COUNT(*) FROM media_items`
	}
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	if mediaType != "" {
		query = `SELECT id, tmdb_id, title, year, type, created_at FROM media_items WHERE type = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
		args = []any{mediaType, limit, offset}
	} else {
		query = `SELECT id, tmdb_id, title, year, type, created_at FROM media_items ORDER BY created_at DESC LIMIT $1 OFFSET $2`
		args = []any{limit, offset}
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []MediaItem
	for rows.Next() {
		var item MediaItem
		if err := rows.Scan(&item.ID, &item.TmdbID, &item.Title, &item.Year, &item.Type, &item.CreatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	if items == nil {
		items = []MediaItem{}
	}
	return items, total, nil
}

func (r *Repository) GetMediaItemByID(ctx context.Context, id int64) (*MediaItem, error) {
	var item MediaItem
	err := r.pool.QueryRow(ctx, `
		SELECT id, tmdb_id, title, year, type, created_at
		FROM media_items WHERE id = $1
	`, id).Scan(&item.ID, &item.TmdbID, &item.Title, &item.Year, &item.Type, &item.CreatedAt)
	if err != nil {
		return nil, err
	}

	files, err := r.ListMediaFiles(ctx, id)
	if err == nil {
		item.Files = files
	}

	return &item, nil
}

func (r *Repository) GetMediaFileByID(ctx context.Context, id int64) (*MediaFile, error) {
	var f MediaFile
	err := r.pool.QueryRow(ctx, `
		SELECT id, media_item_id, file_path, container, codec, duration, width, height, size
		FROM media_files WHERE id = $1
	`, id).Scan(&f.ID, &f.MediaItemID, &f.FilePath, &f.Container, &f.Codec, &f.Duration, &f.Width, &f.Height, &f.Size)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *Repository) ListMediaFiles(ctx context.Context, mediaItemID int64) ([]MediaFile, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, media_item_id, file_path, container, codec, duration, width, height, size
		FROM media_files WHERE media_item_id = $1
	`, mediaItemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []MediaFile
	for rows.Next() {
		var f MediaFile
		if err := rows.Scan(&f.ID, &f.MediaItemID, &f.FilePath, &f.Container, &f.Codec, &f.Duration, &f.Width, &f.Height, &f.Size); err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, rows.Err()
}

func (r *Repository) CreateMediaItem(ctx context.Context, tmdbID *int, title string, year *int, mediaType string) (*MediaItem, error) {
	var item MediaItem
	err := r.pool.QueryRow(ctx, `
		INSERT INTO media_items (tmdb_id, title, year, type)
		VALUES ($1, $2, $3, $4)
		RETURNING id, tmdb_id, title, year, type, created_at
	`, tmdbID, title, year, mediaType).Scan(&item.ID, &item.TmdbID, &item.Title, &item.Year, &item.Type, &item.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Repository) CreateMediaFile(ctx context.Context, mediaItemID int64, filePath string, container, codec *string, duration *float64, width, height *int, size *int64) (*MediaFile, error) {
	var f MediaFile
	err := r.pool.QueryRow(ctx, `
		INSERT INTO media_files (media_item_id, file_path, container, codec, duration, width, height, size)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, media_item_id, file_path, container, codec, duration, width, height, size
	`, mediaItemID, filePath, container, codec, duration, width, height, size).Scan(
		&f.ID, &f.MediaItemID, &f.FilePath, &f.Container, &f.Codec, &f.Duration, &f.Width, &f.Height, &f.Size)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *Repository) DeleteMediaItem(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM media_items WHERE id = $1`, id)
	return err
}
