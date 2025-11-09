// Package models provides shared test constants for model tests.
package models

import "github.com/fjacquet/nbu_exporter/internal/testutil"

// Shared test constants - aliased from testutil for backward compatibility
const (
	// Test endpoints and paths
	testPathMetrics   = testutil.TestPathMetrics
	testPathNetBackup = testutil.TestPathNetBackup
	testPathAdminJobs = testutil.TestPathAdminJobs

	// Test error messages
	testInvalidAPIVersion            = testutil.TestInvalidAPIVersion
	testErrorValidateUnexpected      = testutil.TestErrorValidateUnexpected
	testErrorExpectedError           = testutil.TestErrorExpectedError
	testErrorUnexpected              = testutil.TestErrorUnexpected
	testErrorExpectedErrorContaining = testutil.TestErrorExpectedErrorContaining

	// Test server names and identifiers
	testServerNBUMaster   = testutil.TestServerNBUMaster
	testOTELEndpoint      = testutil.TestOTELEndpoint
	testKeyName           = testutil.TestKeyName
	testAPIKey            = testutil.TestAPIKey
	testInvalidServerPort = testutil.TestInvalidServerPort
	testLogName           = testutil.TestLogName
	testSchemeHTTPS       = testutil.TestSchemeHTTP
	testPort1556          = testutil.TestPort1556
)
