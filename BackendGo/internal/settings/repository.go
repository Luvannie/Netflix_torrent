package settings

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) List(ctx context.Context) ([]StorageProfile, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, base_path, priority, active
		FROM storage_profiles
		ORDER BY priority ASC, id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var profiles []StorageProfile
	for rows.Next() {
		var p StorageProfile
		if err := rows.Scan(&p.ID, &p.Name, &p.BasePath, &p.Priority, &p.Active); err != nil {
			return nil, err
		}
		profiles = append(profiles, p)
	}
	return profiles, rows.Err()
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*StorageProfile, error) {
	var p StorageProfile
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, base_path, priority, active
		FROM storage_profiles
		WHERE id=$1
	`, id).Scan(&p.ID, &p.Name, &p.BasePath, &p.Priority, &p.Active)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *Repository) Create(ctx context.Context, req CreateStorageProfileRequest) (*StorageProfile, error) {
	priority := 0
	if req.Priority != nil {
		priority = *req.Priority
	}
	active := true
	if req.Active != nil {
		active = *req.Active
	}

	var p StorageProfile
	err := r.pool.QueryRow(ctx, `
		INSERT INTO storage_profiles (name, base_path, priority, active)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, base_path, priority, active
	`, req.Name, req.BasePath, priority, active).Scan(&p.ID, &p.Name, &p.BasePath, &p.Priority, &p.Active)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *Repository) Update(ctx context.Context, id int64, req UpdateStorageProfileRequest) (*StorageProfile, error) {
	existing, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	name := existing.Name
	if req.Name != nil && *req.Name != "" {
		name = *req.Name
	} else if req.Name != nil && *req.Name == "" {
		return nil, ValidationError{Field: "name", Message: "Name must not be blank when provided"}
	}

	basePath := existing.BasePath
	if req.BasePath != nil && *req.BasePath != "" {
		basePath = *req.BasePath
	} else if req.BasePath != nil && *req.BasePath == "" {
		return nil, ValidationError{Field: "basePath", Message: "Base path must not be blank when provided"}
	}

	priority := existing.Priority
	if req.Priority != nil {
		if *req.Priority < 0 {
			return nil, ValidationError{Field: "priority", Message: "Priority must be zero or positive"}
		}
		priority = *req.Priority
	}

	active := existing.Active
	if req.Active != nil {
		active = *req.Active
	}

	var p StorageProfile
	err = r.pool.QueryRow(ctx, `
		UPDATE storage_profiles
		SET name=$2, base_path=$3, priority=$4, active=$5
		WHERE id=$1
		RETURNING id, name, base_path, priority, active
	`, id, name, basePath, priority, active).Scan(&p.ID, &p.Name, &p.BasePath, &p.Priority, &p.Active)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	result, err := r.pool.Exec(ctx, `DELETE FROM storage_profiles WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return NotFoundError{ID: id}
	}
	return nil
}

func (r *Repository) Exists(ctx context.Context, id int64) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM storage_profiles WHERE id=$1)`, id).Scan(&exists)
	return exists, err
}

func (r *Repository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM storage_profiles`).Scan(&count)
	return count, err
}

var _ = fmt.Sprintf