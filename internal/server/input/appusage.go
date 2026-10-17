package input

// AppUsage is the request body for creating application usage entries.
type AppUsage struct {
	Body AppUsageBody
}

// AppUsageBody holds the fields accepted when creating application usage entries.
type AppUsageBody struct {
	UserID       string         `json:"user_id" maxLength:"80"`
	Date         string         `json:"date" maxLength:"80"`
	Applications []AppUsageItem `json:"applications"`
}

// AppUsageItem holds the time spent in a single application.
type AppUsageItem struct {
	BundleID  string  `json:"bundle_id" maxLength:"255"`
	MinutesOn float64 `json:"minutes_on"`
}
