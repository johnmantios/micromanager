//go:build unit && darwin

package repo_test

import (
	"context"
	"database/sql"
	"reflect"
	"testing"
	"time"

	"github.com/johnmantios/micromanager/internal/model"
	"github.com/johnmantios/micromanager/internal/repo"
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

	now := time.Now()
	noon := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, time.Local)
	macNoon := float64(noon.Unix() - 978307200)

	for _, row := range rows {
		if _, err := db.Exec(
			`INSERT INTO ZOBJECT (ZSTREAMNAME, ZVALUESTRING, ZVALUEINTEGER, ZSTARTDATE, ZENDDATE) VALUES (?, ?, ?, ?, ?)`,
			row.stream,
			row.bundleID,
			row.valueInteger,
			macNoon+row.startOffset.Seconds(),
			macNoon+row.endOffset.Seconds(),
		); err != nil {
			t.Fatalf("insert ZOBJECT row: %v", err)
		}
	}

	return db
}

// TestActivityRepoGetMinutesPerApplication verifies the /app/usage aggregation,
// filtering and ordering against a SQLite engine using a ZOBJECT fixture.
func TestActivityRepoGetMinutesPerApplication(t *testing.T) {
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

			activityRepo, err := repo.NewActivityRepo(db)
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
