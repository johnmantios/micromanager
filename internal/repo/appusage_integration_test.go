//go:build integration

package repo_test

import (
	"context"
	"io"
	"reflect"
	"testing"

	"github.com/johnmantios/micromanager/internal/jsonlog"
	"github.com/johnmantios/micromanager/internal/model"
	"github.com/johnmantios/micromanager/internal/repo"
	"github.com/johnmantios/micromanager/internal/server/input"
	"github.com/johnmantios/micromanager/internal/service"
)

type storedApplicationUsage struct {
	UserID    string
	Username  string
	Date      string
	BundleID  string
	MinutesOn float64
}

func resetApplicationUsage(t *testing.T) {
	t.Helper()

	if _, err := testPostgres.DB.Exec("TRUNCATE micromanager.application_usage RESTART IDENTITY"); err != nil {
		t.Fatalf("truncate application_usage: %v", err)
	}
}

func assertApplicationUsageRowCount(t *testing.T, want int) {
	t.Helper()

	var got int
	if err := testPostgres.DB.QueryRow("SELECT COUNT(*) FROM micromanager.application_usage").Scan(&got); err != nil {
		t.Fatalf("count application usage rows: %v", err)
	}
	if got != want {
		t.Fatalf("application usage row count: got %d, want %d", got, want)
	}
}

func readApplicationUsage(t *testing.T) []storedApplicationUsage {
	t.Helper()

	rows, err := testPostgres.DB.Query(
		"SELECT user_id, username, date::text, bundle_id, minutes_on FROM micromanager.application_usage ORDER BY bundle_id",
	)
	if err != nil {
		t.Fatalf("query application usage: %v", err)
	}
	defer rows.Close()

	stored := make([]storedApplicationUsage, 0)
	for rows.Next() {
		var row storedApplicationUsage
		if err := rows.Scan(&row.UserID, &row.Username, &row.Date, &row.BundleID, &row.MinutesOn); err != nil {
			t.Fatalf("scan application usage: %v", err)
		}
		stored = append(stored, row)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate application usage: %v", err)
	}

	return stored
}

// TestApplicationUsageRepoInsertApplicationUsage verifies the batch persistence
// of application usage against a real database, including the empty and
// rollback paths.
func TestApplicationUsageRepoInsertApplicationUsage(t *testing.T) {
	applicationUsageRepo, err := repo.NewApplicationUsageRepo(testPostgres.DB)
	if err != nil {
		t.Fatalf("new application usage repo: %v", err)
	}

	tests := []struct {
		name         string
		userID       string
		username     string
		date         string
		applications []model.ApplicationUsage
		wantErr      bool
		wantRows     int
		wantStored   []storedApplicationUsage
	}{
		{
			name:     "persists multiple applications",
			userID:   "user-1",
			username: "jane",
			date:     "2024-05-01",
			applications: []model.ApplicationUsage{
				{BundleID: "com.apple.Safari", MinutesOn: 10},
				{BundleID: "com.microsoft.VSCode", MinutesOn: 3.5},
				{BundleID: "com.tinyspeck.slackmacgap", MinutesOn: 1},
			},
			wantRows: 3,
			wantStored: []storedApplicationUsage{
				{UserID: "user-1", Username: "jane", Date: "2024-05-01", BundleID: "com.apple.Safari", MinutesOn: 10},
				{UserID: "user-1", Username: "jane", Date: "2024-05-01", BundleID: "com.microsoft.VSCode", MinutesOn: 3.5},
				{UserID: "user-1", Username: "jane", Date: "2024-05-01", BundleID: "com.tinyspeck.slackmacgap", MinutesOn: 1},
			},
		},
		{
			name:         "does nothing for an empty application list",
			userID:       "user-2",
			username:     "amy",
			date:         "2024-05-02",
			applications: []model.ApplicationUsage{},
			wantRows:     0,
		},
		{
			name:     "rolls back on an invalid date",
			userID:   "user-3",
			username: "john",
			date:     "not-a-date",
			applications: []model.ApplicationUsage{
				{BundleID: "com.apple.Safari", MinutesOn: 5},
			},
			wantErr:  true,
			wantRows: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetApplicationUsage(t)

			err := applicationUsageRepo.InsertApplicationUsage(context.Background(), &model.AppUsage{
				UserID:       tt.userID,
				Username:     tt.username,
				Date:         tt.date,
				Applications: tt.applications,
			})
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			assertApplicationUsageRowCount(t, tt.wantRows)

			if tt.wantStored == nil {
				return
			}

			stored := readApplicationUsage(t)
			if len(stored) != len(tt.wantStored) {
				t.Fatalf("stored rows: got %+v, want %+v", stored, tt.wantStored)
			}
			for i := range tt.wantStored {
				if stored[i] != tt.wantStored[i] {
					t.Errorf("stored row %d: got %+v, want %+v", i, stored[i], tt.wantStored[i])
				}
			}
		})
	}
}

// TestApplicationUsageRepoUpsert verifies that re-reporting the cumulative
// totals for the same user/day/app updates the existing row instead of
// inserting a duplicate.
func TestApplicationUsageRepoUpsert(t *testing.T) {
	applicationUsageRepo, err := repo.NewApplicationUsageRepo(testPostgres.DB)
	if err != nil {
		t.Fatalf("new application usage repo: %v", err)
	}

	resetApplicationUsage(t)
	ctx := context.Background()

	first := []model.ApplicationUsage{
		{BundleID: "com.apple.Safari", MinutesOn: 10},
		{BundleID: "com.microsoft.VSCode", MinutesOn: 3},
	}
	if err := applicationUsageRepo.InsertApplicationUsage(ctx, &model.AppUsage{UserID: "user-1", Username: "jane", Date: "2024-05-01", Applications: first}); err != nil {
		t.Fatalf("first insert: %v", err)
	}

	second := []model.ApplicationUsage{
		{BundleID: "com.apple.Safari", MinutesOn: 25},
		{BundleID: "com.microsoft.VSCode", MinutesOn: 8},
		{BundleID: "com.tinyspeck.slackmacgap", MinutesOn: 2},
	}
	if err := applicationUsageRepo.InsertApplicationUsage(ctx, &model.AppUsage{UserID: "user-1", Username: "jane", Date: "2024-05-01", Applications: second}); err != nil {
		t.Fatalf("second insert: %v", err)
	}

	assertApplicationUsageRowCount(t, 3)

	stored := readApplicationUsage(t)
	want := []storedApplicationUsage{
		{UserID: "user-1", Username: "jane", Date: "2024-05-01", BundleID: "com.apple.Safari", MinutesOn: 25},
		{UserID: "user-1", Username: "jane", Date: "2024-05-01", BundleID: "com.microsoft.VSCode", MinutesOn: 8},
		{UserID: "user-1", Username: "jane", Date: "2024-05-01", BundleID: "com.tinyspeck.slackmacgap", MinutesOn: 2},
	}
	if !reflect.DeepEqual(stored, want) {
		t.Fatalf("stored rows: got %+v, want %+v", stored, want)
	}
}

// TestApplicationUsageServiceCreateAppUsage verifies the service-to-repository
// path against a real database.
func TestApplicationUsageServiceCreateAppUsage(t *testing.T) {
	applicationUsageRepo, err := repo.NewApplicationUsageRepo(testPostgres.DB)
	if err != nil {
		t.Fatalf("new application usage repo: %v", err)
	}

	applicationUsageService := service.NewApplicationUsageService(applicationUsageRepo, jsonlog.New(io.Discard, jsonlog.LevelError))

	tests := []struct {
		name     string
		input    func() *input.AppUsage
		wantErr  bool
		wantRows int
	}{
		{
			name: "creates application usage entries",
			input: func() *input.AppUsage {
				in := &input.AppUsage{}
				in.Body.UserID = "user-10"
				in.Body.Date = "2024-05-01"
				in.Body.Applications = []input.AppUsageItem{
					{BundleID: "com.apple.Safari", MinutesOn: 30},
					{BundleID: "com.microsoft.VSCode", MinutesOn: 15.5},
				}
				return in
			},
			wantRows: 2,
		},
		{
			name: "fails on an invalid date",
			input: func() *input.AppUsage {
				in := &input.AppUsage{}
				in.Body.UserID = "user-11"
				in.Body.Date = "not-a-date"
				in.Body.Applications = []input.AppUsageItem{
					{BundleID: "com.apple.Safari", MinutesOn: 5},
				}
				return in
			},
			wantErr:  true,
			wantRows: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetApplicationUsage(t)

			out, err := applicationUsageService.CreateAppUsageHandler(context.Background(), tt.input())
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if out.Body.Message != "ok" {
					t.Errorf("message: got %q, want %q", out.Body.Message, "ok")
				}
			}

			assertApplicationUsageRowCount(t, tt.wantRows)
		})
	}
}
