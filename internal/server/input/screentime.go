package input

type Screentime struct {
	UserID    string  `json:"user_id"  maxLength:"80"`
	Username  string  `json:"username"  maxLength:"80"`
	Date      string  `json:"date"  maxLength:"80"`
	MinutesOn float64 `json:"minutes_on"  maxLength:"80"`
}
