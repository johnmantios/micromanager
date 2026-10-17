package repo

import (
	"context"
	"database/sql"
	"github.com/johnmantios/micromanager/internal/model"
	"github.com/pkg/errors"
	"time"
)

type IActivity interface {
	GetBacklitMetrics(ctx context.Context) (*model.Activity, error)
	GetMinutesPerApplication(ctx context.Context) ([]model.ApplicationUsage, error)
}

type ActivityRepo struct {
	DB *sql.DB
}

func (m ActivityRepo) GetBacklitMetrics(ctx context.Context) (*model.Activity, error) {
	query := `SELECT
				  datetime(ZSTARTDATE + 978307200, 'unixepoch', 'localtime') AS day,
				  ROUND(SUM(ZENDDATE - ZSTARTDATE) * 60 / 3600.0, 1) AS minutes_on
				FROM ZOBJECT
				WHERE
				  ZSTREAMNAME = '/display/isBacklit'
				  AND ZVALUEINTEGER = 1
				  AND date(ZSTARTDATE + 978307200, 'unixepoch', 'localtime') = date('now', 'localtime')
				GROUP BY day;`

	var activity model.Activity
	var dateStr string

	cancelCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	row := m.DB.QueryRowContext(cancelCtx, query)

	err := row.Scan(
		&dateStr,
		&activity.MinutesOn,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, errors.New("No activity records found")
		}
		return nil, err
	}

	activity.Date, err = time.ParseInLocation(time.DateTime, dateStr, time.Local)
	if err != nil {
		return nil, err
	}

	return &activity, nil
}

func (m ActivityRepo) GetMinutesPerApplication(ctx context.Context) ([]model.ApplicationUsage, error) {
	query := `SELECT
				  ZOBJECT.ZVALUESTRING AS bundle_id,
				  ROUND(SUM(ZOBJECT.ZENDDATE - ZOBJECT.ZSTARTDATE) * 60 / 3600.0, 1) AS minutes_on
				FROM ZOBJECT
				WHERE
				  ZSTREAMNAME = '/app/usage'
				  AND ZVALUESTRING IS NOT NULL
				  AND date(ZOBJECT.ZSTARTDATE + 978307200, 'unixepoch', 'localtime') = date('now', 'localtime')
				GROUP BY ZOBJECT.ZVALUESTRING
				ORDER BY minutes_on DESC;`

	cancelCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	rows, err := m.DB.QueryContext(cancelCtx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applications := make([]model.ApplicationUsage, 0)
	for rows.Next() {
		var application model.ApplicationUsage
		if err := rows.Scan(
			&application.BundleID,
			&application.MinutesOn,
		); err != nil {
			return nil, err
		}
		applications = append(applications, application)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return applications, nil
}

func NewActivityRepo(db *sql.DB) (*ActivityRepo, error) {
	return &ActivityRepo{
		DB: db,
	}, nil
}
