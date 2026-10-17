//go:build integration || e2e

package testutil

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"
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

// MigrationScripts returns the ordered up-migration scripts for the project.
func MigrationScripts() []string {
	root := RepoRoot()
	return []string{
		filepath.Join(root, "migrations", "000001_create_micromanager_schema.up.sql"),
		filepath.Join(root, "migrations", "000002_create_screentime_table.up.sql"),
	}
}

// StartPostgres starts a TimescaleDB container, applies the migrations and returns a ready connection.
func StartPostgres(ctx context.Context) (*Postgres, error) {
	container, err := postgres.Run(ctx,
		postgresImage,
		postgres.WithDatabase("micromanager"),
		postgres.WithUsername("root"),
		postgres.WithPassword("root"),
		postgres.WithOrderedInitScripts(MigrationScripts()...),
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

// StartPostgresT starts a Postgres container for the duration of a test.
func StartPostgresT(t *testing.T) *Postgres {
	t.Helper()

	pg, err := StartPostgres(context.Background())
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}

	t.Cleanup(func() {
		if err := pg.Close(context.Background()); err != nil {
			t.Errorf("close postgres: %v", err)
		}
	})

	return pg
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
