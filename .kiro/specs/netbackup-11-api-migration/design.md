# Design Document: NetBackup Multi-Version API Support

## Overview

This design document outlines the technical approach for enhancing the NetBackup Prometheus Exporter to support multiple NetBackup API versions (3.0, 12.0, and 13.0) with automatic version detection and fallback capabilities. The solution maintains backward compatibility while enabling seamless operation across NetBackup 10.0, 10.5, and 11.0 environments.

### Goals

- Support NetBackup API versions 3.0, 12.0, and 13.0 simultaneously
- Implement automatic version detection with intelligent fallback
- Maintain backward compatibility with existing configurations
- Preserve existing Prometheus metric names and labels
- Minimize code duplication across version-specific implementations
- Provide clear error messages for version-related issues

### Non-Goals

- Support for NetBackup versions older than 10.0 (API < 3.0)
- Dynamic API version switching during runtime (version is determined at startup)
- Automatic migration of configuration files
- Support for mixed-version NetBackup clusters

## Architecture

### High-Level Design

The solution follows a strategy pattern approach where version-specific behavior is abstracted behind common interfaces. The architecture consists of three main layers:

1. **Configuration Layer**: Handles API version configuration and validation
2. **Detection Layer**: Implements version detection with fallback logic
3. **Client Layer**: Manages version-specific HTTP headers and request formatting

```
┌─────────────────────────────────────────────────────────┐
│                    Application Layer                     │
│                  (Prometheus Collector)                  │
└────────────────────┬────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────┐
│                   Client Layer                           │
│  ┌──────────────────────────────────────────────────┐  │
│  │           NbuClient (HTTP Client)                 │  │
│  │  - Version-aware header construction              │  │
│  │  - Request/response handling                      │  │
│  │  - Error interpretation                           │  │
│  └──────────────────────────────────────────────────┘  │
└────────────────────┬────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────┐
│                Detection Layer                           │
│  ┌──────────────────────────────────────────────────┐  │
│  │        APIVersionDetector                         │  │
│  │  - Probe API endpoints                            │  │
│  │  - Fallback logic (13.0 → 12.0 → 3.0)           │  │
│  │  - Version validation                             │  │
│  └──────────────────────────────────────────────────┘  │
└────────────────────┬────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────┐
│              Configuration Layer                         │
│  ┌──────────────────────────────────────────────────┐  │
│  │              Config Model                         │  │
│  │  - API version field                              │  │
│  │  - Version format validation                      │  │
│  │  - Default values                                 │  │
│  └──────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
```

### Version Detection Flow

```
┌─────────────┐
│   Startup   │
└──────┬──────┘
       │
       ▼
┌─────────────────────┐
│ Config has          │     Yes    ┌──────────────────┐
│ apiVersion field?   ├───────────►│ Use configured   │
└──────┬──────────────┘            │ version          │
       │ No                        └────────┬─────────┘
       ▼                                    │
┌─────────────────────┐                    │
│ Try API v13.0       │                    │
│ (NetBackup 11.0)    │                    │
└──────┬──────────────┘                    │
       │                                    │
       ▼                                    │
┌─────────────────────┐                    │
│ Success?            │     Yes            │
└──────┬──────────────┘────────────────────┤
       │ No (HTTP 406)                     │
       ▼                                    │
┌─────────────────────┐                    │
│ Try API v12.0       │                    │
│ (NetBackup 10.5)    │                    │
└──────┬──────────────┘                    │
       │                                    │
       ▼                                    │
┌─────────────────────┐                    │
│ Success?            │     Yes            │
└──────┬──────────────┘────────────────────┤
       │ No (HTTP 406)                     │
       ▼                                    │
┌─────────────────────┐                    │
│ Try API v3.0        │                    │
│ (NetBackup 10.0)    │                    │
└──────┬──────────────┘                    │
       │                                    │
       ▼                                    │
┌─────────────────────┐                    │
│ Success?            │     Yes            │
└──────┬──────────────┘────────────────────┤
       │ No                                 │
       ▼                                    │
┌─────────────────────┐                    │
│ Log error and exit  │                    │
└─────────────────────┘                    │
                                            │
                                            ▼
                                   ┌────────────────┐
                                   │ Log detected   │
                                   │ version & start│
                                   └────────────────┘
```

## Components and Interfaces

### 1. Configuration Model Enhancement

**File**: `internal/models/Config.go`

**Changes**:

- API version field already exists and is validated
- Default value is currently "12.0" - will be updated to "13.0"
- Validation regex already supports X.Y format
- Add supported versions constant for validation

```go
// Supported API versions
const (
    APIVersion30  = "3.0"   // NetBackup 10.0-10.4
    APIVersion120 = "12.0"  // NetBackup 10.5
    APIVersion130 = "13.0"  // NetBackup 11.0
)

var SupportedAPIVersions = []string{APIVersion130, APIVersion120, APIVersion30}
```

**Rationale**: Centralizes version constants for easy maintenance and provides a single source of truth for supported versions.

### 2. API Version Detector

**New File**: `internal/exporter/version_detector.go`

**Purpose**: Implements automatic API version detection with fallback logic.

**Interface**:

```go
// APIVersionDetector handles automatic detection of supported NetBackup API versions
type APIVersionDetector struct {
    client *NbuClient
    cfg    *models.Config
}

// DetectVersion attempts to detect the highest supported API version
// Returns the detected version string or an error if no version works
func (d *APIVersionDetector) DetectVersion(ctx context.Context) (string, error)

// tryVersion tests a specific API version by making a lightweight API call
// Returns true if the version is supported, false otherwise
func (d *APIVersionDetector) tryVersion(ctx context.Context, version string) bool
```

**Implementation Details**:

- Uses a lightweight endpoint (`/admin/jobs?page[limit]=1`) for version probing
- Implements exponential backoff for transient failures (network issues)
- Distinguishes between version incompatibility (HTTP 406) and other errors
- Logs each version attempt for troubleshooting
- Returns the first working version in descending order

**Error Handling**:

- HTTP 406: Version not supported, try next version
- HTTP 401: Authentication issue, fail immediately (not a version problem)
- Network errors: Retry with backoff, then try next version
- Other HTTP errors: Log and try next version

### 3. Enhanced HTTP Client

**File**: `internal/exporter/client.go`

**Changes**:

- `getHeaders()` method already constructs version-aware Accept headers
- `DetectAPIVersion()` method exists but needs enhancement
- Add retry logic with exponential backoff
- Improve error messages for version-related failures

**Current Implementation** (already correct):

```go
func (c *NbuClient) getHeaders() map[string]string {
    acceptHeader := fmt.Sprintf("application/vnd.netbackup+json;version=%s",
        c.cfg.NbuServer.APIVersion)
    return map[string]string{
        HeaderAccept:        acceptHeader,
        HeaderAuthorization: c.cfg.NbuServer.APIKey,
    }
}
```

**Enhancement**: Add retry configuration

```go
type RetryConfig struct {
    MaxAttempts     int
    InitialDelay    time.Duration
    MaxDelay        time.Duration
    BackoffFactor   float64
}

var DefaultRetryConfig = RetryConfig{
    MaxAttempts:   3,
    InitialDelay:  1 * time.Second,
    MaxDelay:      10 * time.Second,
    BackoffFactor: 2.0,
}
```

### 4. Collector Initialization

**File**: `internal/exporter/prometheus.go`

**Changes**:

- Add version detection during collector initialization
- Log detected version at INFO level
- Handle version detection failures gracefully

**Initialization Flow**:

```go
func NewNbuCollector(cfg models.Config) (*NbuCollector, error) {
    client := NewNbuClient(cfg)

    // Perform version detection if not explicitly configured
    if cfg.NbuServer.APIVersion == "" {
        detector := NewAPIVersionDetector(client, &cfg)
        version, err := detector.DetectVersion(context.Background())
        if err != nil {
            return nil, fmt.Errorf("API version detection failed: %w", err)
        }
        cfg.NbuServer.APIVersion = version
        log.Infof("Detected NetBackup API version: %s", version)
    } else {
        log.Infof("Using configured NetBackup API version: %s", cfg.NbuServer.APIVersion)
    }

    return &NbuCollector{
        client: client,
        cfg:    cfg,
    }, nil
}
```

## Data Models

### API Response Compatibility

All three API versions (3.0, 12.0, 13.0) use the same JSON:API response structure for jobs and storage endpoints. The existing data models in `internal/models/` already handle this correctly:

**Jobs Model** (`internal/models/Jobs.go`):

- Core attributes are consistent across all versions
- Optional fields (e.g., `kilobytesDataTransferred` in 12.0+) use Go's zero-value semantics
- No changes required

**Storage Model** (`internal/models/Storage.go`, `internal/models/Storages.go`):

- Core capacity fields are consistent across all versions
- Optional fields (e.g., `storageCategory`, replication flags in 12.0+) are already optional
- No changes required

### Version Mapping

| NetBackup Version | API Version | Release Date | Status         |
| ----------------- | ----------- | ------------ | -------------- |
| 10.0 - 10.4       | 3.0         | 2021-2023    | Legacy Support |
| 10.5              | 12.0        | 2024         | Current        |
| 11.0              | 13.0        | 2025         | Latest         |

## Error Handling

### Error Categories

1. **Version Detection Errors**
   - All versions fail: Clear error message with troubleshooting steps
   - Network errors: Distinguish from version incompatibility
   - Authentication errors: Fail fast, don't try other versions

2. **Runtime Errors**
   - HTTP 406 during operation: Log error, suggest version reconfiguration
   - Other HTTP errors: Existing error handling remains unchanged

### Error Messages

**Version Detection Failure**:

```
ERROR: Failed to detect compatible NetBackup API version.
Attempted versions: 13.0, 12.0, 3.0
Last error: HTTP 406 Not Acceptable

Possible causes:
1. NetBackup server is running a version older than 10.0
2. Network connectivity issues
3. API endpoint is not accessible
4. Authentication credentials are invalid

Troubleshooting:
- Verify NetBackup server version: bpgetconfig -g | grep VERSION
- Check network connectivity to https://nbu-host:1556
- Verify API key is valid and not expired
- Try manually specifying apiVersion in config.yaml
```

**Configured Version Not Supported**:

```
ERROR: Configured API version 13.0 is not supported by the NetBackup server (HTTP 406 Not Acceptable).

The server may be running an older version of NetBackup.
Detected NetBackup version: 10.5 (supports API version 12.0)

Solution:
Update config.yaml:
  nbuserver:
    apiVersion: "12.0"

Or remove the apiVersion field to enable automatic detection.
```

## Testing Strategy

### Unit Tests

1. **Configuration Validation Tests** (`internal/models/Config_test.go`)
   - Test API version format validation
   - Test default value assignment
   - Test supported version constants
   - Test invalid version formats

2. **Version Detector Tests** (`internal/exporter/version_detector_test.go`)
   - Mock HTTP responses for each version
   - Test fallback logic (13.0 → 12.0 → 3.0)
   - Test retry logic with exponential backoff
   - Test error handling for different HTTP status codes
   - Test early exit on authentication errors

3. **Client Tests** (`internal/exporter/client_test.go`)
   - Test header construction for each version
   - Test error message formatting
   - Test retry behavior

### Integration Tests

1. **Multi-Version Integration Test** (`internal/exporter/integration_test.go`)
   - Test against NetBackup 10.0 (API 3.0) - if available
   - Test against NetBackup 10.5 (API 12.0) - if available
   - Test against NetBackup 11.0 (API 13.0) - primary target
   - Verify metrics consistency across versions

2. **Version Detection Integration Test**
   - Test automatic detection with mock servers
   - Test fallback behavior
   - Test configuration override

### Test Data

**Mock Responses** (`testdata/api-versions/`):

- `jobs-response-v3.json` - NetBackup 10.0 format
- `jobs-response-v12.json` - NetBackup 10.5 format
- `jobs-response-v13.json` - NetBackup 11.0 format
- `storage-response-v3.json` - NetBackup 10.0 format
- `storage-response-v12.json` - NetBackup 10.5 format
- `storage-response-v13.json` - NetBackup 11.0 format
- `error-406-response.json` - Version not supported error

### Coverage Goals

- Unit test coverage: ≥ 80% for new code
- Integration test coverage: All supported versions
- Error path coverage: All error scenarios tested

## Migration Path

### For Existing Deployments

**Scenario 1: NetBackup 10.5 with explicit apiVersion**

```yaml
# Current config.yaml
nbuserver:
  apiVersion: "12.0" # Explicitly set
```

**Action**: No changes required. Exporter will use configured version.

**Scenario 2: NetBackup 10.5 without apiVersion**

```yaml
# Current config.yaml
nbuserver:
  # apiVersion not specified
```

**Action**: No changes required. Exporter will detect version 12.0 automatically.

**Scenario 3: Upgrading to NetBackup 11.0**

```yaml
# Option A: Let exporter detect new version
nbuserver:
    # Remove or comment out apiVersion
    # apiVersion: "12.0"

# Option B: Explicitly set new version
nbuserver:
    apiVersion: "13.0"
```

**Action**: Either remove apiVersion field or update to "13.0".

### Rollback Procedure

If issues occur after deployment:

1. Stop the exporter
2. Restore previous binary
3. Restore previous configuration
4. Restart the exporter
5. Report issue with logs

## Performance Considerations

### Version Detection Overhead

- **Startup Time**: Version detection adds 1-3 seconds to startup (one lightweight API call per version attempt)
- **Mitigation**: Detection only occurs once at startup, not during metric collection
- **Optimization**: Explicit configuration bypasses detection entirely

### Runtime Performance

- **No Impact**: Version-specific behavior is limited to HTTP header construction
- **Memory**: No additional memory overhead (version string is already stored in config)
- **CPU**: Negligible overhead for string formatting in headers

### Network Efficiency

- **Connection Reuse**: Existing connection pooling remains unchanged
- **Request Volume**: No additional API calls during normal operation
- **Bandwidth**: Header size increase is negligible (~20 bytes per request)

## Security Considerations

### API Key Handling

- API keys remain in configuration (no changes)
- Masked logging already implemented
- Version detection uses same authentication as normal operation

### TLS Configuration

- TLS settings remain unchanged
- Version detection respects `insecureSkipVerify` setting
- No additional certificate validation required

### Error Information Disclosure

- Error messages avoid exposing sensitive information
- API keys are never logged in plain text
- Version detection errors provide troubleshooting guidance without revealing internal details

## Deployment Strategy

### Phase 1: Development and Testing

1. Implement version detector with unit tests
2. Update configuration model and validation
3. Enhance client error handling
4. Create integration tests with mock servers

### Phase 2: Internal Testing

1. Test against NetBackup 11.0 environment
2. Test against NetBackup 10.5 environment (backward compatibility)
3. Test against NetBackup 10.0 environment (if available)
4. Verify metrics consistency across versions

### Phase 3: Documentation

1. Update README with version support matrix
2. Document configuration options
3. Create migration guide
4. Update troubleshooting section

### Phase 4: Release

1. Tag release with semantic versioning
2. Publish release notes
3. Update Docker images
4. Notify users of new version

## Monitoring and Observability

### Metrics

Add new metric to track API version in use:

```
# HELP nbu_api_version The NetBackup API version currently in use
# TYPE nbu_api_version gauge
nbu_api_version{version="13.0"} 1
```

### Logging

**Startup Logs**:

```
INFO[0000] Starting NBU Exporter
INFO[0001] Attempting API version detection
DEBUG[0001] Trying API version 13.0
INFO[0002] Detected NetBackup API version: 13.0
INFO[0002] Successfully connected to NetBackup API
```

**Version Detection Failure**:

```
WARN[0001] API version 13.0 not supported (HTTP 406), trying next version
WARN[0002] API version 12.0 not supported (HTTP 406), trying next version
INFO[0003] Successfully detected API version: 3.0
```

## Future Enhancements

### Potential Improvements

1. **Dynamic Version Switching**: Detect version changes during runtime (requires significant refactoring)
2. **Version-Specific Metrics**: Expose new metrics available in newer API versions
3. **Multi-Server Support**: Support different API versions for different NetBackup servers
4. **Version Caching**: Cache detected version to speed up restarts
5. **Health Check Endpoint**: Expose API version in health check response

### API Evolution

As new NetBackup versions are released:

1. Add new version constant to `SupportedAPIVersions`
2. Update version detector to try new version first
3. Add test data for new version
4. Update documentation

## References

- [NetBackup 11.0 API Documentation](https://sort.veritas.com/public/documents/nbu/11.0/windowsandunix/productguides/html/getting-started/)
- [NetBackup 10.5 API Documentation](https://sort.veritas.com/public/documents/nbu/10.5/)
- [Existing Migration Guide](docs/api-10.5-migration.md)
- [Requirements Document](.kiro/specs/netbackup-11-api-migration/requirements.md)
