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

- [ ] 4. Update prometheus.go to batch span attributes
  - Modify `Collect` method to batch all span attributes in a single `SetAttributes` call
  - Use attribute constants from `telemetry.attributes` for all span attributes
  - Ensure nil-safe attribute recording with proper checks
  - _Requirements: 2.2, 7.1, 7.2, 7.3_

- [ ] 5. Enhance configuration validation
  - Add `validateOTelEndpoint` method to `Config` struct in `internal/models/Config.go`
  - Implement endpoint format validation (host:port pattern)
  - Implement port range validation (1-65535)
  - Update `Validate` method to call `validateOTelEndpoint` when OpenTelemetry is enabled
  - Add descriptive error messages for validation failures
  - _Requirements: 4.1, 4.2, 4.3, 4.4_

- [ ] 6. Improve test consistency and documentation
  - Update `internal/telemetry/manager_test.go` to explicitly check for errors instead of using underscore assignment
  - Add comments documenting graceful degradation cases in tests
  - Add godoc comments with parameter and return value documentation to exported functions in `client.go`, `netbackup.go`, and `prometheus.go`
  - Include usage examples in godoc for complex functions like `FetchStorage` and `FetchAllJobs`
  - _Requirements: 5.1, 5.2, 5.3, 8.1, 8.2, 8.3_

- [ ] 7. Add unit tests for new components
  - Write tests for `createSpan` helper function with nil tracer and valid tracer
  - Write tests for `validateOTelEndpoint` with valid and invalid formats
  - Write tests for extracted conditional functions (`shouldPerformVersionDetection`, `isExplicitVersionConfigured`)
  - Verify attribute constants are used correctly (compile-time check)
  - _Requirements: 1.3, 4.1, 6.3_

- [ ] 8. Run integration and regression tests
  - Execute existing OpenTelemetry integration tests to verify no behavioral changes
  - Run full test suite with `go test ./...` to ensure no breaking changes
  - Run benchmarks to verify no performance regression
  - Verify metrics collection continues to work correctly
  - _Requirements: 1.4, 7.4_

- [ ] 9. Fix test function naming convention
  - Rename all test functions to remove underscores (e.g., `TestNbuClient_GetHeaders` → `TestNbuClientGetHeaders`)
  - Use regex find/replace: `func Test(\w+)_(\w+)` → `func Test$1$2`
  - Handle multi-underscore cases: `Test(\w+)_(\w+)_(\w+)` → `func Test$1$2$3`
  - Run full test suite to verify all tests still pass after renaming
  - Update affected files: `client_test.go`, `netbackup_test.go`, `prometheus_test.go`, `version_detector_test.go`, `otel_integration_test.go`, `otel_benchmark_test.go`, `manager_test.go`, `Config_test.go`
  - _Requirements: 9.1, 9.2, 9.3, 9.4, 9.5_

- [ ] 10. Eliminate duplicate string literals
  - Identify string literals duplicated more than 3 times using SonarCloud report or grep
  - Extract test configuration strings to constants (e.g., `testAPIVersion`, `testAPIKey`, `testHost`)
  - Extract content type strings to constants (e.g., `ContentTypeJSON`)
  - Extract repeated YAML tag strings if used in multiple places
  - Group related constants together in appropriate files
  - Verify code still compiles and tests pass after extraction
  - _Requirements: 10.1, 10.2, 10.3, 10.4, 10.5_

- [ ] 11. Reduce cognitive complexity in test helpers
  - Refactor `createMockServerWithFile` function to reduce complexity from 22 to below 15
  - Extract header validation logic into `validateAPIVersionHeader` helper
  - Extract pagination detection into `isPaginatedJobsRequest` helper
  - Extract paginated response handling into `handlePaginatedJobsRequest` helper
  - Extract offset parsing into `parseOffsetFromRequest` helper
  - Extract pagination metadata setting into `setPaginationMetadata` and `setEmptyPaginationMetadata` helpers
  - Extract JSON response writing into `writeJSONResponse` helper
  - Verify all tests still pass after refactoring
  - _Requirements: 11.1, 11.2, 11.3, 11.4, 11.5_

- [ ] 12. Update documentation and changelog
  - Update CHANGELOG.md with improvements in "Changed" section
  - Document performance optimizations and enhanced validation
  - Document SonarCloud compliance improvements (test naming, string literals, cognitive complexity)
  - Add code comments explaining design decisions where appropriate
  - Ensure all new functions have complete godoc comments
  - _Requirements: 8.4, 9.5, 10.5, 11.5_
