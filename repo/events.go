package repo

import (
	"context"
	"database/sql"
	"time"
)

type IEventsModel interface {
	SaveTick(event EventEntity) error
}

type EventsModel struct {
	DB *sql.DB
}

type EventEntity struct {
	UserID   string    `json:"user_id"`
	IsLocked bool      `json:"is_locked"`
	Tick     time.Time `json:"tick"`
}

func (m EventsModel) SaveTick(event EventEntity) error {
	query := `
				INSERT INTO micromanager.tick (tick, is_locked, user_id)
				VALUES ($1, $2, $3)
				RETURNING tick;
				`

	args := []any{
		event.Tick,
		event.IsLocked,
		event.UserID,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return m.DB.QueryRowContext(ctx, query, args...).Scan(&event.Tick)
}

func NewEventsModel(db *sql.DB) (*EventsModel, error) {
	return &EventsModel{
		DB: db,
	}, nil
}
