package models

// Alerts is the response from GET /manage/alerts (JSON:API).
type Alerts struct {
	Data []struct {
		Attributes struct {
			Severity    string `json:"severity"` // INFO, WARNING, ERROR
			Category    string `json:"category"` // e.g. JOB
			SubCategory string `json:"subCategory"`
		} `json:"attributes"`
	} `json:"data"`
}
