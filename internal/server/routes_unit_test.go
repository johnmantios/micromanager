//go:build unit

package server

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/johnmantios/micromanager/internal/assert"
	"github.com/johnmantios/micromanager/internal/jsonlog"
	"github.com/johnmantios/micromanager/internal/model"
	"github.com/johnmantios/micromanager/internal/repo"
	"github.com/johnmantios/micromanager/internal/service"
)

// stubScreentimeRepo is an in-memory test double for repo.IScreentime.
type stubScreentimeRepo struct {
	err      error
	calls    int
	inserted *model.Screentime
}

func (s *stubScreentimeRepo) InsertBacklitMetrics(_ context.Context, screentime *model.Screentime) error {
	s.calls++
	s.inserted = screentime
	return s.err
}

var _ repo.IScreentime = (*stubScreentimeRepo)(nil)

type appUsageInsert struct {
	userID       string
	date         string
	applications []model.ApplicationUsage
}

// stubApplicationUsageRepo is an in-memory test double for repo.IApplicationUsage.
type stubApplicationUsageRepo struct {
	err      error
	calls    int
	inserted appUsageInsert
}

func (s *stubApplicationUsageRepo) InsertApplicationUsage(_ context.Context, userID, date string, applications []model.ApplicationUsage) error {
	s.calls++
	s.inserted = appUsageInsert{userID: userID, date: date, applications: applications}
	return s.err
}

var _ repo.IApplicationUsage = (*stubApplicationUsageRepo)(nil)

func discardLogger() *jsonlog.Logger {
	return jsonlog.New(io.Discard, jsonlog.LevelError)
}

// TestRoutesHandler exercises internal/server/routes.go: the router wiring, the
// Huma screentime and application usage endpoints, the healthcheck endpoint and
// the middleware chain. It uses stubbed repositories so no external services are
// required.
func TestRoutesHandler(t *testing.T) {
	t.Parallel()

	const validScreentimePayload = `{"user_id":"user-1","username":"jane","date":"2024-05-01","minutes_on":42}`
	const validAppUsagePayload = `{"user_id":"user-1","date":"2024-05-01","applications":[{"bundle_id":"com.apple.Safari","minutes_on":10},{"bundle_id":"com.microsoft.VSCode","minutes_on":3}]}`

	tests := []struct {
		name             string
		method           string
		path             string
		body             string
		contentType      string
		repoErr          error
		appRepoErr       error
		wantStatus       int
		wantContentType  string
		wantBodyExact    string
		wantBodyContains []string
		wantRepoCalls    int
		wantInserted     *model.Screentime
		wantAppRepoCalls int
		wantInsertedApp  *appUsageInsert
		wantCORS         bool
		skipSecureHeader bool
	}{
		{
			name:            "healthcheck returns name with 200",
			method:          http.MethodGet,
			path:            "/v1/healthcheck",
			wantStatus:      http.StatusOK,
			wantContentType: "text/plain; charset=utf-8",
			wantBodyExact:   "micromanager",
			wantCORS:        true,
		},
		{
			name:             "CORS preflight short-circuits with 200",
			method:           http.MethodOptions,
			path:             "/v1/healthcheck",
			wantStatus:       http.StatusOK,
			wantCORS:         true,
			skipSecureHeader: true,
		},
		{
			name:          "create screentime maps payload and returns 201",
			method:        http.MethodPost,
			path:          "/v1/screentime",
			body:          validScreentimePayload,
			contentType:   "application/json",
			wantStatus:    http.StatusCreated,
			wantCORS:      true,
			wantRepoCalls: 1,
			wantInserted:  &model.Screentime{UserID: "user-1", Username: "jane", Date: "2024-05-01", MinutesOn: 42},
			wantBodyContains: []string{
				`"message":"ok"`,
			},
		},
		{
			name:             "create screentime returns 500 when repository fails",
			method:           http.MethodPost,
			path:             "/v1/screentime",
			body:             validScreentimePayload,
			contentType:      "application/json",
			repoErr:          errors.New("database is unavailable"),
			wantStatus:       http.StatusInternalServerError,
			wantCORS:         true,
			wantRepoCalls:    1,
			wantBodyContains: []string{`"status":500`, "Could not insert backlit metrics in database"},
		},
		{
			name:             "create screentime rejects malformed JSON with 400",
			method:           http.MethodPost,
			path:             "/v1/screentime",
			body:             `{"user_id":`,
			contentType:      "application/json",
			wantStatus:       http.StatusBadRequest,
			wantCORS:         true,
			wantRepoCalls:    0,
			wantBodyContains: []string{`"status":400`},
		},
		{
			name:             "create screentime rejects wrong field type with 422",
			method:           http.MethodPost,
			path:             "/v1/screentime",
			body:             `{"user_id":"user-1","username":"jane","date":"2024-05-01","minutes_on":"not-a-number"}`,
			contentType:      "application/json",
			wantStatus:       http.StatusUnprocessableEntity,
			wantCORS:         true,
			wantRepoCalls:    0,
			wantBodyContains: []string{`"status":422`},
		},
		{
			name:          "create screentime rejects unsupported method with 405",
			method:        http.MethodGet,
			path:          "/v1/screentime",
			wantStatus:    http.StatusMethodNotAllowed,
			wantCORS:      true,
			wantRepoCalls: 0,
		},
		{
			name:             "create app usage maps payload and returns 201",
			method:           http.MethodPost,
			path:             "/v1/app-usage",
			body:             validAppUsagePayload,
			contentType:      "application/json",
			wantStatus:       http.StatusCreated,
			wantCORS:         true,
			wantAppRepoCalls: 1,
			wantInsertedApp: &appUsageInsert{
				userID: "user-1",
				date:   "2024-05-01",
				applications: []model.ApplicationUsage{
					{BundleID: "com.apple.Safari", MinutesOn: 10},
					{BundleID: "com.microsoft.VSCode", MinutesOn: 3},
				},
			},
			wantBodyContains: []string{`"message":"ok"`},
		},
		{
			name:             "create app usage accepts an empty application list",
			method:           http.MethodPost,
			path:             "/v1/app-usage",
			body:             `{"user_id":"user-2","date":"2024-05-02","applications":[]}`,
			contentType:      "application/json",
			wantStatus:       http.StatusCreated,
			wantCORS:         true,
			wantAppRepoCalls: 1,
			wantInsertedApp: &appUsageInsert{
				userID:       "user-2",
				date:         "2024-05-02",
				applications: []model.ApplicationUsage{},
			},
			wantBodyContains: []string{`"message":"ok"`},
		},
		{
			name:             "create app usage returns 500 when repository fails",
			method:           http.MethodPost,
			path:             "/v1/app-usage",
			body:             validAppUsagePayload,
			contentType:      "application/json",
			appRepoErr:       errors.New("database is unavailable"),
			wantStatus:       http.StatusInternalServerError,
			wantCORS:         true,
			wantAppRepoCalls: 1,
			wantBodyContains: []string{`"status":500`, "Could not insert application usage in database"},
		},
		{
			name:             "create app usage rejects malformed JSON with 400",
			method:           http.MethodPost,
			path:             "/v1/app-usage",
			body:             `{"user_id":`,
			contentType:      "application/json",
			wantStatus:       http.StatusBadRequest,
			wantCORS:         true,
			wantAppRepoCalls: 0,
			wantBodyContains: []string{`"status":400`},
		},
		{
			name:             "create app usage rejects wrong field type with 422",
			method:           http.MethodPost,
			path:             "/v1/app-usage",
			body:             `{"user_id":"user-1","date":"2024-05-01","applications":[{"bundle_id":"com.apple.Safari","minutes_on":"ten"}]}`,
			contentType:      "application/json",
			wantStatus:       http.StatusUnprocessableEntity,
			wantCORS:         true,
			wantAppRepoCalls: 0,
			wantBodyContains: []string{`"status":422`},
		},
		{
			name:             "create app usage rejects unsupported method with 405",
			method:           http.MethodGet,
			path:             "/v1/app-usage",
			wantStatus:       http.StatusMethodNotAllowed,
			wantCORS:         true,
			wantAppRepoCalls: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			stub := &stubScreentimeRepo{err: tt.repoErr}
			appStub := &stubApplicationUsageRepo{err: tt.appRepoErr}
			handler := newAPIRouter(
				service.NewScreentimeService(stub, discardLogger()),
				service.NewApplicationUsageService(appStub, discardLogger()),
				NewMiddleware(discardLogger()),
			).Routes()

			var body io.Reader
			if tt.body != "" {
				body = strings.NewReader(tt.body)
			}

			req := httptest.NewRequest(tt.method, tt.path, body)
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			assert.Equal(t, rec.Code, tt.wantStatus)

			if tt.wantContentType != "" {
				assert.Equal(t, rec.Header().Get("Content-Type"), tt.wantContentType)
			}
			if tt.wantBodyExact != "" {
				assert.Equal(t, rec.Body.String(), tt.wantBodyExact)
			}
			for _, want := range tt.wantBodyContains {
				if !strings.Contains(rec.Body.String(), want) {
					t.Errorf("response body %q does not contain %q", rec.Body.String(), want)
				}
			}

			assert.Equal(t, stub.calls, tt.wantRepoCalls)
			if tt.wantInserted != nil {
				if stub.inserted == nil {
					t.Fatalf("expected screentime repository to receive %+v, got nil", tt.wantInserted)
				}
				assert.Equal(t, *stub.inserted, *tt.wantInserted)
			}

			assert.Equal(t, appStub.calls, tt.wantAppRepoCalls)
			if tt.wantInsertedApp != nil {
				assert.Equal(t, appStub.inserted.userID, tt.wantInsertedApp.userID)
				assert.Equal(t, appStub.inserted.date, tt.wantInsertedApp.date)
				if !reflect.DeepEqual(appStub.inserted.applications, tt.wantInsertedApp.applications) {
					t.Errorf("applications: got %+v, want %+v", appStub.inserted.applications, tt.wantInsertedApp.applications)
				}
			}

			if tt.wantCORS {
				assert.Equal(t, rec.Header().Get("Access-Control-Allow-Origin"), "*")
			}

			if tt.skipSecureHeader {
				return
			}

			assert.Equal(t, rec.Header().Get("X-Content-Type-Options"), "nosniff")
			assert.Equal(t, rec.Header().Get("X-Frame-Options"), "deny")
			assert.Equal(t, rec.Header().Get("X-XSS-Protection"), "0")
			assert.Equal(t, rec.Header().Get("Referrer-Policy"), "origin-when-cross-origin")
		})
	}
}
