package repo

import (
	"time"
)

type IRepository interface {
	SaveTick(event Event) error
}

type Event struct {
	UserID   string    `json:"user_id"`
	IsLocked bool      `json:"is_locked"`
	Tick     time.Time `json:"tick"`
}
