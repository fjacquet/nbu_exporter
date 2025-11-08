# Requirements Document

## Introduction

This document outlines the requirements for enhancing the NetBackup Prometheus Exporter to support multiple NetBackup API versions (3.0, 12.0, and 13.0) with automatic version detection and switching capabilities. The system will maintain backward compatibility with existing deployments while enabling seamless operation across NetBackup 10.0, 10.5, and 11.0 environments.

## Glossary

- **NBU_Exporter**: The NetBackup Prometheus Exporter application that collects metrics from NetBackup APIs
- **NetBackup_API**: The RESTful API provided by Veritas NetBackup for programmatic access
- **API_Version**: The version number of the NetBackup API (e.g., 3.0, 12.0, 13.0)
- **Master_Server**: The NetBackup master server that hosts the API endpoints
- **Storage_Unit**: A NetBackup storage resource for backup data
- **Job_Metrics**: Statistics and information about NetBackup backup/restore jobs
- **Version_Negotiation**: The process of determining which API version to use based on server capabilities
- **Backward_Compatibility**: The ability to work with older NetBackup versions while supporting newer ones

## Requirements

### Requirement 1: API Version Detection

**User Story:** As a system administrator, I want the exporter to automatically detect the NetBackup server version, so that I can deploy the same exporter binary across different NetBackup environments without manual configuration.

#### Acceptance Criteria

1. WHEN the NBU_Exporter starts, THE NBU_Exporter SHALL query the Master_Server to determine the available API_Version
2. THE NBU_Exporter SHALL attempt version detection in descending order (13.0, 12.0, 3.0) until a supported version is found
3. WHEN a supported API_Version is detected, THE NBU_Exporter SHALL use that version for all subsequent requests
4. THE NBU_Exporter SHALL log the detected API_Version at startup with severity level INFO
5. IF the version detection fails for all supported versions, THEN THE NBU_Exporter SHALL retry with exponential backoff up to 3 attempts before failing

### Requirement 2: Configuration Management

**User Story:** As a system administrator, I want to optionally specify the API version in configuration, so that I can override automatic detection when needed for testing or troubleshooting.

#### Acceptance Criteria

1. WHERE the configuration file contains an apiVersion field, THE NBU_Exporter SHALL use the specified API_Version without performing version detection
2. THE NBU_Exporter SHALL validate that the specified API_Version is supported (3.0, 12.0, or 13.0)
3. IF an unsupported API_Version is specified, THEN THE NBU_Exporter SHALL log an error and exit with a non-zero status code
4. WHERE no apiVersion is specified in configuration, THE NBU_Exporter SHALL perform automatic version detection
5. THE NBU_Exporter SHALL document the apiVersion configuration parameter in the README with examples for all supported versions

### Requirement 3: Jobs API Compatibility

**User Story:** As a monitoring engineer, I want job metrics to be collected correctly from both API versions, so that I can monitor backup job performance regardless of NetBackup version.

#### Acceptance Criteria

1. WHEN using API version 3.0, THE NBU_Exporter SHALL request job data using the content-type "application/vnd.netbackup+json;version=3.0"
2. WHEN using API version 12.0, THE NBU_Exporter SHALL request job data using the content-type "application/vnd.netbackup+json;version=12.0"
3. WHEN using API version 13.0, THE NBU_Exporter SHALL request job data using the content-type "application/vnd.netbackup+json;version=13.0"
4. THE NBU_Exporter SHALL parse job response attributes that are common to all API versions (jobId, jobType, policyType, status, kilobytesTransferred, startTime, endTime)
5. WHERE newer API versions provide additional job attributes, THE NBU_Exporter SHALL collect and expose those attributes as optional metrics
6. THE NBU_Exporter SHALL maintain the same Prometheus metric names and labels for job metrics across all API versions

### Requirement 4: Storage API Compatibility

**User Story:** As a storage administrator, I want storage unit capacity metrics to be collected from both API versions, so that I can monitor storage utilization across all NetBackup environments.

#### Acceptance Criteria

1. WHEN using API version 3.0, THE NBU_Exporter SHALL request storage data using the endpoint "/storage/storage-units" with version 3.0
2. WHEN using API version 12.0, THE NBU_Exporter SHALL request storage data using the endpoint "/storage/storage-units" with version 12.0
3. WHEN using API version 13.0, THE NBU_Exporter SHALL request storage data using the endpoint "/storage/storage-units" with version 13.0
4. THE NBU_Exporter SHALL extract freeCapacityBytes, usedCapacityBytes, and totalCapacityBytes from storage unit responses across all API versions
5. THE NBU_Exporter SHALL filter out tape storage units from metrics collection in all API versions
6. THE NBU_Exporter SHALL expose storage metrics with labels for storage unit name and storage server type

### Requirement 5: Authentication Compatibility

**User Story:** As a security administrator, I want the exporter to use the same authentication method across API versions, so that I don't need to reconfigure authentication when upgrading NetBackup.

#### Acceptance Criteria

1. THE NBU_Exporter SHALL support API key authentication for all API versions (3.0, 12.0, and 13.0)
2. THE NBU_Exporter SHALL include the API key in the Authorization header for all API requests
3. IF authentication fails with status code 401, THEN THE NBU_Exporter SHALL log the error and retry after the configured scraping interval
4. THE NBU_Exporter SHALL support JWT token authentication as an alternative to API keys for all versions
5. WHERE the configuration specifies an API key, THE NBU_Exporter SHALL use that key without attempting JWT login

### Requirement 6: Error Handling and Resilience

**User Story:** As a DevOps engineer, I want the exporter to handle API errors gracefully, so that temporary network issues or API unavailability don't cause monitoring gaps.

#### Acceptance Criteria

1. WHEN an API request fails with a network error, THE NBU_Exporter SHALL log the error and continue with the next scraping cycle
2. WHEN an API request returns status code 500, THE NBU_Exporter SHALL log the error and retry the request once before continuing
3. THE NBU_Exporter SHALL expose a metric indicating the last successful scrape timestamp for each API endpoint
4. THE NBU_Exporter SHALL expose a metric indicating the total number of API errors by endpoint and error type
5. IF the Master_Server becomes unreachable, THEN THE NBU_Exporter SHALL continue running and retry connections at each scraping interval

### Requirement 7: Pagination Handling

**User Story:** As a monitoring engineer, I want the exporter to handle large result sets correctly, so that all jobs are included in metrics regardless of the total count.

#### Acceptance Criteria

1. WHEN the jobs API returns paginated results, THE NBU_Exporter SHALL follow pagination links to retrieve all job records
2. THE NBU_Exporter SHALL use the page[limit] query parameter with a value of 100 for efficient data retrieval
3. THE NBU_Exporter SHALL use the page[offset] query parameter to iterate through all pages of results
4. THE NBU_Exporter SHALL detect the end of pagination when the response contains no data or when offset equals last
5. THE NBU_Exporter SHALL implement a maximum page limit of 1000 pages to prevent infinite loops

### Requirement 8: Metrics Consistency

**User Story:** As a Grafana dashboard user, I want metric names and labels to remain consistent, so that my existing dashboards continue to work after the exporter is upgraded.

#### Acceptance Criteria

1. THE NBU_Exporter SHALL maintain the same Prometheus metric names for job and storage metrics across API versions
2. THE NBU_Exporter SHALL maintain the same label names and values for existing metrics
3. WHERE new attributes are available in API version 13.0, THE NBU_Exporter SHALL expose them as new optional labels
4. THE NBU_Exporter SHALL document any new metrics or labels in the README with version annotations
5. THE NBU_Exporter SHALL provide a metric indicating which API_Version is currently in use

### Requirement 9: Performance and Efficiency

**User Story:** As a system administrator, I want the exporter to minimize API load on the NetBackup server, so that monitoring doesn't impact backup operations.

#### Acceptance Criteria

1. THE NBU_Exporter SHALL reuse HTTP connections for multiple API requests within a scraping cycle
2. THE NBU_Exporter SHALL implement a configurable timeout for API requests with a default of 60 seconds
3. THE NBU_Exporter SHALL use efficient filtering to retrieve only jobs within the configured time window
4. THE NBU_Exporter SHALL complete a full scraping cycle within 80% of the configured scraping interval
5. THE NBU_Exporter SHALL limit concurrent API requests to a maximum of 5 to prevent overwhelming the Master_Server

### Requirement 10: Testing and Validation

**User Story:** As a developer, I want comprehensive tests to validate API compatibility, so that I can confidently deploy the exporter across different NetBackup versions.

#### Acceptance Criteria

1. THE NBU_Exporter SHALL include unit tests for version detection logic with mock API responses for all supported versions
2. THE NBU_Exporter SHALL include unit tests for parsing job and storage responses from all API versions (3.0, 12.0, and 13.0)
3. THE NBU_Exporter SHALL include integration tests that can run against API version 3.0, 12.0, and 13.0 servers
4. THE NBU_Exporter SHALL include tests for error handling and retry logic across version detection scenarios
5. THE NBU_Exporter SHALL achieve at least 80% code coverage for the API client and parser modules
