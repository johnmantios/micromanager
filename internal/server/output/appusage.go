package output

// AppUsage is the response body returned after creating application usage entries.
type AppUsage struct {
	Body AppUsageResponse
}

// AppUsageResponse holds the fields returned after creating application usage entries.
type AppUsageResponse struct {
	Message string `json:"message"`
}
