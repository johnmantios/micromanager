package repo

import (
	"context"
	"database/sql"
	"time"

	"github.com/johnmantios/micromanager/internal/model"
	"github.com/pkg/errors"
)

// cocoaEpochOffset is the number of seconds between 1970-01-01 and 2001-01-01,
// the epoch used by the Knowledge store's ZOBJECT timestamps.
const cocoaEpochOffset = 978307200

type IActivity interface {
	GetBacklitMetrics(ctx context.Context) (*model.Activity, error)
	GetMinutesPerApplication(ctx context.Context) ([]model.ApplicationUsage, error)
}

// nowFunc returns the current time and can be overridden in tests.
var nowFunc = time.Now

type ActivityRepo struct {
	DB *sql.DB
}

// todayWindow returns the bounds of the current local day expressed in
// Knowledge/Cocoa epoch seconds, clamped to now so in-progress intervals are
// not over-counted.
func todayWindow(now time.Time) (start, end float64) {
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return float64(startOfDay.Unix() - cocoaEpochOffset), float64(now.Unix() - cocoaEpochOffset)
}

func (m ActivityRepo) GetBacklitMetrics(ctx context.Context) (*model.Activity, error) {
	now := nowFunc()
	startCocoa, nowCocoa := todayWindow(now)

	// Aggregate over backlit-on intervals that overlap today, clamping each
	// interval to the day's window. The aggregate always returns a row (0 when
	// there is no activity yet today), so an absence of data is not an error.
	query := `SELECT
				  COALESCE(ROUND(SUM((MIN(ZENDDATE, ?) - MAX(ZSTARTDATE, ?)) * 60 / 3600.0), 1), 0) AS minutes_on
				FROM ZOBJECT
				WHERE
				  ZSTREAMNAME = '/display/isBacklit'
				  AND ZVALUEINTEGER = 1
				  AND ZENDDATE > ?
				  AND ZSTARTDATE <= ?;`

	var minutesOn float64

	cancelCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(cancelCtx, query, nowCocoa, startCocoa, startCocoa, nowCocoa).Scan(&minutesOn)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	return &model.Activity{
		Date:      time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local),
		MinutesOn: minutesOn,
	}, nil
}

func (m ActivityRepo) GetMinutesPerApplication(ctx context.Context) ([]model.ApplicationUsage, error) {
	now := nowFunc()
	startCocoa, nowCocoa := todayWindow(now)

	// Clamp each usage interval to today's window so an interval that started
	// before midnight or is still running contributes only its overlap.
	query := `SELECT
				  ZOBJECT.ZVALUESTRING AS bundle_id,
				  ROUND(SUM((MIN(ZENDDATE, ?) - MAX(ZSTARTDATE, ?)) * 60 / 3600.0), 1) AS minutes_on
				FROM ZOBJECT
				WHERE
				  ZSTREAMNAME = '/app/usage'
				  AND ZVALUESTRING IS NOT NULL
				  AND ZENDDATE > ?
				  AND ZSTARTDATE <= ?
				GROUP BY ZOBJECT.ZVALUESTRING
				ORDER BY minutes_on DESC;`

	cancelCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	rows, err := m.DB.QueryContext(cancelCtx, query, nowCocoa, startCocoa, startCocoa, nowCocoa)
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
