//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/johnmantios/micromanager/internal/jsonlog"
	"github.com/johnmantios/micromanager/internal/repo"
	"github.com/johnmantios/micromanager/internal/server"
	"github.com/johnmantios/micromanager/internal/service"
	"github.com/johnmantios/micromanager/internal/testutil"
)

var (
	testPostgres *testutil.Postgres
	apiBaseURL   string
)

// TestMain boots a disposable database and the fully wired API server for all end-to-end tests.
func TestMain(m *testing.M) {
	ctx := context.Background()

	var err error
	testPostgres, err = testutil.StartPostgres(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "start postgres: %v\n", err)
		os.Exit(1)
	}

	screentimeRepo, err := repo.NewScreentimeRepo(testPostgres.DB)
	if err != nil {
		fmt.Fprintf(os.Stderr, "new screentime repo: %v\n", err)
		_ = testPostgres.Close(ctx)
		os.Exit(1)
	}

	log := jsonlog.New(io.Discard, jsonlog.LevelError)
	httpServer := httptest.NewServer(server.NewHandler(service.NewScreentimeService(screentimeRepo, log), log))
	apiBaseURL = httpServer.URL

	code := m.Run()

	httpServer.Close()
	if err := testPostgres.Close(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "close postgres: %v\n", err)
	}
	os.Exit(code)
}

type storedScreentime struct {
	UserID    string
	Username  string
	MinutesOn int
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

func assertStored(t *testing.T, want *storedScreentime) {
	t.Helper()

	var got storedScreentime
	row := testPostgres.DB.QueryRow(
		"SELECT user_id, username, minutes_on FROM micromanager.screentime WHERE user_id = $1",
		want.UserID,
	)
	if err := row.Scan(&got.UserID, &got.Username, &got.MinutesOn); err != nil {
		t.Fatalf("scan stored row: %v", err)
	}
	if got != *want {
		t.Fatalf("stored row: got %+v, want %+v", got, want)
	}
}

// TestAPIEndpoints drives the running API over HTTP and verifies both responses and persisted state.
func TestAPIEndpoints(t *testing.T) {
	const validPayload = `{"user_id":"user-1","username":"jane","date":"2024-05-01","minutes_on":42}`

	tests := []struct {
		name             string
		method           string
		path             string
		body             string
		contentType      string
		wantStatus       int
		wantBodyContains []string
		wantRows         int
		wantStored       *storedScreentime
	}{
		{
			name:             "healthcheck returns 200",
			method:           http.MethodGet,
			path:             "/v1/healthcheck",
			wantStatus:       http.StatusOK,
			wantBodyContains: []string{"micromanager"},
			wantRows:         0,
		},
		{
			name:             "create screentime persists and returns 201",
			method:           http.MethodPost,
			path:             "/v1/screentime",
			body:             validPayload,
			contentType:      "application/json",
			wantStatus:       http.StatusCreated,
			wantBodyContains: []string{`"message":"ok"`},
			wantRows:         1,
			wantStored:       &storedScreentime{UserID: "user-1", Username: "jane", MinutesOn: 42},
		},
		{
			name:             "create screentime rejects invalid date with 500",
			method:           http.MethodPost,
			path:             "/v1/screentime",
			body:             `{"user_id":"user-2","username":"john","date":"not-a-date","minutes_on":5}`,
			contentType:      "application/json",
			wantStatus:       http.StatusInternalServerError,
			wantBodyContains: []string{`"status":500`},
			wantRows:         0,
		},
		{
			name:             "create screentime rejects malformed JSON with 400",
			method:           http.MethodPost,
			path:             "/v1/screentime",
			body:             `{"user_id":`,
			contentType:      "application/json",
			wantStatus:       http.StatusBadRequest,
			wantBodyContains: []string{`"status":400`},
			wantRows:         0,
		},
		{
			name:             "create screentime rejects wrong field type with 422",
			method:           http.MethodPost,
			path:             "/v1/screentime",
			body:             `{"user_id":"user-3","username":"amy","date":"2024-05-01","minutes_on":"many"}`,
			contentType:      "application/json",
			wantStatus:       http.StatusUnprocessableEntity,
			wantBodyContains: []string{`"status":422`},
			wantRows:         0,
		},
		{
			name:       "create screentime rejects unsupported method with 405",
			method:     http.MethodGet,
			path:       "/v1/screentime",
			wantStatus: http.StatusMethodNotAllowed,
			wantRows:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetScreentime(t)

			var body io.Reader
			if tt.body != "" {
				body = strings.NewReader(tt.body)
			}

			req, err := http.NewRequest(tt.method, apiBaseURL+tt.path, body)
			if err != nil {
				t.Fatalf("build request: %v", err)
			}
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("perform request: %v", err)
			}
			defer resp.Body.Close()

			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("read response body: %v", err)
			}

			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("status: got %d, want %d (body: %s)", resp.StatusCode, tt.wantStatus, respBody)
			}
			for _, want := range tt.wantBodyContains {
				if !strings.Contains(string(respBody), want) {
					t.Errorf("response body %q does not contain %q", respBody, want)
				}
			}

			assertRowCount(t, tt.wantRows)
			if tt.wantStored != nil {
				assertStored(t, tt.wantStored)
			}
		})
	}
}
