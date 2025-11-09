// Package exporter provides shared test constants and utilities.
// This file contains common constants used across multiple test files
// to avoid duplication and ensure consistency.
package exporter

import "github.com/fjacquet/nbu_exporter/internal/testutil"

// Shared test constants - aliased from testutil for backward compatibility
const (
	// HTTP headers
	contentTypeHeader   = testutil.ContentTypeHeader
	acceptHeader        = testutil.AcceptHeader
	authorizationHeader = testutil.AuthorizationHeader

	// Common test values
	contentTypeJSON                = testutil.ContentTypeJSON
	testAPIKey                     = testutil.TestAPIKey
	contentTypeNetBackupJSONFormat = testutil.ContentTypeNetBackupJSONFormat

	// API version strings
	apiVersion30  = testutil.APIVersion30
	apiVersion120 = testutil.APIVersion120
	apiVersion130 = testutil.APIVersion130

	// Test endpoints and paths
	testSchemeHTTPS      = testutil.TestSchemeHTTPS
	testPathAdminJobs    = testutil.TestPathAdminJobs
	testPathStorageUnits = testutil.TestPathStorageUnits
	testPathMetrics      = testutil.TestPathMetrics
	testPathNetBackup    = testutil.TestPathNetBackup

	// Test error messages
	testErrorAPIVersionNotSupported  = testutil.TestErrorAPIVersionNotSupported
	testErrorFetchAllJobsFailed      = testutil.TestErrorFetchAllJobsFailed
	testErrorFetchDataUnexpected     = testutil.TestErrorFetchDataUnexpected
	testErrorResponseShouldContain   = testutil.TestErrorResponseShouldContain
	testErrorExpectedError           = testutil.TestErrorExpectedError
	testErrorUnexpected              = testutil.TestErrorUnexpected
	testErrorValidateUnexpected      = testutil.TestErrorValidateUnexpected
	testErrorExpectedErrorContaining = testutil.TestErrorExpectedErrorContaining
	testInvalidAPIVersion            = testutil.TestInvalidAPIVersion

	// Test storage unit names
	testStorageUnitDiskPool1 = testutil.TestStorageUnitDiskPool1
	testStorageUnitTapeStu1  = testutil.TestStorageUnitTapeStu1

	// Test server names and identifiers
	testServerNBUMaster   = testutil.TestServerNBUMaster
	testServerName        = testutil.TestServerName
	testOTELEndpoint      = testutil.TestOTELEndpoint
	testServiceName       = testutil.TestServiceName
	testServiceVersion    = testutil.TestServiceVersion
	testKeyName           = testutil.TestKeyName
	testInvalidServerPort = testutil.TestInvalidServerPort
)
