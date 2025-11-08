# Implementation Plan

- [x] 1. Update configuration model to support API version
  - Add `APIVersion` field to `NbuServer` struct in `internal/models/Config.go`
  - Set default value to `"12.0"` for NetBackup 10.5
  - Add validation to ensure API version format is valid
  - Update `Validate()` method to check API version field
  - _Requirements: 4.1, 4.2_

- [x] 2. Update HTTP client to include API version in headers
  - Modify `FetchData()` method in `internal/exporter/client.go` to construct versioned Accept header
  - Format header as `application/vnd.netbackup+json;version=<apiVersion>`
  - Use API version from configuration
  - Ensure Authorization header remains unchanged
  - _Requirements: 4.2, 8.1, 8.2_

- [x] 3. Add optional fields to storage data model
  - Update `Storages` struct in `internal/models/Storages.go`
  - Add `StorageCategory` field (string, optional)
  - Add replication-related fields: `ReplicationCapable`, `ReplicationSourceCapable`, `ReplicationTargetCapable` (bool, optional)
  - Add snapshot-related fields: `Snapshot`, `Mirror`, `Independent`, `Primary` (bool, optional)
  - Add `ScaleOutEnabled`, `WormCapable`, `UseWorm` fields (bool, optional)
  - Use `omitempty` JSON tags for all new optional fields
  - _Requirements: 5.1, 5.3_

- [x] 4. Add optional fields to jobs data model
  - Update `Jobs` struct in `internal/models/Jobs.go`
  - Add `KilobytesDataTransferred` field (int, optional) to distinguish from estimated `KilobytesTransferred`
  - Use `omitempty` JSON tag for new field
  - _Requirements: 5.2, 5.3_

- [x] 5. Update configuration file and documentation
  - Add `apiVersion: "12.0"` to example `config.yaml`
  - Add comment explaining the API version field
  - Update README.md to specify NetBackup 10.5+ requirement
  - Document the API version configuration option
  - _Requirements: 6.2, 6.4_

- [x] 6. Create API version detection utility
  - Implement `DetectAPIVersion()` method in `internal/exporter/client.go`
  - Log detected API version during client initialization
  - Handle cases where version detection fails gracefully
  - _Requirements: 4.3, 4.5_

- [x] 7. Update error handling for version-specific responses
  - Add handling for 406 (Not Acceptable) errors in `FetchData()` method
  - Log clear error messages when API version is not supported
  - Provide guidance in error messages about version compatibility
  - _Requirements: 4.4_

- [x] 8. Create test fixtures for API 10.5 responses
  - Create `testdata/api-10.5/` directory
  - Add `storage-units-response.json` with sample 10.5 storage API response
  - Add `jobs-response.json` with sample 10.5 jobs API response
  - Add `error-responses.json` with sample 10.5 error responses
  - Include responses with new optional fields populated
  - _Requirements: 7.1_

- [x] 9. Update unit tests for configuration model
  - Test API version field parsing from YAML
  - Test default value assignment when API version is omitted
  - Test validation of API version format
  - Test backward compatibility with configs missing API version field
  - _Requirements: 7.2_

- [x] 10. Update unit tests for HTTP client
  - Mock API responses using 10.5 format with version 12.0
  - Test Accept header construction with API version
  - Test error handling for 406 (Not Acceptable) responses
  - Verify Authorization header remains unchanged
  - _Requirements: 7.2_

- [x] 11. Update unit tests for data models
  - Test JSON unmarshaling with 10.5 response samples
  - Test parsing of new optional fields in storage model
  - Test parsing of new optional fields in jobs model
  - Test handling when optional fields are absent
  - Verify pagination parsing still works correctly
  - _Requirements: 7.1, 7.2_

- [x] 12. Create integration tests for end-to-end flow
  - Test complete scrape cycle with mocked 10.5 API responses
  - Verify Prometheus metrics output format unchanged
  - Test storage metrics collection with 10.5 responses
  - Test job metrics collection with 10.5 responses
  - Verify filtering and pagination work correctly
  - _Requirements: 7.5_

- [x] 13. Create migration documentation
  - Create `docs/api-10.5-migration.md` guide
  - Document upgrade steps for existing deployments
  - List configuration changes required
  - Provide troubleshooting guide for common issues
  - Include version compatibility matrix
  - _Requirements: 6.1, 6.3, 6.5_

- [x] 14. Update CHANGELOG with API 10.5 support
  - Add entry for API 10.5 support
  - Document new configuration field
  - Note backward compatibility maintained
  - List new optional fields available in data models
  - _Requirements: 6.1_

- [x] 15. Verify deployment and rollback procedures
  - Document deployment steps for upgrading to 10.5 support
  - Test that existing configs work with default API version
  - Verify rollback procedure if issues occur
  - Ensure no breaking changes for existing users
  - _Requirements: 6.3_
