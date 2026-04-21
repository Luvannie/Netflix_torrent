package downloads

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) CreateTask(ctx context.Context, searchResultID int64) (*DownloadTask, error) {
	var task DownloadTask
	err := r.pool.QueryRow(ctx, `
		INSERT INTO download_tasks (search_result_id, torrent_hash, status, progress, speed, peer_count, created_at, updated_at)
		VALUES ($1, '', 'REQUESTED', 0, 0, 0, now(), now())
		RETURNING id, search_result_id, torrent_hash, status, progress, speed, peer_count, created_at, updated_at
	`, searchResultID).Scan(
		&task.ID,
		&task.SearchResultID,
		&task.TorrentHash,
		&task.Status,
		&task.Progress,
		&task.Speed,
		&task.PeerCount,
		&task.CreatedAt,
		&task.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *Repository) GetTask(ctx context.Context, id int64) (*DownloadTask, error) {
	var task DownloadTask
	err := r.pool.QueryRow(ctx, `
		SELECT id, search_result_id, torrent_hash, status, progress, COALESCE(speed,0), peer_count, created_at, updated_at
		FROM download_tasks
		WHERE id=$1
	`, id).Scan(
		&task.ID,
		&task.SearchResultID,
		&task.TorrentHash,
		&task.Status,
		&task.Progress,
		&task.Speed,
		&task.PeerCount,
		&task.CreatedAt,
		&task.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTaskNotFound
	}
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *Repository) SaveTask(ctx context.Context, task DownloadTask) (*DownloadTask, error) {
	var updated DownloadTask
	err := r.pool.QueryRow(ctx, `
		UPDATE download_tasks
		SET torrent_hash=$2, status=$3, progress=$4, speed=$5, peer_count=$6, updated_at=$7
		WHERE id=$1
		RETURNING id, search_result_id, torrent_hash, status, progress, COALESCE(speed,0), peer_count, created_at, updated_at
	`, task.ID, task.TorrentHash, task.Status, task.Progress, task.Speed, task.PeerCount, time.Now()).Scan(
		&updated.ID,
		&updated.SearchResultID,
		&updated.TorrentHash,
		&updated.Status,
		&updated.Progress,
		&updated.Speed,
		&updated.PeerCount,
		&updated.CreatedAt,
		&updated.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTaskNotFound
	}
	if err != nil {
		return nil, err
	}
	return &updated, nil
}

func (r *Repository) ListTasks(ctx context.Context, limit, offset int) ([]DownloadTask, int64, error) {
	var total int64
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM download_tasks`).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, search_result_id, torrent_hash, status, progress, COALESCE(speed,0), peer_count, created_at, updated_at
		FROM download_tasks
		ORDER BY created_at DESC, id DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tasks []DownloadTask
	for rows.Next() {
		var task DownloadTask
		if err := rows.Scan(
			&task.ID,
			&task.SearchResultID,
			&task.TorrentHash,
			&task.Status,
			&task.Progress,
			&task.Speed,
			&task.PeerCount,
			&task.CreatedAt,
			&task.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		tasks = append(tasks, task)
	}
	return tasks, total, rows.Err()
}

func (r *Repository) RecordTransition(ctx context.Context, taskID int64, from *DownloadTaskStatus, to DownloadTaskStatus, reason string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO download_state_transitions (download_task_id, from_status, to_status, timestamp, reason)
		VALUES ($1, $2, $3, now(), $4)
	`, taskID, from, to, reason)
	return err
}

func (r *Repository) ListByStatusesUpdatedAsc(ctx context.Context, statuses []DownloadTaskStatus) ([]DownloadTask, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, search_result_id, torrent_hash, status, progress, COALESCE(speed,0), peer_count, created_at, updated_at
		FROM download_tasks
		WHERE status = ANY($1)
		ORDER BY updated_at ASC, id ASC
	`, statuses)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTasks(rows)
}

func (r *Repository) ListByStatusCreatedAsc(ctx context.Context, status DownloadTaskStatus) ([]DownloadTask, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, search_result_id, torrent_hash, status, progress, COALESCE(speed,0), peer_count, created_at, updated_at
		FROM download_tasks
		WHERE status = $1
		ORDER BY created_at ASC, id ASC
	`, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTasks(rows)
}

func scanTasks(rows pgx.Rows) ([]DownloadTask, error) {
	var tasks []DownloadTask
	for rows.Next() {
		var task DownloadTask
		if err := rows.Scan(
			&task.ID,
			&task.SearchResultID,
			&task.TorrentHash,
			&task.Status,
			&task.Progress,
			&task.Speed,
			&task.PeerCount,
			&task.CreatedAt,
			&task.UpdatedAt,
		); err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, rows.Err()
}
