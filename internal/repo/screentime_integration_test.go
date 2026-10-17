//go:build integration

package repo_test

import (
	"context"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/johnmantios/micromanager/internal/jsonlog"
	"github.com/johnmantios/micromanager/internal/model"
	"github.com/johnmantios/micromanager/internal/repo"
	"github.com/johnmantios/micromanager/internal/server/input"
	"github.com/johnmantios/micromanager/internal/service"
	"github.com/johnmantios/micromanager/internal/testutil"
)

var testPostgres *testutil.Postgres

// TestMain boots a single disposable database for all integration tests in this package.
func TestMain(m *testing.M) {
	ctx := context.Background()

	var err error
	testPostgres, err = testutil.StartPostgres(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "start postgres: %v\n", err)
		os.Exit(1)
	}

	code := m.Run()

	if err := testPostgres.Close(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "close postgres: %v\n", err)
	}
	os.Exit(code)
}

func resetScreentime(t *testing.T) {
	t.Helper()

	if _, err := testPostgres.DB.Exec("TRUNCATE micromanager.screentime RESTART IDENTITY"); err != nil {
		t.Fatalf("truncate screentime: %v", err)
	}
}

func assertRowCount(t *testing.T, want int) {
	t.Helper()

	var got int
	if err := testPostgres.DB.QueryRow("SELECT COUNT(*) FROM micromanager.screentime").Scan(&got); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if got != want {
		t.Fatalf("row count: got %d, want %d", got, want)
	}
}

// TestScreentimeRepoInsertBacklitMetrics verifies persistence against a real database.
func TestScreentimeRepoInsertBacklitMetrics(t *testing.T) {
	screentimeRepo, err := repo.NewScreentimeRepo(testPostgres.DB)
	if err != nil {
		t.Fatalf("new screentime repo: %v", err)
	}

	tests := []struct {
		name          string
		input         *model.Screentime
		wantErr       bool
		wantRows      int
		wantUsername  string
		wantMinutesOn int
	}{
		{
			name:          "persists a valid record",
			input:         &model.Screentime{UserID: "user-1", Username: "jane", Date: "2024-05-01", MinutesOn: 42},
			wantRows:      1,
			wantUsername:  "jane",
			wantMinutesOn: 42,
		},
		{
			name:          "persists a zero minutes record",
			input:         &model.Screentime{UserID: "user-2", Username: "amy", Date: "2024-06-01", MinutesOn: 0},
			wantRows:      1,
			wantUsername:  "amy",
			wantMinutesOn: 0,
		},
		{
			name:     "rejects an invalid date",
			input:    &model.Screentime{UserID: "user-3", Username: "john", Date: "not-a-date", MinutesOn: 10},
			wantErr:  true,
			wantRows: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetScreentime(t)

			err := screentimeRepo.InsertBacklitMetrics(context.Background(), tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			assertRowCount(t, tt.wantRows)

			if tt.wantRows == 0 {
				return
			}

			var (
				username  string
				minutesOn int
			)
			row := testPostgres.DB.QueryRow(
				"SELECT username, minutes_on FROM micromanager.screentime WHERE user_id = $1",
				tt.input.UserID,
			)
			if err := row.Scan(&username, &minutesOn); err != nil {
				t.Fatalf("scan row: %v", err)
			}

			if username != tt.wantUsername {
				t.Errorf("username: got %q, want %q", username, tt.wantUsername)
			}
			if minutesOn != tt.wantMinutesOn {
				t.Errorf("minutes_on: got %d, want %d", minutesOn, tt.wantMinutesOn)
			}
		})
	}
}

// TestScreentimeServiceCreateScreentime verifies the service-to-repository path against a real database.
func TestScreentimeServiceCreateScreentime(t *testing.T) {
	screentimeRepo, err := repo.NewScreentimeRepo(testPostgres.DB)
	if err != nil {
		t.Fatalf("new screentime repo: %v", err)
	}

	screentimeService := service.NewScreentimeService(screentimeRepo, jsonlog.New(io.Discard, jsonlog.LevelError))

	tests := []struct {
		name     string
		input    func() *input.Screentime
		wantErr  bool
		wantRows int
	}{
		{
			name: "creates a screentime entry",
			input: func() *input.Screentime {
				in := &input.Screentime{}
				in.Body.UserID = "user-10"
				in.Body.Username = "jane"
				in.Body.Date = "2024-05-01"
				in.Body.MinutesOn = 30
				return in
			},
			wantRows: 1,
		},
		{
			name: "fails on an invalid date",
			input: func() *input.Screentime {
				in := &input.Screentime{}
				in.Body.UserID = "user-11"
				in.Body.Username = "john"
				in.Body.Date = "not-a-date"
				in.Body.MinutesOn = 30
				return in
			},
			wantErr:  true,
			wantRows: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetScreentime(t)

			out, err := screentimeService.CreateScreentimeHandler(context.Background(), tt.input())
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

			assertRowCount(t, tt.wantRows)
		})
	}
}
