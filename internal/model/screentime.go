package model

import "time"

// Screentime is the entity that we have to send to Prometheus. Every OS implementation has to respect it
type Screentime struct {
	Date      time.Time
	MinutesOn int
}
