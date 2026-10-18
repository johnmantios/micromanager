//go:build unit && darwin

package repo

import (
	"context"
	"database/sql"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/johnmantios/micromanager/internal/model"
	_ "github.com/mattn/go-sqlite3"
)

type zobjectRow struct {
	stream       string
	bundleID     *string
	valueInteger int64
	startOffset  time.Duration
	endOffset    time.Duration
}

func strPtr(s string) *string {
	return &s
}

// withFixedNow pins the package clock for the duration of a test so the
// day-window clamping in the queries is deterministic.
func withFixedNow(t *testing.T, now time.Time) {
	t.Helper()

	original := nowFunc
	nowFunc = func() time.Time { return now }
	t.Cleanup(func() { nowFunc = original })
}

func newZObjectDB(t *testing.T, rows []zobjectRow) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.Exec(`CREATE TABLE ZOBJECT (
		Z_PK INTEGER PRIMARY KEY,
		ZSTREAMNAME TEXT,
		ZVALUESTRING TEXT,
		ZVALUEINTEGER INTEGER,
		ZSTARTDATE REAL,
		ZENDDATE REAL
	)`); err != nil {
		t.Fatalf("create ZOBJECT: %v", err)
	}

	base := nowFunc()
	macBase := float64(base.Unix() - cocoaEpochOffset)

	for _, row := range rows {
		if _, err := db.Exec(
			`INSERT INTO ZOBJECT (ZSTREAMNAME, ZVALUESTRING, ZVALUEINTEGER, ZSTARTDATE, ZENDDATE) VALUES (?, ?, ?, ?, ?)`,
			row.stream,
			row.bundleID,
			row.valueInteger,
			macBase+row.startOffset.Seconds(),
			macBase+row.endOffset.Seconds(),
		); err != nil {
			t.Fatalf("insert ZOBJECT row: %v", err)
		}
	}

	return db
}

// TestActivityRepoGetMinutesPerApplication verifies the /app/usage aggregation,
// filtering and ordering against a SQLite engine using a ZOBJECT fixture.
func TestActivityRepoGetMinutesPerApplication(t *testing.T) {
	withFixedNow(t, time.Date(2026, 1, 2, 15, 0, 0, 0, time.Local))

	tests := []struct {
		name string
		rows []zobjectRow
		want []model.ApplicationUsage
	}{
		{
			name: "aggregates, ranks and filters usage for today",
			rows: []zobjectRow{
				{stream: "/app/usage", bundleID: strPtr("com.apple.Safari"), startOffset: -10 * time.Minute, endOffset: -5 * time.Minute},
				{stream: "/app/usage", bundleID: strPtr("com.apple.Safari"), startOffset: -5 * time.Minute, endOffset: 0},
				{stream: "/app/usage", bundleID: strPtr("com.microsoft.VSCode"), startOffset: -3 * time.Minute, endOffset: 0},
				{stream: "/app/usage", bundleID: strPtr("com.tinyspeck.slackmacgap"), startOffset: -time.Minute, endOffset: 0},
				{stream: "/display/isBacklit", valueInteger: 1, startOffset: -time.Hour, endOffset: 0},
				{stream: "/app/usage", startOffset: -2 * time.Minute, endOffset: 0},
				{stream: "/app/usage", bundleID: strPtr("com.yesterday.App"), startOffset: -36 * time.Hour, endOffset: -36*time.Hour + 2*time.Minute},
			},
			want: []model.ApplicationUsage{
				{BundleID: "com.apple.Safari", MinutesOn: 10},
				{BundleID: "com.microsoft.VSCode", MinutesOn: 3},
				{BundleID: "com.tinyspeck.slackmacgap", MinutesOn: 1},
			},
		},
		{
			name: "returns an empty slice when there is no usage today",
			rows: []zobjectRow{
				{stream: "/display/isBacklit", valueInteger: 1, startOffset: -time.Hour, endOffset: 0},
			},
			want: []model.ApplicationUsage{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newZObjectDB(t, tt.rows)

			activityRepo, err := NewActivityRepo(db)
			if err != nil {
				t.Fatalf("new activity repo: %v", err)
			}

			got, err := activityRepo.GetMinutesPerApplication(context.Background())
			if err != nil {
				t.Fatalf("get minutes per application: %v", err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

// TestActivityRepoGetBacklitMetrics verifies the backlit aggregation, including
// clamping intervals to the current day and returning zero (rather than an
// error) when there is no activity yet today.
func TestActivityRepoGetBacklitMetrics(t *testing.T) {
	withFixedNow(t, time.Date(2026, 1, 2, 15, 0, 0, 0, time.Local))

	tests := []struct {
		name string
		rows []zobjectRow
		want float64
	}{
		{
			name: "sums the backlit-on intervals overlapping today",
			rows: []zobjectRow{
				{stream: "/display/isBacklit", valueInteger: 1, startOffset: -30 * time.Minute, endOffset: -10 * time.Minute},
				{stream: "/display/isBacklit", valueInteger: 1, startOffset: -5 * time.Minute, endOffset: 0},
				{stream: "/display/isBacklit", valueInteger: 0, startOffset: -10 * time.Minute, endOffset: -5 * time.Minute},
				{stream: "/display/isBacklit", valueInteger: 1, startOffset: -36 * time.Hour, endOffset: -35 * time.Hour},
			},
			want: 25,
		},
		{
			name: "clamps an interval that is still running",
			rows: []zobjectRow{
				{stream: "/display/isBacklit", valueInteger: 1, startOffset: -20 * time.Minute, endOffset: 5 * time.Minute},
			},
			want: 20,
		},
		{
			name: "returns zero when there is no backlit activity today",
			rows: []zobjectRow{
				{stream: "/display/isBacklit", valueInteger: 0, startOffset: -time.Hour, endOffset: 0},
				{stream: "/display/isBacklit", valueInteger: 1, startOffset: -36 * time.Hour, endOffset: -35 * time.Hour},
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newZObjectDB(t, tt.rows)

			activityRepo, err := NewActivityRepo(db)
			if err != nil {
				t.Fatalf("new activity repo: %v", err)
			}

			got, err := activityRepo.GetBacklitMetrics(context.Background())
			if err != nil {
				t.Fatalf("get backlit metrics: %v", err)
			}

			if got.Date.Format("2006-01-02") != nowFunc().Format("2006-01-02") {
				t.Errorf("date: got %s, want today", got.Date.Format("2006-01-02"))
			}
			if diff := math.Abs(got.MinutesOn - tt.want); diff > 0.05 {
				t.Errorf("minutes_on: got %v, want %v", got.MinutesOn, tt.want)
			}
		})
	}
}
