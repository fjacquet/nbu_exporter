# Design Document

## Overview

This design document outlines the approach for upgrading the NBU Exporter to support Veritas NetBackup API version 10.5. The upgrade maintains backward compatibility with existing Prometheus metrics while adapting to API changes in version 10.5. The design focuses on minimal code changes, version awareness, and maintaining the existing architecture patterns.

### Key Design Goals

1. **API Compatibility**: Ensure all existing endpoints work with version 10.5
2. **Version Awareness**: Add API version management to the configuration and client
3. **Schema Alignment**: Update data models to match 10.5 response structures
4. **Backward Compatibility**: Maintain existing Prometheus metric names and labels
5. **Minimal Disruption**: Leverage existing architecture patterns and minimize refactoring

## Architecture

### Current Architecture Overview

```
┌─────────────────┐
│   main.go       │
│  (CLI + HTTP)   │
└────────┬────────┘
         │
         ├──────────────────┐
         │                  │
┌────────▼────────┐  ┌──────▼──────────┐
│  NbuCollector   │  │   NbuClient     │
│  (Prometheus)   │  │   (Resty HTTP)  │
└────────┬────────┘  └──────┬──────────┘
         │                  │
         │                  │
┌────────▼────────┐  ┌──────▼──────────┐
│  FetchStorage   │  │  FetchAllJobs   │
│  FetchJobDetails│  │                 │
└────────┬────────┘  └──────┬──────────┘
         │                  │
         └──────────┬───────┘
                    │
         ┌──────────▼──────────┐
         │   NetBackup API     │
         │   (version 10.5)    │
         └─────────────────────┘
```

### API Version 10.5 Changes

Based on analysis of the API specifications:

**Storage API (`/storage/storage-units`)**

- ✅ Endpoint path: **UNCHANGED** - `/storage/storage-units`
- ✅ Pagination: **COMPATIBLE** - `page[limit]`, `page[offset]` still supported
- ✅ Required fields: **ALL PRESENT** - `name`, `storageType`, `storageServerType`, `freeCapacityBytes`, `usedCapacityBytes`
- 🆕 New optional fields: `storageCategory`, `replicationCapable`, `wormCapable`, etc.
- 📝 Header change: Must include `Accept: application/vnd.netbackup+json;version=12.0`

**Jobs API (`/admin/jobs`)**

- ✅ Endpoint path: **UNCHANGED** - `/admin/jobs`
- ✅ Filter syntax: **COMPATIBLE** - OData format `endTime gt <timestamp>` works
- ✅ Required fields: **ALL PRESENT** - `jobId`, `jobType`, `policyType`, `status`, `kilobytesTransferred`
- ⚠️ Pagination: Spec shows cursor-based (`next` as string), but offset-based still supported
- 🆕 New field: `kilobytesDataTransferred` (actual vs estimated data)
- 📝 Header change: Must include `Accept: application/vnd.netbackup+json;version=12.0`

**Authentication**

- ✅ API Key: **UNCHANGED** - `Authorization: <API_KEY>` format maintained
- ✅ JWT: Supported (not currently used, no changes needed)

**Breaking Changes Assessment**: ❌ **NONE** - All current code is compatible with 10.5

**Detailed Comparison**: See `api-comparison.md` for field-by-field analysis

## Components and Interfaces

### 1. Configuration Model Updates

**File**: `internal/models/Config.go`

Add API version configuration:

```go
type Config struct {
    Server struct {
        // ... existing fields
    } `yaml:"server"`

    NbuServer struct {
        // ... existing fields
        APIVersion string `yaml:"apiVersion"` // NEW: e.g., "12.0"
    } `yaml:"nbuserver"`
}
```

**Default Value**: `"12.0"` (NetBackup 10.5 API version)

**Validation**: Ensure API version follows semantic versioning pattern

### 2. HTTP Client Updates

**File**: `internal/exporter/client.go`

Update request headers to include API version:

```go
func (c *NbuClient) FetchData(ctx context.Context, url string, result interface{}) error {
    contentType := fmt.Sprintf("application/vnd.netbackup+json;version=%s", c.cfg.NbuServer.APIVersion)

    resp, err := c.client.R().
        SetContext(ctx).
        SetHeader("Accept", contentType).
        SetHeader("Authorization", c.cfg.NbuServer.APIKey).
        SetResult(result).
        Get(url)
    // ... error handling
}
```

### 3. Data Model Verification

**Files**: `internal/models/Storage.go`, `internal/models/Jobs.go`

Review and validate existing models against 10.5 API schemas:

**Storage Model** (no changes expected):

```go
type Storages struct {
    Data []struct {
        Attributes struct {
            Name               string `json:"name"`
            StorageType        string `json:"storageType"`
            StorageServerType  string `json:"storageServerType"`
            FreeCapacityBytes  int64  `json:"freeCapacityBytes"`
            UsedCapacityBytes  int64  `json:"usedCapacityBytes"`
            // ... other fields
        } `json:"attributes"`
    } `json:"data"`
    Meta struct {
        Pagination struct {
            Count  int `json:"count"`
            Offset int `json:"offset"`
            Limit  int `json:"limit"`
            First  int `json:"first"`
            Last   int `json:"last"`
            Prev   int `json:"prev"`
            Next   int `json:"next"`
        } `json:"pagination"`
    } `json:"meta"`
}
```

**Jobs Model** (no changes expected):

```go
type Jobs struct {
    Data []struct {
        Attributes struct {
            JobID                int    `json:"jobId"`
            JobType              string `json:"jobType"`
            PolicyType           string `json:"policyType"`
            Status               int    `json:"status"`
            KilobytesTransferred int64  `json:"kilobytesTransferred"`
            // ... other fields
        } `json:"attributes"`
    } `json:"data"`
    Meta struct {
        Pagination struct {
            Count  int `json:"count"`
            Offset int `json:"offset"`
            Limit  int `json:"limit"`
            First  int `json:"first"`
            Last   int `json:"last"`
            Prev   int `json:"prev"`
            Next   int `json:"next"`
        } `json:"pagination"`
    } `json:"meta"`
}
```

### 4. API Version Detection (Optional Enhancement)

**File**: `internal/exporter/client.go`

Add version detection capability:

```go
func (c *NbuClient) DetectAPIVersion(ctx context.Context) (string, error) {
    // Call a lightweight endpoint to detect version
    // Parse response headers or version info
    // Return detected version string
}
```

This can be used during initialization to log the server's API version.

## Data Models

### Configuration Schema

**config.yaml** updates:

```yaml
server:
  host: "localhost"
  port: "2112"
  uri: "/metrics"
  scrapingInterval: "5m"
  logName: "nbu_exporter.log"

nbuserver:
  host: "netbackup-master.example.com"
  port: "1556"
  scheme: "https"
  uri: "/netbackup"
  apiVersion: "12.0" # NEW: API version for NetBackup 10.5
  apiKey: "your-api-key-here"
  contentType: "application/json"
  insecureSkipVerify: false
```

### API Request/Response Flow

```mermaid
sequenceDiagram
    participant Prometheus
    participant NbuCollector
    participant NbuClient
    participant NetBackup API

    Prometheus->>NbuCollector: Scrape /metrics
    NbuCollector->>NbuClient: FetchStorage()
    NbuClient->>NetBackup API: GET /storage/storage-units<br/>Accept: application/vnd.netbackup+json;version=12.0
    NetBackup API-->>NbuClient: JSON Response (v12.0)
    NbuClient-->>NbuCollector: Parsed Storage Data
    NbuCollector->>NbuClient: FetchAllJobs()
    NbuClient->>NetBackup API: GET /admin/jobs<br/>Accept: application/vnd.netbackup+json;version=12.0
    NetBackup API-->>NbuClient: JSON Response (v12.0)
    NbuClient-->>NbuCollector: Parsed Job Data
    NbuCollector-->>Prometheus: Prometheus Metrics
```

## Error Handling

### API Version Mismatch

**Scenario**: NetBackup server doesn't support requested API version

**Handling**:

1. Log warning with detected vs. requested version
2. Attempt request with configured version
3. If 406 (Not Acceptable) error, log detailed error message
4. Provide clear guidance in error message about version compatibility

### Schema Parsing Errors

**Scenario**: API response doesn't match expected schema

**Handling**:

1. Log the raw JSON response (truncated for security)
2. Return descriptive error with field name that failed to parse
3. Continue with partial data if possible
4. Increment error metric for monitoring

### Authentication Errors

**Scenario**: API key invalid or expired

**Handling**:

1. Return 401/403 errors immediately
2. Log masked API key for debugging
3. Provide clear error message about authentication failure
4. Do not retry authentication errors

## Testing Strategy

### Unit Tests

**Test Files**: `*_test.go` in respective packages

1. **Configuration Validation**
   - Test API version parsing and validation
   - Test default values for new fields
   - Test backward compatibility with configs missing API version

2. **HTTP Client**
   - Mock API responses with version 12.0 format
   - Test header construction with API version
   - Test error handling for version-related errors

3. **Data Model Parsing**
   - Test JSON unmarshaling with 10.5 response samples
   - Test handling of optional fields
   - Test pagination parsing

### Integration Tests

1. **End-to-End Flow**
   - Test complete scrape cycle with mocked 10.5 API
   - Verify Prometheus metrics output format
   - Test with sample data from actual 10.5 API responses

2. **Error Scenarios**
   - Test with invalid API version
   - Test with malformed responses
   - Test with authentication failures

### Test Data

**Location**: `testdata/api-10.5/`

Create sample response files:

- `storage-units-response.json` - Sample storage API response
- `jobs-response.json` - Sample jobs API response
- `error-responses.json` - Sample error responses

## Impact Analysis

### Breaking Changes

**None Expected**: The API structure remains consistent between versions. The primary change is the version number in the Accept header.

### Configuration Changes

**Required**:

- Add `apiVersion: "12.0"` to `nbuserver` section in config.yaml
- Update documentation to reflect NetBackup 10.5 requirement

**Optional**:

- Existing configs without `apiVersion` will default to "12.0"

### Code Changes Summary

| File                          | Change Type  | Description                           |
| ----------------------------- | ------------ | ------------------------------------- |
| `internal/models/Config.go`   | Addition     | Add APIVersion field and validation   |
| `internal/exporter/client.go` | Modification | Update Accept header with API version |
| `config.yaml`                 | Addition     | Add apiVersion configuration          |
| `README.md`                   | Update       | Document NetBackup 10.5 requirement   |
| `testdata/`                   | Addition     | Add 10.5 API response samples         |

### Deployment Impact

**Backward Compatibility**:

- Existing deployments will continue to work with default version "12.0"
- No database migrations required
- No Prometheus metric changes

**Upgrade Path**:

1. Update binary
2. Optionally add `apiVersion` to config (defaults to 12.0)
3. Restart service
4. Verify metrics collection

### Performance Impact

**Minimal**: No performance changes expected. The API version header adds negligible overhead.

## Migration Guide

### For Users Upgrading from Previous Versions

1. **Verify NetBackup Version**: Ensure NetBackup server is version 10.5 or later
2. **Update Configuration** (optional): Add `apiVersion: "12.0"` to config.yaml
3. **Update Binary**: Deploy new nbu_exporter binary
4. **Restart Service**: Restart the exporter service
5. **Verify Metrics**: Check that metrics are being collected successfully

### Rollback Plan

If issues occur:

1. Revert to previous binary version
2. Remove `apiVersion` from config if added
3. Restart service

## Documentation Updates

### Files to Update

1. **README.md**
   - Update prerequisites to specify NetBackup 10.5+
   - Update API version reference
   - Add migration section

2. **config.yaml (example)**
   - Add apiVersion field with comment

3. **CHANGELOG.md**
   - Document API 10.5 support
   - List configuration changes
   - Note backward compatibility

4. **docs/api-upgrade-guide.md** (new)
   - Detailed upgrade instructions
   - Troubleshooting guide
   - Version compatibility matrix

## Version Compatibility Matrix

| NBU Exporter Version | NetBackup API Version | NetBackup Version |
| -------------------- | --------------------- | ----------------- |
| 1.x.x (current)      | 10.x                  | 10.0 - 10.4       |
| 2.0.0 (new)          | 12.0                  | 10.5+             |

## Security Considerations

1. **API Key Handling**: No changes to existing secure handling
2. **TLS Configuration**: No changes to existing TLS verification
3. **Logging**: Ensure API version doesn't leak sensitive information
4. **Configuration**: API version is not sensitive data

## Future Enhancements

1. **Multi-Version Support**: Support multiple API versions simultaneously
2. **Automatic Version Detection**: Detect and adapt to server API version
3. **Version-Specific Features**: Leverage new 10.5 features (if any)
4. **Deprecation Warnings**: Warn about deprecated API features
