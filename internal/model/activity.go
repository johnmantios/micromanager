package model

import "time"

// Activity is the entity that we have to send to the web service
type Activity struct {
	UserID    string    `json:"user_id"`
	Date      time.Time `json:"date"`
	MinutesOn float64   `json:"minutes_on"`
}
