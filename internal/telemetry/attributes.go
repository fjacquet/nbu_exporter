package telemetry

// This file defines centralized span attribute constants for OpenTelemetry instrumentation.
// Using constants instead of string literals prevents typos, enables IDE autocomplete,
// and makes refactoring easier.
//
// Attributes are organized into three categories:
//   - HTTP: Standard HTTP semantic conventions
//   - NetBackup: NetBackup-specific attributes for API calls and data operations
//   - Scrape: Prometheus scrape cycle metrics
//
// Usage:
//
//	span.SetAttributes(
//	    attribute.String(telemetry.AttrHTTPMethod, "GET"),
//	    attribute.String(telemetry.AttrHTTPURL, url),
//	    attribute.Int(telemetry.AttrHTTPStatusCode, statusCode),
//	)

// HTTP semantic convention attributes
const (
	AttrHTTPMethod                = "http.method"
	AttrHTTPURL                   = "http.url"
	AttrHTTPStatusCode            = "http.status_code"
	AttrHTTPRequestContentLength  = "http.request_content_length"
	AttrHTTPResponseContentLength = "http.response_content_length"
	AttrHTTPDurationMS            = "http.duration_ms"
)

// NetBackup-specific attributes
const (
	AttrNetBackupEndpoint     = "netbackup.endpoint"
	AttrNetBackupStorageUnits = "netbackup.storage_units"
	AttrNetBackupAPIVersion   = "netbackup.api_version"
	AttrNetBackupTimeWindow   = "netbackup.time_window"
	AttrNetBackupStartTime    = "netbackup.start_time"
	AttrNetBackupTotalJobs    = "netbackup.total_jobs"
	AttrNetBackupTotalPages   = "netbackup.total_pages"
	AttrNetBackupPageOffset   = "netbackup.page_offset"
	AttrNetBackupPageNumber   = "netbackup.page_number"
	AttrNetBackupJobsInPage   = "netbackup.jobs_in_page"
)

// Scrape cycle attributes
const (
	AttrScrapeDurationMS          = "scrape.duration_ms"
	AttrScrapeStorageMetricsCount = "scrape.storage_metrics_count"
	AttrScrapeJobMetricsCount     = "scrape.job_metrics_count"
	AttrScrapeStatus              = "scrape.status"
)

// Error attributes
const (
	AttrError = "error"
)
