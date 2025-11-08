# Requirements Document

## Introduction

This document specifies the requirements for upgrading the NBU Exporter to support Veritas NetBackup API version 10.5. The upgrade involves analyzing the new API specifications, identifying breaking changes and new features, updating the codebase to maintain compatibility, and implementing version management to support multiple API versions if needed.

## Glossary

- **NBU Exporter**: The Prometheus exporter application that collects metrics from Veritas NetBackup
- **NetBackup API**: The RESTful API provided by Veritas NetBackup for programmatic access
- **API Version 10.5**: The new version of the NetBackup API being integrated
- **Storage Unit**: A NetBackup resource representing backup storage capacity
- **Job Metrics**: Statistics about backup jobs including status, size, and policy information
- **Prometheus Collector**: The component that implements the Prometheus metrics collection interface
- **HTTP Client**: The Resty-based client used for API communication
- **Configuration Model**: The data structure representing application and API connection settings

## Requirements

### Requirement 1

**User Story:** As a DevOps engineer, I want to analyze the NetBackup 10.5 API specifications, so that I can identify changes that impact the current implementation

#### Acceptance Criteria

1. WHEN the API specification files are read, THE NBU Exporter SHALL identify all endpoints currently used by the application
2. THE NBU Exporter SHALL document any changes to existing endpoint paths, parameters, or response schemas
3. THE NBU Exporter SHALL identify new authentication or authorization requirements in version 10.5
4. THE NBU Exporter SHALL list any deprecated endpoints or fields that are currently in use
5. THE NBU Exporter SHALL document new API features that could enhance monitoring capabilities

### Requirement 2

**User Story:** As a backup administrator, I want the exporter to continue collecting storage metrics using the 10.5 API, so that I can monitor storage capacity without interruption

#### Acceptance Criteria

1. WHEN the storage endpoint is called, THE NBU Exporter SHALL retrieve storage unit data using the 10.5 API format
2. THE NBU Exporter SHALL parse storage unit attributes including name, type, free capacity, and used capacity
3. IF the API response schema has changed, THEN THE NBU Exporter SHALL adapt the parsing logic accordingly
4. THE NBU Exporter SHALL maintain backward compatibility with existing Prometheus metric names and labels
5. WHEN tape storage units are encountered, THE NBU Exporter SHALL continue to exclude them from metrics

### Requirement 3

**User Story:** As a backup administrator, I want the exporter to continue collecting job metrics using the 10.5 API, so that I can monitor backup job success rates and data transfer volumes

#### Acceptance Criteria

1. WHEN the jobs endpoint is called, THE NBU Exporter SHALL retrieve job data using the 10.5 API format
2. THE NBU Exporter SHALL handle pagination for job listings according to 10.5 API specifications
3. THE NBU Exporter SHALL parse job attributes including type, policy, status, and bytes transferred
4. IF the API response schema has changed, THEN THE NBU Exporter SHALL update the data models accordingly
5. THE NBU Exporter SHALL maintain time-based filtering for jobs within the configured scraping interval

### Requirement 4

**User Story:** As a developer, I want the codebase to support API version management, so that the exporter can handle different NetBackup versions gracefully

#### Acceptance Criteria

1. THE NBU Exporter SHALL include API version information in the configuration model
2. WHEN making API requests, THE NBU Exporter SHALL include the appropriate version in the Accept header
3. THE NBU Exporter SHALL validate that the connected NetBackup server supports the required API version
4. IF version-specific behavior is needed, THEN THE NBU Exporter SHALL implement version detection logic
5. THE NBU Exporter SHALL log the detected API version during initialization

### Requirement 5

**User Story:** As a developer, I want updated data models that reflect the 10.5 API schema, so that the application correctly parses API responses

#### Acceptance Criteria

1. THE NBU Exporter SHALL update the Storage model to match the 10.5 API response schema
2. THE NBU Exporter SHALL update the Jobs model to match the 10.5 API response schema
3. WHEN new fields are added to API responses, THE NBU Exporter SHALL include them in the data models
4. WHEN fields are removed from API responses, THE NBU Exporter SHALL remove them from the data models
5. THE NBU Exporter SHALL maintain Go struct tags for proper JSON unmarshaling

### Requirement 6

**User Story:** As a DevOps engineer, I want comprehensive documentation of the API changes, so that I can understand the impact on deployment and configuration

#### Acceptance Criteria

1. THE NBU Exporter SHALL provide an impact analysis document listing all code changes required
2. THE NBU Exporter SHALL document any configuration changes needed for 10.5 compatibility
3. THE NBU Exporter SHALL document any breaking changes that affect existing deployments
4. THE NBU Exporter SHALL update the README with 10.5 API version requirements
5. THE NBU Exporter SHALL provide migration guidance for users upgrading from previous versions

### Requirement 7

**User Story:** As a quality assurance engineer, I want updated tests that validate 10.5 API compatibility, so that I can ensure the exporter works correctly with the new version

#### Acceptance Criteria

1. THE NBU Exporter SHALL update test fixtures to reflect 10.5 API response formats
2. THE NBU Exporter SHALL verify that all existing tests pass with the updated implementation
3. WHEN new API features are implemented, THE NBU Exporter SHALL include corresponding test coverage
4. THE NBU Exporter SHALL validate error handling for 10.5-specific error responses
5. THE NBU Exporter SHALL include integration tests that verify end-to-end functionality with 10.5 API

### Requirement 8

**User Story:** As a security administrator, I want the exporter to maintain secure API communication with NetBackup 10.5, so that credentials and data remain protected

#### Acceptance Criteria

1. THE NBU Exporter SHALL support API key authentication as specified in the 10.5 API
2. THE NBU Exporter SHALL support JWT token authentication as specified in the 10.5 API
3. WHEN TLS verification is enabled, THE NBU Exporter SHALL validate server certificates according to 10.5 requirements
4. THE NBU Exporter SHALL mask API keys in all log output
5. THE NBU Exporter SHALL use HTTPS for all API communications unless explicitly configured otherwise
