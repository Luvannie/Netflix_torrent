package catalog

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/netflixtorrent/backend-go/internal/pagination"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) ListMovies(ctx context.Context, page, size int) ([]Movie, int64, error) {
	limit, offset := pagination.LimitOffset(page, size)

	rows, err := r.pool.Query(ctx, `
		SELECT id, tmdb_id, title, overview, poster_path, backdrop_path, release_date,
		       vote_average, vote_count, popularity, original_language, original_title, catalog_added_at
		FROM movies
		ORDER BY popularity DESC NULLS LAST, id ASC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var movies []Movie
	for rows.Next() {
		var m Movie
		if err := rows.Scan(&m.ID, &m.TmdbID, &m.Title, &m.Overview, &m.PosterPath, &m.BackdropPath,
			&m.ReleaseDate, &m.VoteAverage, &m.VoteCount, &m.Popularity, &m.OriginalLanguage,
			&m.OriginalTitle, &m.CatalogAddedAt); err != nil {
			return nil, 0, err
		}
		movies = append(movies, m)
	}

	var total int64
	r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM movies`).Scan(&total)

	return movies, total, rows.Err()
}

func (r *Repository) GetMovieByID(ctx context.Context, id int64) (*Movie, error) {
	var m Movie
	err := r.pool.QueryRow(ctx, `
		SELECT id, tmdb_id, title, overview, poster_path, backdrop_path, release_date,
		       vote_average, vote_count, popularity, original_language, original_title, catalog_added_at
		FROM movies
		WHERE id=$1
	`, id).Scan(&m.ID, &m.TmdbID, &m.Title, &m.Overview, &m.PosterPath, &m.BackdropPath,
		&m.ReleaseDate, &m.VoteAverage, &m.VoteCount, &m.Popularity, &m.OriginalLanguage,
		&m.OriginalTitle, &m.CatalogAddedAt)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *Repository) GetMovieByTmdbID(ctx context.Context, tmdbID int) (*Movie, error) {
	var m Movie
	err := r.pool.QueryRow(ctx, `
		SELECT id, tmdb_id, title, overview, poster_path, backdrop_path, release_date,
		       vote_average, vote_count, popularity, original_language, original_title, catalog_added_at
		FROM movies
		WHERE tmdb_id=$1
	`, tmdbID).Scan(&m.ID, &m.TmdbID, &m.Title, &m.Overview, &m.PosterPath, &m.BackdropPath,
		&m.ReleaseDate, &m.VoteAverage, &m.VoteCount, &m.Popularity, &m.OriginalLanguage,
		&m.OriginalTitle, &m.CatalogAddedAt)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *Repository) SaveMovie(ctx context.Context, m *Movie) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO movies (tmdb_id, title, overview, poster_path, backdrop_path, release_date,
		                    vote_average, vote_count, popularity, original_language, original_title)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (tmdb_id) DO UPDATE SET
			title=EXCLUDED.title, overview=EXCLUDED.overview, poster_path=EXCLUDED.poster_path,
			backdrop_path=EXCLUDED.backdrop_path, release_date=EXCLUDED.release_date,
			vote_average=EXCLUDED.vote_average, vote_count=EXCLUDED.vote_count,
			popularity=EXCLUDED.popularity, original_language=EXCLUDED.original_language,
			original_title=EXCLUDED.original_title
		RETURNING id
	`, m.TmdbID, m.Title, m.Overview, m.PosterPath, m.BackdropPath, m.ReleaseDate,
		m.VoteAverage, m.VoteCount, m.Popularity, m.OriginalLanguage, m.OriginalTitle).Scan(&m.ID)
}

var _ = context.Background