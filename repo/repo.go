package repo

import (
	"time"
)

type IRepository interface {
	SaveTick(event Event) error
}

type Event struct {
	IsLocked bool      `json:"is_locked"`
	Tick     time.Time `json:"tick"`
	UserID   string    `json:"user_id"`
}
