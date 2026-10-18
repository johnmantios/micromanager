//go:build integration || e2e

package testutil

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"sort"
	"time"

	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

const postgresImage = "timescale/timescaledb:latest-pg17"

// Postgres bundles a disposable Postgres/TimescaleDB container with an open database handle.
type Postgres struct {
	DB        *sql.DB
	DSN       string
	container *postgres.PostgresContainer
}

// RepoRoot returns the absolute path to the repository root.
func RepoRoot() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

// MigrationScripts returns every up-migration script in the migrations
// directory, ordered by version. The version prefix in each filename keeps the
// lexical order equal to the migration order.
func MigrationScripts() ([]string, error) {
	dir := filepath.Join(RepoRoot(), "migrations")

	scripts, err := filepath.Glob(filepath.Join(dir, "*.up.sql"))
	if err != nil {
		return nil, fmt.Errorf("glob migrations in %s: %w", dir, err)
	}
	if len(scripts) == 0 {
		return nil, fmt.Errorf("no up-migration scripts found in %s", dir)
	}

	sort.Strings(scripts)

	return scripts, nil
}

// StartPostgres starts a TimescaleDB container, applies the migrations and returns a ready connection.
func StartPostgres(ctx context.Context) (*Postgres, error) {
	scripts, err := MigrationScripts()
	if err != nil {
		return nil, err
	}

	container, err := postgres.Run(ctx,
		postgresImage,
		postgres.WithDatabase("micromanager"),
		postgres.WithUsername("root"),
		postgres.WithPassword("root"),
		postgres.WithOrderedInitScripts(scripts...),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		return nil, fmt.Errorf("start postgres container: %w", err)
	}

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("build connection string: %w", err)
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("open database: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &Postgres{DB: db, DSN: dsn, container: container}, nil
}

// Close releases the database connection and terminates the container.
func (p *Postgres) Close(ctx context.Context) error {
	var errs []error

	if err := p.DB.Close(); err != nil {
		errs = append(errs, err)
	}
	if err := p.container.Terminate(ctx); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}
