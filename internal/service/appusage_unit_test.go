//go:build unit

package service

import (
	"context"
	"errors"
	"io"
	"reflect"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/johnmantios/micromanager/internal/assert"
	"github.com/johnmantios/micromanager/internal/jsonlog"
	"github.com/johnmantios/micromanager/internal/model"
	"github.com/johnmantios/micromanager/internal/repo"
	"github.com/johnmantios/micromanager/internal/server/input"
)

type stubApplicationUsageRepo struct {
	err      error
	calls    int
	userID   string
	date     string
	inserted []model.ApplicationUsage
}

func (s *stubApplicationUsageRepo) InsertApplicationUsage(_ context.Context, userID, date string, applications []model.ApplicationUsage) error {
	s.calls++
	s.userID = userID
	s.date = date
	s.inserted = applications
	return s.err
}

var _ repo.IApplicationUsage = (*stubApplicationUsageRepo)(nil)

func newAppUsageInput(userID, date string, applications ...input.AppUsageItem) *input.AppUsage {
	in := &input.AppUsage{}
	in.Body.UserID = userID
	in.Body.Date = date
	in.Body.Applications = applications
	return in
}

// TestCreateAppUsageHandler verifies the mapping from API input to the
// persistence call and the error translation performed by the service.
func TestCreateAppUsageHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		input        *input.AppUsage
		repoErr      error
		wantErr      bool
		wantStatus   int
		wantMessage  string
		wantUserID   string
		wantDate     string
		wantInserted []model.ApplicationUsage
	}{
		{
			name: "maps applications and returns ok",
			input: newAppUsageInput("user-1", "2024-05-01",
				input.AppUsageItem{BundleID: "com.apple.Safari", MinutesOn: 10},
				input.AppUsageItem{BundleID: "com.microsoft.VSCode", MinutesOn: 3},
			),
			wantMessage: "ok",
			wantUserID:  "user-1",
			wantDate:    "2024-05-01",
			wantInserted: []model.ApplicationUsage{
				{BundleID: "com.apple.Safari", MinutesOn: 10},
				{BundleID: "com.microsoft.VSCode", MinutesOn: 3},
			},
		},
		{
			name:         "accepts an empty application list",
			input:        newAppUsageInput("user-2", "2024-05-02"),
			wantMessage:  "ok",
			wantUserID:   "user-2",
			wantDate:     "2024-05-02",
			wantInserted: []model.ApplicationUsage{},
		},
		{
			name:        "translates repository error to 500",
			input:       newAppUsageInput("user-3", "2024-05-03", input.AppUsageItem{BundleID: "com.apple.Safari", MinutesOn: 1}),
			repoErr:     errors.New("insert failed"),
			wantErr:     true,
			wantStatus:  500,
			wantMessage: "Could not insert application usage in database",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			stub := &stubApplicationUsageRepo{err: tt.repoErr}
			svc := NewApplicationUsageService(stub, jsonlog.New(io.Discard, jsonlog.LevelError))

			got, err := svc.CreateAppUsageHandler(context.Background(), tt.input)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}

				var statusErr huma.StatusError
				if !errors.As(err, &statusErr) {
					t.Fatalf("expected huma.StatusError, got %T", err)
				}
				assert.Equal(t, statusErr.GetStatus(), tt.wantStatus)
				assert.Equal(t, statusErr.Error(), tt.wantMessage)
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				assert.Equal(t, got.Body.Message, tt.wantMessage)
			}

			if tt.wantInserted == nil {
				return
			}
			assert.Equal(t, stub.userID, tt.wantUserID)
			assert.Equal(t, stub.date, tt.wantDate)
			if !reflect.DeepEqual(stub.inserted, tt.wantInserted) {
				t.Errorf("inserted applications: got %+v, want %+v", stub.inserted, tt.wantInserted)
			}
		})
	}
}
