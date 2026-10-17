package output

// Screentime is the response body returned after creating a screentime entry.
type Screentime struct {
	Body ScreentimeResponse
}

// ScreentimeResponse holds the fields returned after creating a screentime entry.
type ScreentimeResponse struct {
	Message string `json:"message"`
}
