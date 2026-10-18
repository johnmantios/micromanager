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
	inserted *model.AppUsage
}

func (s *stubApplicationUsageRepo) InsertApplicationUsage(_ context.Context, usage *model.AppUsage) error {
	s.calls++
	s.inserted = usage
	return s.err
}

var _ repo.IApplicationUsage = (*stubApplicationUsageRepo)(nil)

func newAppUsageInput(userID, username, date string, applications ...input.AppUsageItem) *input.AppUsage {
	in := &input.AppUsage{}
	in.Body.UserID = userID
	in.Body.Username = username
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
		wantUsername string
		wantDate     string
		wantInserted []model.ApplicationUsage
	}{
		{
			name: "maps applications and returns ok",
			input: newAppUsageInput("user-1", "jane", "2024-05-01",
				input.AppUsageItem{BundleID: "com.apple.Safari", MinutesOn: 10},
				input.AppUsageItem{BundleID: "com.microsoft.VSCode", MinutesOn: 3},
			),
			wantMessage:  "ok",
			wantUserID:   "user-1",
			wantUsername: "jane",
			wantDate:     "2024-05-01",
			wantInserted: []model.ApplicationUsage{
				{BundleID: "com.apple.Safari", MinutesOn: 10},
				{BundleID: "com.microsoft.VSCode", MinutesOn: 3},
			},
		},
		{
			name:         "accepts an empty application list",
			input:        newAppUsageInput("user-2", "amy", "2024-05-02"),
			wantMessage:  "ok",
			wantUserID:   "user-2",
			wantUsername: "amy",
			wantDate:     "2024-05-02",
			wantInserted: []model.ApplicationUsage{},
		},
		{
			name:        "translates repository error to 500",
			input:       newAppUsageInput("user-3", "john", "2024-05-03", input.AppUsageItem{BundleID: "com.apple.Safari", MinutesOn: 1}),
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
			if stub.inserted == nil {
				t.Fatalf("expected repository to receive usage, got nil")
			}
			assert.Equal(t, stub.inserted.UserID, tt.wantUserID)
			assert.Equal(t, stub.inserted.Username, tt.wantUsername)
			assert.Equal(t, stub.inserted.Date, tt.wantDate)
			if !reflect.DeepEqual(stub.inserted.Applications, tt.wantInserted) {
				t.Errorf("inserted applications: got %+v, want %+v", stub.inserted.Applications, tt.wantInserted)
			}
		})
	}
}
