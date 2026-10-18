package model

// AppUsage is the entity that we have to send to the web service: a single
// user's per-application usage for one day.
type AppUsage struct {
	UserID       string             `json:"user_id"`
	Username     string             `json:"username"`
	Date         string             `json:"date"`
	Applications []ApplicationUsage `json:"applications"`
}
