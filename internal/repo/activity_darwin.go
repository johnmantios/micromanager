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

func NewActivityRepo(db *sql.DB) (*ActivityRepo, error) {
	return &ActivityRepo{
		DB: db,
	}, nil
}
