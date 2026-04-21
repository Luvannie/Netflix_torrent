package search

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

func (r *Repository) CreateJob(ctx context.Context, query string) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx, `
		INSERT INTO search_jobs (query, status, created_at, updated_at)
		VALUES ($1, 'REQUESTED', $2, $2)
		RETURNING id
	`, query, time.Now()).Scan(&id)
	return id, err
}

func (r *Repository) GetJob(ctx context.Context, id int64) (*SearchJob, error) {
	var job SearchJob
	err := r.pool.QueryRow(ctx, `
		SELECT id, query, status, created_at, updated_at, COALESCE(error_message, '')
		FROM search_jobs
		WHERE id=$1
	`, id).Scan(&job.ID, &job.Query, &job.Status, &job.CreatedAt, &job.UpdatedAt, &job.ErrorMessage)
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *Repository) ListJobs(ctx context.Context, queryFilter string, limit, offset int) ([]SearchJob, int64, error) {
	var jobs []SearchJob
	var total int64

	countQuery := `SELECT COUNT(*) FROM search_jobs WHERE ($1 = '' OR lower(query) LIKE '%' || lower($1) || '%')`
	err := r.pool.QueryRow(ctx, countQuery, queryFilter).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, query, status, created_at, updated_at, COALESCE(error_message, '')
		FROM search_jobs
		WHERE ($1 = '' OR lower(query) LIKE '%' || lower($1) || '%')
		ORDER BY created_at DESC, id DESC
		LIMIT $2 OFFSET $3
	`, queryFilter, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var job SearchJob
		if err := rows.Scan(&job.ID, &job.Query, &job.Status, &job.CreatedAt, &job.UpdatedAt, &job.ErrorMessage); err != nil {
			return nil, 0, err
		}
		jobs = append(jobs, job)
	}

	return jobs, total, rows.Err()
}

func (r *Repository) UpdateJobStatus(ctx context.Context, id int64, status SearchJobStatus, errorMsg string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE search_jobs
		SET status=$2, updated_at=$3, error_message=$4
		WHERE id=$1
	`, id, status, time.Now(), errorMsg)
	return err
}

func (r *Repository) GetJobResults(ctx context.Context, jobID int64) ([]SearchResult, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, search_job_id, COALESCE(guid,''), title, COALESCE(link,''), COALESCE(permalink,''),
		       COALESCE(size,0), pub_date, COALESCE(seeders,0), COALESCE(leechers,0),
		       COALESCE(indexer,''), provider, COALESCE(hash,''), COALESCE(score,0), created_at
		FROM search_results
		WHERE search_job_id=$1
		ORDER BY score DESC NULLS LAST, seeders DESC NULLS LAST, id ASC
	`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.ID, &r.SearchJobID, &r.Guid, &r.Title, &r.Link, &r.Permalink,
			&r.Size, &r.PubDate, &r.Seeders, &r.Leechers, &r.Indexer, &r.Provider, &r.Hash, &r.Score, &r.CreatedAt); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

func (r *Repository) SaveResults(ctx context.Context, jobID int64, results []SearchResult) error {
	for _, result := range results {
		_, err := r.pool.Exec(ctx, `
			INSERT INTO search_results (search_job_id, guid, title, link, permalink, size, pub_date,
			                            seeders, leechers, indexer, provider, hash, score, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		`, jobID, result.Guid, result.Title, result.Link, result.Permalink, result.Size, result.PubDate,
			result.Seeders, result.Leechers, result.Indexer, result.Provider, result.Hash, result.Score, time.Now())
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) GetRequestedJobs(ctx context.Context, limit int) ([]SearchJob, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, query, status, created_at, updated_at, COALESCE(error_message, '')
		FROM search_jobs
		WHERE status = 'REQUESTED'
		ORDER BY created_at ASC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []SearchJob
	for rows.Next() {
		var job SearchJob
		if err := rows.Scan(&job.ID, &job.Query, &job.Status, &job.CreatedAt, &job.UpdatedAt, &job.ErrorMessage); err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

func (r *Repository) CancelJob(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE search_jobs
		SET status='CANCELLED', updated_at=$2
		WHERE id=$1 AND status IN ('REQUESTED', 'SEARCHING')
	`, id, time.Now())
	return err
}