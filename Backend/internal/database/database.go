package database

import (
	"context"
	"embed"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type Pool interface {
	Ping(ctx context.Context) error
	Close()
}

func Open(ctx context.Context, postgresURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, postgresURL)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	return pool, nil
}

func Ping(ctx context.Context, pool Pool) error {
	return pool.Ping(ctx)
}

func Close(pool Pool) {
	pool.Close()
}

func ApplyMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	if err := ensureMigrationTable(ctx, pool); err != nil {
		return err
	}

	names, err := MigrationNames()
	if err != nil {
		return fmt.Errorf("list migrations: %w", err)
	}

	for _, name := range names {
		applied, err := isMigrationApplied(ctx, pool, name)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		sql, err := ReadMigration(name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}

		if _, err := pool.Exec(ctx, sql); err != nil {
			return fmt.Errorf("apply migration %s: %w", name, err)
		}

		if err := recordMigration(ctx, pool, name); err != nil {
			return fmt.Errorf("record migration %s: %w", name, err)
		}
	}

	return nil
}

func MigrationNames() ([]string, error) {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return nil, fmt.Errorf("read migration dir: %w", err)
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".sql") {
			names = append(names, name)
		}
	}

	for i := 0; i < len(names)-1; i++ {
		for j := i + 1; j < len(names); j++ {
			if names[j] < names[i] {
				names[i], names[j] = names[j], names[i]
			}
		}
	}

	return names, nil
}

func ReadMigration(name string) (string, error) {
	data, err := migrationsFS.ReadFile("migrations/" + name)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}
	return string(data), nil
}

func ensureMigrationTable(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS go_schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		)
	`)
	return err
}

func isMigrationApplied(ctx context.Context, pool *pgxpool.Pool, name string) (bool, error) {
	var count int
	err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM go_schema_migrations WHERE version = $1", name).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func recordMigration(ctx context.Context, pool *pgxpool.Pool, name string) error {
	_, err := pool.Exec(ctx, "INSERT INTO go_schema_migrations (version, applied_at) VALUES ($1, $2)", name, time.Now())
	return err
}