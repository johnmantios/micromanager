//go:build unit

package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/johnmantios/micromanager/internal/jsonlog"
	"github.com/johnmantios/micromanager/internal/model"
	"github.com/johnmantios/micromanager/internal/server/input"
)

type stubActivityRepo struct {
	activity    *model.Activity
	activityErr error
	apps        []model.ApplicationUsage
	appsErr     error
}

func (s *stubActivityRepo) GetBacklitMetrics(_ context.Context) (*model.Activity, error) {
	return s.activity, s.activityErr
}

func (s *stubActivityRepo) GetMinutesPerApplication(_ context.Context) ([]model.ApplicationUsage, error) {
	return s.apps, s.appsErr
}

type capturedRequest struct {
	method      string
	path        string
	contentType string
	body        []byte
}

func newCaptureServer(t *testing.T, status int, captured *[]capturedRequest) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		*captured = append(*captured, capturedRequest{
			method:      r.Method,
			path:        r.URL.Path,
			contentType: r.Header.Get("Content-Type"),
			body:        body,
		})
		w.WriteHeader(status)
	}))
	t.Cleanup(server.Close)

	return server
}

func newActivityServiceForTest(stub *stubActivityRepo, server *httptest.Server) ActivityService {
	return NewActivityServiceWithAPI(stub, jsonlog.New(os.Stdout, jsonlog.LevelInfo), server.URL, server.Client())
}

func capturedPaths(captured []capturedRequest) []string {
	paths := make([]string, 0, len(captured))
	for _, request := range captured {
		paths = append(paths, request.path)
	}
	return paths
}

// TestCaptureApplicationUsage verifies the client-side capture of per application
// usage, including payload mapping, the empty result, repository errors and API errors.
func TestCaptureApplicationUsage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		apps             []model.ApplicationUsage
		repoErr          error
		apiStatus        int
		wantErr          bool
		wantRequests     int
		wantPath         string
		wantApplications []input.AppUsageItem
	}{
		{
			name: "posts application usage and returns nil on 201",
			apps: []model.ApplicationUsage{
				{BundleID: "com.apple.Safari", MinutesOn: 10},
				{BundleID: "com.microsoft.VSCode", MinutesOn: 3.5},
			},
			apiStatus:    http.StatusCreated,
			wantRequests: 1,
			wantPath:     "/v1/app-usage",
			wantApplications: []input.AppUsageItem{
				{BundleID: "com.apple.Safari", MinutesOn: 10},
				{BundleID: "com.microsoft.VSCode", MinutesOn: 3.5},
			},
		},
		{
			name:         "skips the request when there is no application usage",
			apps:         []model.ApplicationUsage{},
			apiStatus:    http.StatusCreated,
			wantRequests: 0,
		},
		{
			name:         "returns the repository error without calling the API",
			repoErr:      errors.New("database is unavailable"),
			wantErr:      true,
			wantRequests: 0,
		},
		{
			name:         "returns an error when the API responds with a non-201",
			apps:         []model.ApplicationUsage{{BundleID: "com.apple.Safari", MinutesOn: 1}},
			apiStatus:    http.StatusInternalServerError,
			wantErr:      true,
			wantRequests: 1,
			wantPath:     "/v1/app-usage",
			wantApplications: []input.AppUsageItem{
				{BundleID: "com.apple.Safari", MinutesOn: 1},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var captured []capturedRequest
			server := newCaptureServer(t, tt.apiStatus, &captured)

			stub := &stubActivityRepo{apps: tt.apps, appsErr: tt.repoErr}
			svc := newActivityServiceForTest(stub, server)

			err := svc.CaptureApplicationUsage(context.Background())

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(captured) != tt.wantRequests {
				t.Fatalf("requests: got %d, want %d", len(captured), tt.wantRequests)
			}
			if tt.wantRequests == 0 {
				return
			}

			if captured[0].path != tt.wantPath {
				t.Errorf("path: got %q, want %q", captured[0].path, tt.wantPath)
			}
			if captured[0].contentType != "application/json" {
				t.Errorf("content type: got %q, want %q", captured[0].contentType, "application/json")
			}

			var payload input.AppUsageBody
			if err := json.Unmarshal(captured[0].body, &payload); err != nil {
				t.Fatalf("decode request body: %v", err)
			}
			if payload.Date == "" {
				t.Errorf("expected a non-empty date")
			}
			if len(payload.Applications) != len(tt.wantApplications) {
				t.Fatalf("applications: got %+v, want %+v", payload.Applications, tt.wantApplications)
			}
			for i := range tt.wantApplications {
				if payload.Applications[i] != tt.wantApplications[i] {
					t.Errorf("application %d: got %+v, want %+v", i, payload.Applications[i], tt.wantApplications[i])
				}
			}
		})
	}
}

// TestCaptureAll verifies that the combined capture runs both collectors and
// joins their errors.
func TestCaptureAll(t *testing.T) {
	t.Parallel()

	activity := &model.Activity{
		Date:      time.Date(2024, 5, 1, 12, 0, 0, 0, time.Local),
		MinutesOn: 42,
	}

	tests := []struct {
		name      string
		apps      []model.ApplicationUsage
		appsErr   error
		apiStatus int
		wantErr   bool
		wantPaths []string
	}{
		{
			name:      "captures backlit and application usage",
			apps:      []model.ApplicationUsage{{BundleID: "com.apple.Safari", MinutesOn: 10}},
			apiStatus: http.StatusCreated,
			wantPaths: []string{"/v1/screentime", "/v1/app-usage"},
		},
		{
			name:      "still captures screentime when application usage fails",
			appsErr:   errors.New("no application usage"),
			apiStatus: http.StatusCreated,
			wantErr:   true,
			wantPaths: []string{"/v1/screentime"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var captured []capturedRequest
			server := newCaptureServer(t, tt.apiStatus, &captured)

			stub := &stubActivityRepo{activity: activity, apps: tt.apps, appsErr: tt.appsErr}
			svc := newActivityServiceForTest(stub, server)

			err := svc.CaptureAll(context.Background())

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			gotPaths := capturedPaths(captured)
			if len(gotPaths) != len(tt.wantPaths) {
				t.Fatalf("request paths: got %v, want %v", gotPaths, tt.wantPaths)
			}
			for i := range tt.wantPaths {
				if gotPaths[i] != tt.wantPaths[i] {
					t.Errorf("request path %d: got %q, want %q", i, gotPaths[i], tt.wantPaths[i])
				}
			}
		})
	}
}
