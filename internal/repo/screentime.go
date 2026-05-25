package repo

import (
	"context"
	"database/sql"
	"github.com/johnmantios/micromanager/internal/model"
	"time"
)

type IScreentime interface {
	InsertBacklitMetrics(ctx context.Context, screentime *model.Screentime) error
}

type ScreentimeRepo struct {
	DB *sql.DB
}

func (m ScreentimeRepo) InsertBacklitMetrics(ctx context.Context, screentime *model.Screentime) error {
	query := `INSERT INTO micromanager.screentime (user_id, username, date, minutes_on)
    			VALUES ($1, $2, $3, $4)
				RETURNING username;`

	args := []any{screentime.UserID, screentime.Username, screentime.Date, screentime.MinutesOn}

	cancelCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	return m.DB.QueryRowContext(cancelCtx, query, args).Scan(&screentime.Username)

}

func NewScreentimeRepo(db *sql.DB) (*ScreentimeRepo, error) {
	return &ScreentimeRepo{
		DB: db,
	}, nil
}
