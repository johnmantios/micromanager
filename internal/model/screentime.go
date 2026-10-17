package model

// Screentime is the entity that we have to send to the web service
type Screentime struct {
	UserID    string  `json:"user_id" `
	Username  string  `json:"username" `
	Date      string  `json:"date" `
	MinutesOn float64 `json:"minutes_on" `
}
