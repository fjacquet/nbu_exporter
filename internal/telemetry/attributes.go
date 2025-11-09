package telemetry

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
