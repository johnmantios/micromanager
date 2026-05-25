package model

import "time"

// Screentime is the entity that we have to send to the web service
type Screentime struct {
	UserID    string
	Username  string
	Date      time.Time
	MinutesOn int
}
