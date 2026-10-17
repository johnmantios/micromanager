package db

import (
	"context"
	"database/sql"
	"github.com/johnmantios/micromanager/internal/config"
	"time"
)

func OpenDB(cfg config.DB) (*sql.DB, error) {
	db, err := sql.Open(cfg.Driver, cfg.DataSourceName)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		return nil, err
	}

	return db, nil
}
