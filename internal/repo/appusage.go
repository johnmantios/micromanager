package repo

import (
	"context"
	"database/sql"
	"time"

	"github.com/johnmantios/micromanager/internal/model"
)

type IApplicationUsage interface {
	InsertApplicationUsage(ctx context.Context, usage *model.AppUsage) error
}

type ApplicationUsageRepo struct {
	DB *sql.DB
}

// InsertApplicationUsage upserts the per-application totals for a user/day.
// Each poll reports the cumulative total for the day, so an existing row is
// updated rather than duplicated.
func (m ApplicationUsageRepo) InsertApplicationUsage(ctx context.Context, usage *model.AppUsage) error {
	if len(usage.Applications) == 0 {
		return nil
	}

	query := `INSERT INTO micromanager.application_usage (user_id, username, date, bundle_id, minutes_on)
    			VALUES ($1, $2, $3, $4, $5)
				ON CONFLICT (user_id, date, bundle_id)
				DO UPDATE SET username = EXCLUDED.username, minutes_on = EXCLUDED.minutes_on;`

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

	for _, application := range usage.Applications {
		if _, err := statement.ExecContext(cancelCtx, usage.UserID, usage.Username, usage.Date, application.BundleID, application.MinutesOn); err != nil {
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
