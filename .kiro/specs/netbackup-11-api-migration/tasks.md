# Implementation Plan

- [x] 1. Update configuration model for multi-version support
  - Add supported API version constants (3.0, 12.0, 13.0) to `internal/models/Config.go`
  - Update default API version from "12.0" to "13.0" in `SetDefaults()` method
  - Enhance validation to check against supported versions list
  - _Requirements: 2.2, 2.3_

- [x] 2. Implement API version detection with fallback logic
  - [x] 2.1 Create version detector module
    - Create new file `internal/exporter/version_detector.go`
    - Implement `APIVersionDetector` struct with client and config fields
    - Implement `DetectVersion()` method with descending version order (13.0 → 12.0 → 3.0)
    - Implement `tryVersion()` helper method for testing individual versions
    - _Requirements: 1.1, 1.2, 1.3_

  - [x] 2.2 Add retry logic with exponential backoff
    - Define `RetryConfig` struct with max attempts, delays, and backoff factor
    - Implement exponential backoff in version detection
    - Distinguish between version incompatibility (HTTP 406) and transient errors
    - _Requirements: 1.5, 6.1, 6.2_

  - [x] 2.3 Enhance error handling and logging
    - Add detailed logging for each version attempt (DEBUG level)
    - Log detected version at INFO level on success
    - Provide comprehensive error messages with troubleshooting steps on failure
    - Handle authentication errors (HTTP 401) separately from version errors
    - _Requirements: 1.4, 6.1, 6.3, 6.4_

  - [x] 2.4 Write unit tests for version detector
    - Test fallback logic with mock HTTP responses
    - Test retry behavior with transient failures
    - Test error handling for different HTTP status codes
    - Test early exit on authentication errors
    - Verify logging output at each stage
    - _Requirements: 10.1, 10.4_

- [x] 3. Update HTTP client for version-aware requests
  - [x] 3.1 Enhance client initialization
    - Update `NewNbuClient()` to support version detection during initialization
    - Integrate `APIVersionDetector` into client creation flow
    - Add version detection bypass when apiVersion is explicitly configured
    - _Requirements: 1.1, 2.1_

  - [x] 3.2 Improve error messages for version-related failures
    - Enhance HTTP 406 error handling in `FetchData()` method
    - Add suggestions for version configuration in error messages
    - Include detected vs configured version information in errors
    - _Requirements: 6.1, 6.2_

  - [x] 3.3 Write unit tests for enhanced client
    - Test header construction for all three API versions
    - Test error message formatting for version failures
    - Test configuration override behavior
    - Verify retry logic integration
    - _Requirements: 10.1, 10.4_

- [ ] 4. Update collector initialization with version detection
  - [ ] 4.1 Integrate version detection into collector
    - Modify `NewNbuCollector()` in `internal/exporter/prometheus.go`
    - Add version detection call when apiVersion is not configured
    - Log detected version at INFO level
    - Handle version detection failures gracefully
    - _Requirements: 1.1, 1.3, 1.4_

  - [ ] 4.2 Add API version metric
    - Create new Prometheus gauge metric `nbu_api_version`
    - Expose current API version as metric with version label
    - Update metric in collector's `Describe()` and `Collect()` methods
    - _Requirements: 8.5_

  - [ ] 4.3 Write unit tests for collector initialization
    - Test collector creation with explicit version configuration
    - Test collector creation with automatic version detection
    - Test error handling when version detection fails
    - Verify API version metric is exposed correctly
    - _Requirements: 10.1, 10.2_

- [ ] 5. Create test data for all API versions
- [ ] 5.1 Add mock response files
  - Create `testdata/api-versions/` directory
  - Add `jobs-response-v3.json` for NetBackup 10.0 format
  - Add `jobs-response-v12.json` for NetBackup 10.5 format
  - Add `jobs-response-v13.json` for NetBackup 11.0 format
  - Add `storage-response-v3.json` for NetBackup 10.0 format
  - Add `storage-response-v12.json` for NetBackup 10.5 format
  - Add `storage-response-v13.json` for NetBackup 11.0 format
  - Add `error-406-response.json` for version not supported error
  - _Requirements: 10.2_

- [ ] 5.2 Verify response compatibility
  - Ensure all mock responses match actual NetBackup API responses
  - Verify common fields are present in all versions
  - Document version-specific optional fields
  - _Requirements: 3.4, 4.4, 8.3_

- [ ] 6. Update integration tests for multi-version support
- [ ] 6.1 Enhance existing integration tests
  - Update `internal/exporter/integration_test.go` to test all versions
  - Add test cases for version detection with mock servers
  - Add test cases for fallback behavior
  - Add test cases for configuration override
  - _Requirements: 10.3_

- [ ] 6.2 Add version-specific integration tests
  - Test jobs API compatibility across all versions
  - Test storage API compatibility across all versions
  - Verify metrics consistency across versions
  - Test authentication with all versions
  - _Requirements: 3.1, 3.2, 3.3, 4.1, 4.2, 4.3, 5.1, 5.2, 10.3_

- [ ] 6.3 Verify test coverage
  - Run coverage analysis on new code
  - Ensure ≥80% coverage for API client and parser modules
  - Document any uncovered edge cases
  - _Requirements: 10.5_

- [ ] 7. Update documentation
- [ ] 7.1 Update README
  - Add version support matrix (NetBackup 10.0-11.0)
  - Document apiVersion configuration parameter with examples
  - Add automatic version detection explanation
  - Update troubleshooting section with version-related issues
  - _Requirements: 2.5, 8.4_

- [ ] 7.2 Create migration guide
  - Document upgrade path from current implementation
  - Provide examples for different deployment scenarios
  - Add rollback procedure
  - Include troubleshooting for common version issues
  - _Requirements: 2.5_

- [ ] 7.3 Update configuration examples
  - Add example config with explicit version (13.0)
  - Add example config with automatic detection (no version field)
  - Add example config for backward compatibility (12.0, 3.0)
  - Document version detection behavior
  - _Requirements: 2.5_

- [ ] 8. Validate backward compatibility
- [ ] 8.1 Test with existing configurations
  - Test with config containing apiVersion: "12.0"
  - Test with config missing apiVersion field
  - Test with config containing apiVersion: "3.0"
  - Verify no breaking changes to existing deployments
  - _Requirements: 2.1, 2.4, 8.1, 8.2_

- [ ] 8.2 Verify metrics consistency
  - Compare metrics output across all API versions
  - Verify metric names remain unchanged
  - Verify label names and values remain consistent
  - Test existing Grafana dashboards with new exporter
  - _Requirements: 8.1, 8.2, 8.3_

- [ ] 8.3 Performance validation
  - Measure startup time with version detection
  - Measure startup time with explicit configuration
  - Verify no runtime performance degradation
  - Test connection reuse across versions
  - _Requirements: 9.1, 9.2, 9.4, 9.5_

- [ ] 9. Final integration and deployment preparation
- [ ] 9.1 End-to-end testing
  - Test complete workflow with NetBackup 11.0 (API 13.0)
  - Test complete workflow with NetBackup 10.5 (API 12.0)
  - Test fallback scenario (13.0 → 12.0 → 3.0)
  - Test error scenarios and recovery
  - _Requirements: All requirements_

- [ ] 9.2 Update build and deployment artifacts
  - Update Makefile if needed
  - Update Dockerfile if needed
  - Verify binary builds successfully
  - Test Docker image with all configurations
  - _Requirements: N/A (deployment)_

- [ ] 9.3 Prepare release notes
  - Document new features (multi-version support, auto-detection)
  - Document configuration changes
  - Document migration steps
  - List known issues or limitations
  - _Requirements: N/A (documentation)_
