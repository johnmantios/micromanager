package db

import (
	"context"
	"database/sql"
	"errors"
	"github.com/johnmantios/micromanager/pkg/config"
	"os"
	"time"

	_ "github.com/lib/pq"
)

func OpenDB(cfg config.EventsDB) (*sql.DB, error) {
	present := false
	dsn, present := os.LookupEnv("DATABASE_DSN")
	if !present {
		return nil, errors.New("env variable DATABASE_DSN missing")
	}

	db, err := sql.Open(cfg.Driver, dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)

	db.SetConnMaxIdleTime(cfg.MaxConnLifetime)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		return nil, err
	}

	return db, nil
}
