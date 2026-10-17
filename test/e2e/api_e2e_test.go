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
	"time"

	"github.com/johnmantios/micromanager/internal/jsonlog"
	"github.com/johnmantios/micromanager/internal/model"
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

	applicationUsageRepo, err := repo.NewApplicationUsageRepo(testPostgres.DB)
	if err != nil {
		fmt.Fprintf(os.Stderr, "new application usage repo: %v\n", err)
		_ = testPostgres.Close(ctx)
		os.Exit(1)
	}

	log := jsonlog.New(io.Discard, jsonlog.LevelError)
	screentimeService := service.NewScreentimeService(screentimeRepo, log)
	applicationUsageService := service.NewApplicationUsageService(applicationUsageRepo, log)

	httpServer := httptest.NewServer(server.NewHandler(screentimeService, applicationUsageService, log))
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

type storedApplicationUsage struct {
	UserID    string
	Date      string
	BundleID  string
	MinutesOn float64
}

func resetTables(t *testing.T) {
	t.Helper()

	if _, err := testPostgres.DB.Exec("TRUNCATE micromanager.screentime RESTART IDENTITY"); err != nil {
		t.Fatalf("truncate screentime: %v", err)
	}
	if _, err := testPostgres.DB.Exec("TRUNCATE micromanager.application_usage RESTART IDENTITY"); err != nil {
		t.Fatalf("truncate application_usage: %v", err)
	}
}

func assertScreentimeRowCount(t *testing.T, want int) {
	t.Helper()

	var got int
	if err := testPostgres.DB.QueryRow("SELECT COUNT(*) FROM micromanager.screentime").Scan(&got); err != nil {
		t.Fatalf("count screentime rows: %v", err)
	}
	if got != want {
		t.Fatalf("screentime row count: got %d, want %d", got, want)
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

func assertStoredScreentime(t *testing.T, want *storedScreentime) {
	t.Helper()

	var got storedScreentime
	row := testPostgres.DB.QueryRow(
		"SELECT user_id, username, minutes_on FROM micromanager.screentime WHERE user_id = $1",
		want.UserID,
	)
	if err := row.Scan(&got.UserID, &got.Username, &got.MinutesOn); err != nil {
		t.Fatalf("scan stored screentime: %v", err)
	}
	if got != *want {
		t.Fatalf("stored screentime: got %+v, want %+v", got, want)
	}
}

func readApplicationUsage(t *testing.T) []storedApplicationUsage {
	t.Helper()

	rows, err := testPostgres.DB.Query(
		"SELECT user_id, date::text, bundle_id, minutes_on FROM micromanager.application_usage ORDER BY bundle_id",
	)
	if err != nil {
		t.Fatalf("query application usage: %v", err)
	}
	defer rows.Close()

	stored := make([]storedApplicationUsage, 0)
	for rows.Next() {
		var row storedApplicationUsage
		if err := rows.Scan(&row.UserID, &row.Date, &row.BundleID, &row.MinutesOn); err != nil {
			t.Fatalf("scan application usage: %v", err)
		}
		stored = append(stored, row)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate application usage: %v", err)
	}

	return stored
}

func doRequest(t *testing.T, method, path, body, contentType string) (int, []byte) {
	t.Helper()

	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}

	req, err := http.NewRequest(method, apiBaseURL+path, reader)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
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

	return resp.StatusCode, respBody
}

// TestScreentimeEndpoints drives the screentime endpoint over HTTP and verifies responses and persisted state.
func TestScreentimeEndpoints(t *testing.T) {
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
			resetTables(t)

			status, respBody := doRequest(t, tt.method, tt.path, tt.body, tt.contentType)

			if status != tt.wantStatus {
				t.Fatalf("status: got %d, want %d (body: %s)", status, tt.wantStatus, respBody)
			}
			for _, want := range tt.wantBodyContains {
				if !strings.Contains(string(respBody), want) {
					t.Errorf("response body %q does not contain %q", respBody, want)
				}
			}

			assertScreentimeRowCount(t, tt.wantRows)
			if tt.wantStored != nil {
				assertStoredScreentime(t, tt.wantStored)
			}
		})
	}
}

// TestAppUsageEndpoints drives the application usage endpoint over HTTP and verifies responses and persisted state.
func TestAppUsageEndpoints(t *testing.T) {
	const validPayload = `{"user_id":"user-1","date":"2024-05-01","applications":[{"bundle_id":"com.apple.Safari","minutes_on":10},{"bundle_id":"com.microsoft.VSCode","minutes_on":3.5}]}`

	tests := []struct {
		name             string
		method           string
		path             string
		body             string
		contentType      string
		wantStatus       int
		wantBodyContains []string
		wantRows         int
		wantStored       []storedApplicationUsage
	}{
		{
			name:             "create app usage persists and returns 201",
			method:           http.MethodPost,
			path:             "/v1/app-usage",
			body:             validPayload,
			contentType:      "application/json",
			wantStatus:       http.StatusCreated,
			wantBodyContains: []string{`"message":"ok"`},
			wantRows:         2,
			wantStored: []storedApplicationUsage{
				{UserID: "user-1", Date: "2024-05-01", BundleID: "com.apple.Safari", MinutesOn: 10},
				{UserID: "user-1", Date: "2024-05-01", BundleID: "com.microsoft.VSCode", MinutesOn: 3.5},
			},
		},
		{
			name:             "create app usage accepts an empty application list with 201",
			method:           http.MethodPost,
			path:             "/v1/app-usage",
			body:             `{"user_id":"user-2","date":"2024-05-02","applications":[]}`,
			contentType:      "application/json",
			wantStatus:       http.StatusCreated,
			wantBodyContains: []string{`"message":"ok"`},
			wantRows:         0,
		},
		{
			name:             "create app usage rejects invalid date with 500",
			method:           http.MethodPost,
			path:             "/v1/app-usage",
			body:             `{"user_id":"user-3","date":"not-a-date","applications":[{"bundle_id":"com.apple.Safari","minutes_on":5}]}`,
			contentType:      "application/json",
			wantStatus:       http.StatusInternalServerError,
			wantBodyContains: []string{`"status":500`},
			wantRows:         0,
		},
		{
			name:             "create app usage rejects malformed JSON with 400",
			method:           http.MethodPost,
			path:             "/v1/app-usage",
			body:             `{"user_id":`,
			contentType:      "application/json",
			wantStatus:       http.StatusBadRequest,
			wantBodyContains: []string{`"status":400`},
			wantRows:         0,
		},
		{
			name:             "create app usage rejects wrong field type with 422",
			method:           http.MethodPost,
			path:             "/v1/app-usage",
			body:             `{"user_id":"user-4","date":"2024-05-01","applications":[{"bundle_id":"com.apple.Safari","minutes_on":"many"}]}`,
			contentType:      "application/json",
			wantStatus:       http.StatusUnprocessableEntity,
			wantBodyContains: []string{`"status":422`},
			wantRows:         0,
		},
		{
			name:       "create app usage rejects unsupported method with 405",
			method:     http.MethodGet,
			path:       "/v1/app-usage",
			wantStatus: http.StatusMethodNotAllowed,
			wantRows:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetTables(t)

			status, respBody := doRequest(t, tt.method, tt.path, tt.body, tt.contentType)

			if status != tt.wantStatus {
				t.Fatalf("status: got %d, want %d (body: %s)", status, tt.wantStatus, respBody)
			}
			for _, want := range tt.wantBodyContains {
				if !strings.Contains(string(respBody), want) {
					t.Errorf("response body %q does not contain %q", respBody, want)
				}
			}

			assertApplicationUsageRowCount(t, tt.wantRows)
			if tt.wantStored != nil {
				stored := readApplicationUsage(t)
				if len(stored) != len(tt.wantStored) {
					t.Fatalf("stored rows: got %+v, want %+v", stored, tt.wantStored)
				}
				for i := range tt.wantStored {
					if stored[i] != tt.wantStored[i] {
						t.Errorf("stored row %d: got %+v, want %+v", i, stored[i], tt.wantStored[i])
					}
				}
			}
		})
	}
}

type fakeActivityRepo struct {
	activity *model.Activity
	apps     []model.ApplicationUsage
}

func (f fakeActivityRepo) GetBacklitMetrics(context.Context) (*model.Activity, error) {
	return f.activity, nil
}

func (f fakeActivityRepo) GetMinutesPerApplication(context.Context) ([]model.ApplicationUsage, error) {
	return f.apps, nil
}

// TestActivityServiceCaptureEndToEnd drives the poller client against the running API and checks persisted state.
func TestActivityServiceCaptureEndToEnd(t *testing.T) {
	log := jsonlog.New(io.Discard, jsonlog.LevelError)

	tests := []struct {
		name               string
		captureAll         bool
		activity           *model.Activity
		apps               []model.ApplicationUsage
		wantScreentimeRows int
		wantAppRows        int
	}{
		{
			name: "captures application usage through the running API",
			apps: []model.ApplicationUsage{
				{BundleID: "com.apple.Safari", MinutesOn: 10},
				{BundleID: "com.microsoft.VSCode", MinutesOn: 3.5},
			},
			wantAppRows: 2,
		},
		{
			name:               "captures screentime and application usage through the running API",
			captureAll:         true,
			activity:           &model.Activity{Date: time.Date(2024, 5, 1, 12, 0, 0, 0, time.Local), MinutesOn: 42},
			apps:               []model.ApplicationUsage{{BundleID: "com.apple.Safari", MinutesOn: 10}},
			wantScreentimeRows: 1,
			wantAppRows:        1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetTables(t)

			svc := service.NewActivityServiceWithAPI(
				fakeActivityRepo{activity: tt.activity, apps: tt.apps},
				log,
				apiBaseURL,
				nil,
			)

			var err error
			if tt.captureAll {
				err = svc.CaptureAll(context.Background())
			} else {
				err = svc.CaptureApplicationUsage(context.Background())
			}
			if err != nil {
				t.Fatalf("capture: %v", err)
			}

			assertScreentimeRowCount(t, tt.wantScreentimeRows)
			assertApplicationUsageRowCount(t, tt.wantAppRows)
		})
	}
}
