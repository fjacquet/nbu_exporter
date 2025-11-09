// Package exporter provides shared test constants and utilities.
// This file contains common constants used across multiple test files
// to avoid duplication and ensure consistency.
package exporter

// Shared test constants used across multiple test files in the exporter package
const (
	// HTTP headers
	contentTypeHeader   = "Content-Type"
	acceptHeader        = "Accept"
	authorizationHeader = "Authorization"

	// Common test values
	contentTypeJSON = "application/json"
	testAPIKey      = "test-api-key"

	// Test endpoints and paths
	testSchemeHTTPS      = "https://"
	testPathAdminJobs    = "/admin/jobs"
	testPathStorageUnits = "/storage/storage-units"
)
