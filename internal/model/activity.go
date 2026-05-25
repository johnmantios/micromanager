package model

import "time"

// Activity is the entity that we have to send to the web service
type Activity struct {
	UserID    string
	Date      time.Time
	MinutesOn int
}
