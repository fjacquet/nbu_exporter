# Implementation Plan

- [x] 1. Create new telemetry infrastructure files
  - Create `internal/telemetry/attributes.go` with all span attribute constants organized by category (HTTP, NetBackup, Scrape)
  - Create `internal/telemetry/errors.go` with error message templates for API version and non-JSON response errors
  - Create `internal/exporter/tracing.go` with consolidated `createSpan` helper function
  - _Requirements: 1.1, 2.1, 2.3, 3.1, 3.2_

- [x] 2. Update client.go to use centralized helpers and constants
  - Replace `createHTTPSpan` with calls to consolidated `createSpan` function from `tracing.go`
  - Update `recordHTTPAttributes` to use attribute constants from `telemetry.attributes`
  - Replace inline error messages with templates from `telemetry.errors`
  - Extract `shouldPerformVersionDetection` and `isExplicitVersionConfigured` helper functions
  - Update all span attribute recording to use constants instead of string literals
  - _Requirements: 1.1, 1.2, 1.3, 2.2, 3.3, 6.1, 6.2_

- [x] 3. Update netbackup.go to use centralized helpers and constants
  - Replace `createStorageSpan`, `createJobsSpan`, and `createJobPageSpan` with calls to consolidated `createSpan` function
  - Update all span attribute recording to use constants from `telemetry.attributes`
  - Batch span attributes in `FetchStorage`, `FetchJobDetails`, and `FetchAllJobs` functions
  - _Requirements: 1.1, 1.2, 2.2, 7.1, 7.2_

- [x] 4. Update prometheus.go to batch span attributes
  - Modify `Collect` method to batch all span attributes in a single `SetAttributes` call
  - Use attribute constants from `telemetry.attributes` for all span attributes
  - Ensure nil-safe attribute recording with proper checks
  - _Requirements: 2.2, 7.1, 7.2, 7.3_

- [x] 5. Enhance configuration validation
  - Add `validateOTelEndpoint` method to `Config` struct in `internal/models/Config.go`
  - Implement endpoint format validation (host:port pattern)
  - Implement port range validation (1-65535)
  - Update `Validate` method to call `validateOTelEndpoint` when OpenTelemetry is enabled
  - Add descriptive error messages for validation failures
  - _Requirements: 4.1, 4.2, 4.3, 4.4_

- [x] 6. Improve test consistency and documentation
  - Update `internal/telemetry/manager_test.go` to explicitly check for errors instead of using underscore assignment
  - Add comments documenting graceful degradation cases in tests
  - Add godoc comments with parameter and return value documentation to exported functions in `client.go`, `netbackup.go`, and `prometheus.go`
  - Include usage examples in godoc for complex functions like `FetchStorage` and `FetchAllJobs`
  - _Requirements: 5.1, 5.2, 5.3, 8.1, 8.2, 8.3_

- [x] 7. Add unit tests for new components
  - Write tests for `createSpan` helper function with nil tracer and valid tracer
  - Write tests for `validateOTelEndpoint` with valid and invalid formats
  - Write tests for extracted conditional functions (`shouldPerformVersionDetection`, `isExplicitVersionConfigured`)
  - Verify attribute constants are used correctly (compile-time check)
  - _Requirements: 1.3, 4.1, 6.3_

- [x] 8. Run integration and regression tests
  - Execute existing OpenTelemetry integration tests to verify no behavioral changes
  - Run full test suite with `go test ./...` to ensure no breaking changes
  - Run benchmarks to verify no performance regression
  - Verify metrics collection continues to work correctly
  - _Requirements: 1.4, 7.4_

- [x] 9. Fix test function naming convention
  - Rename all test functions to remove underscores (e.g., `TestNbuClient_GetHeaders` → `TestNbuClientGetHeaders`)
  - Use regex find/replace: `func Test(\w+)_(\w+)` → `func Test$1$2`
  - Handle multi-underscore cases: `Test(\w+)_(\w+)_(\w+)` → `func Test$1$2$3`
  - Run full test suite to verify all tests still pass after renaming
  - Update affected files: `client_test.go`, `netbackup_test.go`, `prometheus_test.go`, `version_detector_test.go`, `otel_integration_test.go`, `otel_benchmark_test.go`, `manager_test.go`, `Config_test.go`
  - _Requirements: 9.1, 9.2, 9.3, 9.4, 9.5_

- [x] 10. Eliminate duplicate string literals
  - Identify string literals duplicated more than 3 times using SonarCloud report or grep
  - Extract test configuration strings to centralized constants (e.g., `testAPIVersion`, `testAPIKey`, `testHost`)
  - Extract content type strings to constants (e.g., `ContentTypeJSON`)
  - Extract repeated YAML tag strings if used in multiple places
  - Group related constants together in appropriate files
  - Verify code still compiles and tests pass after extraction
  - _Requirements: 10.1, 10.2, 10.3, 10.4, 10.5_

- [x] 11. Reduce cognitive complexity in test helpers
  - Refactor `createMockServerWithFile` function to reduce complexity from 22 to below 15
  - Extract header validation logic into `validateAPIVersionHeader` helper
  - Extract pagination detection into `isPaginatedJobsRequest` helper
  - Extract paginated response handling into `handlePaginatedJobsRequest` helper
  - Extract offset parsing into `parseOffsetFromRequest` helper
  - Extract pagination metadata setting into `setPaginationMetadata` and `setEmptyPaginationMetadata` helpers
  - Extract JSON response writing into `writeJSONResponse` helper
  - Verify all tests still pass after refactoring
  - _Requirements: 11.1, 11.2, 11.3, 11.4, 11.5_

- [x] 12. Centralize test helper functions
  - Create `internal/testutil/helpers.go` with shared test utilities
  - Implement `TestConfigBuilder` with fluent interface for building test configurations
  - Implement `MockServerBuilder` with fluent interface for creating mock HTTP servers
  - Add `LoadTestData` helper function for loading test fixtures
  - Migrate duplicate test helper functions from individual test files to centralized helpers
  - Update test files to use centralized helpers instead of local duplicates
  - _Requirements: 1.1, 6.1, 6.2_

- [x] 13. Reduce cognitive complexity

  - [x] 13.1 Reduce cognitive complexityin prometheus_test.go
    - Refactor `TestNewNbuCollectorAutomaticDetection` (line 76) to reduce complexity from 24 to below 15
    - Extract `createVersionMockServer` helper to handle mock server creation
    - Extract `extractVersionFromHeader` helper to parse version from Accept header
    - Extract `handleVersionResponse` helper to handle version-specific responses
    - Extract `assertCollectorResult` helper to validate collector creation results
    - Verify all tests still pass after refactoring
    - _Requirements: 11.1, 11.2, 11.3, 11.4, 11.5_

  - [x] 13.2 Reduce cognitive complexity in version_detection_integration_test.go
    - Refactor `TestAPIVersionDetectorIntegration` (line 14) to reduce complexity from 37 to below 15
    - Extract helper functions for test case setup and validation
    - Extract mock server creation logic into separate helper
    - Extract version detection validation into separate helper
    - Verify all tests still pass after refactoring
    - _Requirements: 11.1, 11.2, 11.3, 11.4, 11.5_

  - [x] 13.3 Reduce cognitive complexity in Config_test.go
    - Refactor `TestConfigValidateNbuServer` (line 69) to reduce complexity from 23 to below 15
    - Refactor `TestConfigValidateOpenTelemetry` (line 276) to reduce complexity from 16 to below 15
    - Refactor `TestConfigValidateServer` (line 514) to reduce complexity from 17 to below 15
    - Refactor `TestConfigGetNBUBaseURL` (line 898) to reduce complexity from 23 to below 15
    - Refactor `TestConfigValidate` (line 1385) to reduce complexity from 23 to below 15
    - Extract validation assertion helpers for each test
    - Extract test case setup helpers to reduce nesting
    - Verify all tests still pass after refactoring
    - _Requirements: 11.1, 11.2, 11.3, 11.4, 11.5_

- [x] 14. Eliminate duplicate string literals in version_detection_integration_test.go
  - Extract "version=13.0" to constant (duplicated 3 times, line 53)
  - Extract "version=12.0" to constant (duplicated 3 times, line 55)
  - Extract "Content-Type" to constant (duplicated 3 times, line 73)
  - Group version-related constants together
  - Verify all tests still pass after extraction
  - _Requirements: 10.1, 10.2, 10.3, 10.4, 10.5_

- [x] 15. Enhance error messages with additional context
  - Update error messages in `client.go` to include URL, status code, and content-type
  - Add request context to JSON unmarshal errors
  - Add response preview to non-JSON response errors
  - Ensure error messages provide actionable debugging information
  - _Requirements: 3.1, 3.2, 8.4_

- [x] 16. Add package-level documentation
  - Add comprehensive package documentation to `internal/testutil` package
  - Document key components: constants, builders, helper functions
  - Include usage examples in package documentation
  - Document the test constants consolidation pattern
  - Add package documentation to `internal/telemetry` if missing
  - _Requirements: 8.1, 8.2, 8.3, 8.4_

- [x] 17. Update documentation and changelog
  - Update CHANGELOG.md with improvements in "Changed" section
  - Document performance optimizations and enhanced validation
  - Document SonarCloud compliance improvements (test naming, string literals, cognitive complexity)
  - Document test infrastructure improvements (centralized helpers, reduced complexity)
  - List all cognitive complexity reductions (8 test functions refactored)
  - List all string literal extractions (version headers, content-type)
  - Add code comments explaining design decisions where appropriate
  - Ensure all new functions have complete godoc comments
  - _Requirements: 8.4, 9.5, 10.5, 11.5_
