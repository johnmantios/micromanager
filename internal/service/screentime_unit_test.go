//go:build unit

package service

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/johnmantios/micromanager/internal/assert"
	"github.com/johnmantios/micromanager/internal/jsonlog"
	"github.com/johnmantios/micromanager/internal/model"
	"github.com/johnmantios/micromanager/internal/repo"
	"github.com/johnmantios/micromanager/internal/server/input"
)

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

func newInput(userID, username, date string, minutesOn float64) *input.Screentime {
	in := &input.Screentime{}
	in.Body.UserID = userID
	in.Body.Username = username
	in.Body.Date = date
	in.Body.MinutesOn = minutesOn
	return in
}

// TestCreateScreentimeHandler verifies the mapping from API input to the
// persistence model and the error translation performed by the service.
func TestCreateScreentimeHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		input        *input.Screentime
		repoErr      error
		wantErr      bool
		wantStatus   int
		wantMessage  string
		wantInserted *model.Screentime
	}{
		{
			name:        "maps input to model and returns ok",
			input:       newInput("user-1", "jane", "2024-05-01", 42),
			wantMessage: "ok",
			wantInserted: &model.Screentime{
				UserID:    "user-1",
				Username:  "jane",
				Date:      "2024-05-01",
				MinutesOn: 42,
			},
		},
		{
			name:        "translates repository error to 500",
			input:       newInput("user-2", "john", "2024-05-02", 10),
			repoErr:     errors.New("insert failed"),
			wantErr:     true,
			wantStatus:  500,
			wantMessage: "Could not insert backlit metrics in database",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			stub := &stubScreentimeRepo{err: tt.repoErr}
			svc := NewScreentimeService(stub, jsonlog.New(io.Discard, jsonlog.LevelError))

			got, err := svc.CreateScreentimeHandler(context.Background(), tt.input)

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
				t.Fatalf("expected repository to receive %+v, got nil", tt.wantInserted)
			}
			assert.Equal(t, *stub.inserted, *tt.wantInserted)
		})
	}
}
