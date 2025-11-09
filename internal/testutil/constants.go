// Package testutil provides shared testing utilities and constants for the NBU exporter.
//
// This package centralizes common test constants, helper functions, and mock builders
// to reduce duplication across test files and improve test maintainability.
//
// # Key Components
//
// Constants: Shared test values (API keys, endpoints, error messages) defined in constants.go
//
// MockServerBuilder: Fluent interface for creating mock HTTP servers with configurable endpoints
//
// Helper Functions: Common test utilities (data loading, assertions) for cleaner test code
//
// # Usage Examples
//
// Creating a mock server:
//
//	server := testutil.NewMockServer().
//	    WithJobsEndpoint("13.0", jobsResponse).
//	    WithStorageEndpoint("13.0", storageResponse).
//	    Build()
//	defer server.Close()
//
// Loading test data:
//
//	data := testutil.LoadTestData(t, "testdata/jobs.json")
//
// Using shared constants:
//
//	apiKey := testutil.TestAPIKey
//	endpoint := testutil.TestPathAdminJobs
//
// # Design Pattern
//
// This package follows the Test Data Builder pattern to provide:
//   - Fluent, chainable API for mock server construction
//   - Centralized test constants to avoid duplication
//   - Clear, self-documenting test code
//   - Reusable helper functions for common test operations
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
