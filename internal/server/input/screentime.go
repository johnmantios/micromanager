package input

// Screentime is the request body for creating a screentime entry.
type Screentime struct {
	Body ScreentimeBody
}

// ScreentimeBody holds the fields accepted when creating a screentime entry.
type ScreentimeBody struct {
	UserID    string  `json:"user_id" maxLength:"80"`
	Username  string  `json:"username" maxLength:"80"`
	Date      string  `json:"date" maxLength:"80"`
	MinutesOn float64 `json:"minutes_on"`
}
