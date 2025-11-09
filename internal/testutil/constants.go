// Package testutil provides shared test constants and utilities across all test files.
// This package consolidates common test values to avoid duplication and ensure consistency.
package testutil

// HTTP headers
const (
	ContentTypeHeader   = "Content-Type"
	AcceptHeader        = "Accept"
	AuthorizationHeader = "Authorization"
)

// Common test values
const (
	ContentTypeJSON                = "application/json"
	TestAPIKey                     = "test-api-key"
	ContentTypeNetBackupJSONFormat = "application/vnd.netbackup+json;version=%s"
)

// API version strings
const (
	APIVersion30  = "version=3.0"
	APIVersion120 = "version=12.0"
	APIVersion130 = "version=13.0"
)

// Test endpoints and paths
const (
	TestSchemeHTTPS      = "https://"
	TestSchemeHTTP       = "https" // Without ://
	TestPathAdminJobs    = "/admin/jobs"
	TestPathStorageUnits = "/storage/storage-units"
	TestPathMetrics      = "/metrics"
	TestPathNetBackup    = "/netbackup"
)

// Test URLs
const (
	TestFullAdminJobsURL = "https://nbu-master:1556/netbackup/admin/jobs"
)

// Test error messages
const (
	TestErrorAPIVersionNotSupported  = "API version not supported"
	TestErrorFetchAllJobsFailed      = "FetchAllJobs failed: %v"
	TestErrorFetchDataUnexpected     = "FetchData() unexpected error = %v"
	TestErrorResponseShouldContain   = "Response should contain data"
	TestErrorExpectedError           = "Expected error, got nil"
	TestErrorUnexpected              = "Unexpected error: %v"
	TestErrorValidateUnexpected      = "Validate() unexpected error = %v"
	TestErrorExpectedErrorContaining = "Expected error containing %q, got %q"
	TestInvalidAPIVersion            = "invalid API version format"
)

// Test storage unit names
const (
	TestStorageUnitDiskPool1 = "disk-pool-1"
	TestStorageUnitTapeStu1  = "tape-stu-1"
)

// Test server names and identifiers
const (
	TestServerNBUMaster   = "nbu-master"
	TestServerName        = "test-server"
	TestOTELEndpoint      = "localhost:4317"
	TestServiceName       = "nbu-exporter-test"
	TestServiceVersion    = "1.0.0-test"
	TestKeyName           = "test-key"
	TestInvalidServerPort = "invalid server port"
	TestLogName           = "test.log"
	TestPort1556          = "1556"
)
