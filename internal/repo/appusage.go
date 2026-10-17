package repo

import (
	"context"
	"database/sql"
	"time"

	"github.com/johnmantios/micromanager/internal/model"
)

type IApplicationUsage interface {
	InsertApplicationUsage(ctx context.Context, userID, date string, applications []model.ApplicationUsage) error
}

type ApplicationUsageRepo struct {
	DB *sql.DB
}

func (m ApplicationUsageRepo) InsertApplicationUsage(ctx context.Context, userID, date string, applications []model.ApplicationUsage) error {
	if len(applications) == 0 {
		return nil
	}

	query := `INSERT INTO micromanager.application_usage (user_id, date, bundle_id, minutes_on)
    			VALUES ($1, $2, $3, $4);`

	cancelCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	tx, err := m.DB.BeginTx(cancelCtx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	statement, err := tx.PrepareContext(cancelCtx, query)
	if err != nil {
		return err
	}
	defer statement.Close()

	for _, application := range applications {
		if _, err := statement.ExecContext(cancelCtx, userID, date, application.BundleID, application.MinutesOn); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func NewApplicationUsageRepo(db *sql.DB) (*ApplicationUsageRepo, error) {
	return &ApplicationUsageRepo{
		DB: db,
	}, nil
}
