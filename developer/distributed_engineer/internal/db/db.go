// Package db owns the Postgres connection pool and schema migrations.
package db

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"time"

	"github.com/golang-migrate/migrate/v4"
	migratepgx "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect opens a pgx connection pool and verifies it with a ping.
func Connect(ctx context.Context, url string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parse db url: %w", err)
	}
	cfg.MaxConns = 10
	cfg.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return pool, nil
}

// Migrate applies all up migrations from the embedded migrations filesystem.
// It is safe to call on every boot — already-applied migrations are skipped.
func Migrate(url string, migrationFS fs.FS) error {
	src, err := iofs.New(migrationFS, ".")
	if err != nil {
		return fmt.Errorf("open migration source: %w", err)
	}

	// golang-migrate wants a standard pgx/database/sql connection string.
	m, err := migrate.NewWithSourceInstance("iofs", src, "pgx5://"+stripScheme(url))
	if err != nil {
		return fmt.Errorf("init migrator: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}

// stripScheme turns "postgres://..." / "postgresql://..." into the bare
// "user@host/db?..." form so we can prefix the migrate-specific "pgx5://".
func stripScheme(url string) string {
	for _, p := range []string{"postgres://", "postgresql://"} {
		if len(url) >= len(p) && url[:len(p)] == p {
			return url[len(p):]
		}
	}
	return url
}

// ensure the driver is registered.
var _ = migratepgx.Postgres{}
