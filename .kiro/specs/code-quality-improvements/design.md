# Design Document

## Overview

This design addresses code quality improvements for the NBU Exporter codebase, focusing on the OpenTelemetry integration and related components. The improvements maintain backward compatibility while enhancing maintainability, readability, and performance.

## Architecture

### Component Organization

```dir
internal/
├── telemetry/
│   ├── manager.go          # Existing telemetry manager
│   ├── attributes.go       # NEW: Centralized attribute constants
│   └── errors.go           # NEW: Error message templates
├── exporter/
│   ├── client.go           # Modified: Use centralized helpers
│   ├── netbackup.go        # Modified: Use centralized helpers
│   ├── prometheus.go       # Modified: Batch attributes
│   └── tracing.go          # NEW: Consolidated span helpers
└── models/
    └── Config.go           # Modified: Enhanced validation
```

## Components and Interfaces

### 1. Centralized Span Creation Helper

**File**: `internal/exporter/tracing.go`

```go
package exporter

import (
 "context"
 "go.opentelemetry.io/otel/trace"
)

// createSpan creates a new span with the given operation name and span kind.
// Returns the original context and nil span if tracer is nil (tracing disabled).
func createSpan(ctx context.Context, tracer trace.Tracer, operation string, kind trace.SpanKind) (context.Context, trace.Span) {
 if tracer == nil {
  return ctx, nil
 }
 return tracer.Start(ctx, operation, trace.WithSpanKind(kind))
}
```

**Rationale**: Eliminates 4 duplicate helper functions with identical implementations.

### 2. Attribute Constants

**File**: `internal/telemetry/attributes.go`

```go
package telemetry

// HTTP semantic convention attributes
const (
 AttrHTTPMethod                = "http.method"
 AttrHTTPURL                   = "http.url"
 AttrHTTPStatusCode            = "http.status_code"
 AttrHTTPRequestContentLength  = "http.request_content_length"
 AttrHTTPResponseContentLength = "http.response_content_length"
 AttrHTTPDurationMS            = "http.duration_ms"
)

// NetBackup-specific attributes
const (
 AttrNetBackupEndpoint       = "netbackup.endpoint"
 AttrNetBackupStorageUnits   = "netbackup.storage_units"
 AttrNetBackupAPIVersion     = "netbackup.api_version"
 AttrNetBackupTimeWindow     = "netbackup.time_window"
 AttrNetBackupStartTime      = "netbackup.start_time"
 AttrNetBackupTotalJobs      = "netbackup.total_jobs"
 AttrNetBackupTotalPages     = "netbackup.total_pages"
 AttrNetBackupPageOffset     = "netbackup.page_offset"
 AttrNetBackupPageNumber     = "netbackup.page_number"
 AttrNetBackupJobsInPage     = "netbackup.jobs_in_page"
)

// Scrape cycle attributes
const (
 AttrScrapeDurationMS          = "scrape.duration_ms"
 AttrScrapeStorageMetricsCount = "scrape.storage_metrics_count"
 AttrScrapeJobMetricsCount     = "scrape.job_metrics_count"
 AttrScrapeStatus              = "scrape.status"
)

// Error attributes
const (
 AttrError = "error"
)
```

**Rationale**: Prevents typos, enables IDE autocomplete, centralizes attribute naming.

### 3. Error Message Templates

**File**: `internal/telemetry/errors.go`

```go
package telemetry

// Error message templates for common scenarios
const (
 // ErrAPIVersionNotSupported is returned when the NetBackup server doesn't support the configured API version
 ErrAPIVersionNotSupportedTemplate = `API version %s is not supported by the NetBackup server (HTTP 406 Not Acceptable).

The server may be running a version of NetBackup that does not support API version %s.

Supported API versions:
  - 3.0  (NetBackup 10.0-10.4)
  - 12.0 (NetBackup 10.5)
  - 13.0 (NetBackup 11.0)

Troubleshooting steps:
1. Verify your NetBackup server version: bpgetconfig -g | grep VERSION
2. Update the 'apiVersion' field in config.yaml to match your server version
3. Or remove the 'apiVersion' field to enable automatic version detection

Example configuration:
  nbuserver:
    apiVersion: "12.0"  # For NetBackup 10.5
    # Or omit apiVersion for automatic detection

Request URL: %s`

 // ErrNonJSONResponse is returned when the server returns non-JSON content
 ErrNonJSONResponseTemplate = `NetBackup server returned non-JSON response (Content-Type: %s).

This usually indicates:
1. Wrong API endpoint URL (check 'uri' in config.yaml)
2. Authentication failure (verify API key is valid)
3. Server configuration issue (check NetBackup REST API is enabled)

Request URL: %s
Response preview: %s`
)
```

**Rationale**: Improves maintainability, makes error messages easier to update.

### 4. Enhanced Configuration Validation

**File**: `internal/models/Config.go`

Add new validation method:

```go
// validateOTelEndpoint validates the OpenTelemetry endpoint format
func (c *Config) validateOTelEndpoint() error {
 if c.OpenTelemetry.Endpoint == "" {
  return errors.New("OpenTelemetry endpoint is required when enabled")
 }
 
 // Validate endpoint format (host:port)
 parts := strings.Split(c.OpenTelemetry.Endpoint, ":")
 if len(parts) != 2 {
  return fmt.Errorf("invalid OpenTelemetry endpoint format: %s (expected host:port)", c.OpenTelemetry.Endpoint)
 }
 
 // Validate port
 port, err := strconv.Atoi(parts[1])
 if err != nil || port < 1 || port > 65535 {
  return fmt.Errorf("invalid OpenTelemetry endpoint port: %s", parts[1])
 }
 
 return nil
}
```

Update `Validate()` method to call `validateOTelEndpoint()`.

**Rationale**: Provides early feedback on configuration errors, prevents runtime failures.

### 5. Extracted Conditional Logic

**File**: `internal/exporter/client.go`

```go
// shouldPerformVersionDetection determines if automatic API version detection is needed
func shouldPerformVersionDetection(cfg *models.Config) bool {
 return cfg.NbuServer.APIVersion == ""
}

// isExplicitVersionConfigured checks if the user explicitly configured an API version
func isExplicitVersionConfigured(cfg *models.Config) bool {
 return cfg.NbuServer.APIVersion != "" && cfg.NbuServer.APIVersion != models.APIVersion130
}
```

**Rationale**: Makes business logic clearer, improves testability.

### 6. Batched Span Attributes

**Pattern**: Replace multiple `SetAttributes` calls with single batched call

**Before**:

```go
span.SetAttributes(attribute.String("key1", "value1"))
span.SetAttributes(attribute.Int("key2", 42))
span.SetAttributes(attribute.Float64("key3", 3.14))
```

**After**:

```go
if span != nil {
 attrs := []attribute.KeyValue{
  attribute.String("key1", "value1"),
  attribute.Int("key2", 42),
  attribute.Float64("key3", 3.14),
 }
 span.SetAttributes(attrs...)
}
```

**Rationale**: Reduces function call overhead, improves performance.

### 7. Test Function Naming Convention

**Pattern**: Remove underscores from test function names to comply with Go conventions and SonarCloud rules

**Before**:

```go
func TestNbuClient_GetHeaders(t *testing.T) { ... }
func TestNbuClient_FetchData_Success(t *testing.T) { ... }
func TestManager_Initialize_Disabled(t *testing.T) { ... }
```

**After**:

```go
func TestNbuClientGetHeaders(t *testing.T) { ... }
func TestNbuClientFetchDataSuccess(t *testing.T) { ... }
func TestManagerInitializeDisabled(t *testing.T) { ... }
```

**Rationale**:

- Follows official Go naming conventions
- Passes SonarCloud "Sonar Way" quality profile rule (go:S100)
- Maintains readability through descriptive subtest names with `t.Run()`
- Aligns with Go community standards

**Affected Files**:

- `internal/exporter/client_test.go`
- `internal/exporter/netbackup_test.go`
- `internal/exporter/prometheus_test.go`
- `internal/exporter/version_detector_test.go`
- `internal/exporter/otel_integration_test.go`
- `internal/exporter/otel_benchmark_test.go`
- `internal/telemetry/manager_test.go`
- `internal/models/Config_test.go`
- Any other test files with underscore-separated function names

**Implementation Strategy**:

1. Use find/replace with regex pattern: `func Test(\w+)_(\w+)` → `func Test$1$2`
2. Handle multi-underscore cases: `Test(\w+)_(\w+)_(\w+)` → `Test$1$2$3`
3. Verify all tests still pass after renaming
4. Update any test documentation that references old names

### 8. Eliminate Duplicate String Literals

**Issue**: SonarCloud rule go:S1192 flags string literals that are duplicated more than 3 times

**Pattern**: Extract repeated string literals to named constants

**Common Duplications to Address**:

```go
// Before: Repeated YAML struct tags
type NbuServer struct {
    Port               string `yaml:"port"`
    Scheme             string `yaml:"scheme"`
    URI                string `yaml:"uri"`
    Domain             string `yaml:"domain"`
    DomainType         string `yaml:"domainType"`
    Host               string `yaml:"host"`
    APIKey             string `yaml:"apiKey"`
    APIVersion         string `yaml:"apiVersion"`
    ContentType        string `yaml:"contentType"`
    InsecureSkipVerify bool   `yaml:"insecureSkipVerify"`
}

// After: Extract to constants (if used in multiple places)
const (
    YAMLTagPort               = "port"
    YAMLTagScheme             = "scheme"
    YAMLTagURI                = "uri"
    YAMLTagAPIKey             = "apiKey"
    YAMLTagAPIVersion         = "apiVersion"
    YAMLTagInsecureSkipVerify = "insecureSkipVerify"
)
```

**Common String Literals to Extract**:

1. **Test Configuration Strings**:

```go
// Before: Repeated in multiple test functions
cfg.NbuServer.APIVersion = "13.0"
cfg.NbuServer.APIKey = "test-key"

// After: Extract to test constants
const (
    testAPIVersion = "13.0"
    testAPIKey     = "test-key"
    testHost       = "localhost"
)
```

2. **Content Type Strings**:

```go
// Before: Repeated in multiple places
w.Header().Set("Content-Type", "application/json")

// After: Use existing constant or create new one
const ContentTypeJSON = "application/json"
w.Header().Set("Content-Type", ContentTypeJSON)
```

3. **Error Message Fragments**:

```go
// Before: Repeated error message parts
"API version %s is not supported"
"HTTP 406 Not Acceptable"

// After: Extract to constants (already done in errors.go)
```

**Rationale**:

- Passes SonarCloud go:S1192 rule
- Improves maintainability (change in one place)
- Reduces typo risk
- Makes refactoring easier

**Affected Files**:

- `internal/exporter/client_test.go` (test configuration strings)
- `internal/models/Config.go` (YAML tags if duplicated)
- `internal/exporter/client.go` (content type strings)
- Any file with repeated string literals (>3 occurrences)

**Implementation Notes**:

- Only extract strings that are truly duplicated (same semantic meaning)
- Don't extract strings that happen to be identical but have different purposes
- Use descriptive constant names that indicate purpose
- Group related constants together

## Data Models

No changes to existing data models. All improvements are internal refactorings.

## Error Handling

### Existing Pattern (Maintained)

- Wrap errors with context using `fmt.Errorf`
- Log errors before returning
- Graceful degradation where appropriate

### Enhanced Pattern

- Use error templates for complex messages
- Validate configuration early to fail fast
- Provide actionable error messages with troubleshooting steps

## Testing Strategy

### Unit Tests

- Test consolidated span creation helper with nil tracer
- Test attribute constant usage (compile-time check)
- Test endpoint validation with various formats
- Test extracted conditional functions

### Integration Tests

- Verify existing OpenTelemetry integration tests still pass
- Ensure no behavioral changes in span creation
- Validate error messages are correctly formatted

### Regression Tests

- Run full test suite to ensure no breaking changes
- Verify all existing functionality remains intact
- Check that metrics collection continues to work

## Migration Strategy

### Phase 1: Add New Components

1. Create `internal/telemetry/attributes.go`
2. Create `internal/telemetry/errors.go`
3. Create `internal/exporter/tracing.go`

### Phase 2: Update Existing Code

1. Update `internal/exporter/client.go` to use new helpers
2. Update `internal/exporter/netbackup.go` to use new helpers
3. Update `internal/exporter/prometheus.go` to batch attributes
4. Update `internal/models/Config.go` to enhance validation

### Phase 3: Remove Deprecated Code

1. Remove duplicate span creation helpers
2. Update all string literals to use constants
3. Update error messages to use templates

### Phase 4: Update Tests

1. Fix inconsistent error handling in tests
2. Add tests for new validation logic
3. Update integration tests if needed

## Performance Considerations

### Expected Improvements

- **Reduced allocations**: Batching attributes reduces function call overhead
- **Better inlining**: Simpler helper functions are more likely to be inlined
- **Compile-time checks**: Constants enable compile-time validation

### Benchmarking

- Run existing benchmarks to ensure no performance regression
- Compare before/after metrics for span creation overhead

## Backward Compatibility

### Guarantees

- All existing functionality remains unchanged
- No changes to public APIs
- No changes to configuration format (only validation enhanced)
- No changes to metric names or labels
- No changes to span structure or attributes

### Breaking Changes

**None** - This is a pure refactoring with no breaking changes.

## Documentation Updates

### Code Documentation

- Add complete godoc comments to new functions
- Update existing comments to reference new constants
- Add examples for complex validation logic

### README Updates

- No changes needed (internal refactoring only)

### CHANGELOG Updates

- Document improvements in "Changed" section
- Note performance optimizations
- Mention enhanced validation

## Design Decisions

### Decision 1: Consolidate vs. Keep Separate Helpers

**Choice**: Consolidate into single helper
**Rationale**: Reduces duplication, easier to maintain, no performance impact

### Decision 2: Constants vs. Enums for Attributes

**Choice**: Use string constants
**Rationale**: Matches OpenTelemetry conventions, simpler, no type conversion needed

### Decision 3: Template Strings vs. Builder Pattern for Errors

**Choice**: Use template strings with fmt.Sprintf
**Rationale**: Simpler, more idiomatic Go, easier to read

### Decision 4: Validation Location

**Choice**: Keep validation in Config.Validate()
**Rationale**: Centralized validation, fails fast at startup, consistent with existing pattern

### 9. Centralized Test Helpers

**File**: `internal/testutil/helpers.go`

**Purpose**: Consolidate duplicate test helper functions into a shared package with fluent builder interfaces.

**TestConfigBuilder Pattern**:

```go
package testutil

import (
    "strings"
    "github.com/fjacquet/nbu_exporter/internal/models"
)

// TestConfigBuilder provides a fluent interface for building test configurations
type TestConfigBuilder struct {
    cfg models.Config
}

func NewTestConfig() *TestConfigBuilder {
    return &TestConfigBuilder{
        cfg: models.Config{
            Server: models.ServerConfig{
                Host:             "localhost",
                Port:             "2112",
                URI:              "/metrics",
                ScrapingInterval: "5m",
                LogName:          "test.log",
            },
            NbuServer: models.NbuServerConfig{
                Scheme:             "http",
                InsecureSkipVerify: true,
            },
        },
    }
}

func (b *TestConfigBuilder) WithAPIVersion(version string) *TestConfigBuilder {
    b.cfg.NbuServer.APIVersion = version
    return b
}

func (b *TestConfigBuilder) WithServerURL(url string) *TestConfigBuilder {
    parts := strings.Split(strings.TrimPrefix(url, "http://"), ":")
    b.cfg.NbuServer.Host = parts[0]
    if len(parts) > 1 {
        b.cfg.NbuServer.Port = parts[1]
    }
    return b
}

func (b *TestConfigBuilder) WithAPIKey(key string) *TestConfigBuilder {
    b.cfg.NbuServer.APIKey = key
    return b
}

func (b *TestConfigBuilder) Build() models.Config {
    return b.cfg
}
```

**MockServerBuilder Pattern**:

```go
// MockServerBuilder provides a fluent interface for building mock HTTP servers
type MockServerBuilder struct {
    handlers map[string]http.HandlerFunc
}

func NewMockServer() *MockServerBuilder {
    return &MockServerBuilder{
        handlers: make(map[string]http.HandlerFunc),
    }
}

func (b *MockServerBuilder) WithJobsEndpoint(version string, response interface{}) *MockServerBuilder {
    b.handlers["/admin/jobs"] = func(w http.ResponseWriter, r *http.Request) {
        if validateAPIVersion(r, version) {
            writeJSONResponse(w, response)
        } else {
            w.WriteHeader(http.StatusNotAcceptable)
        }
    }
    return b
}

func (b *MockServerBuilder) WithStorageEndpoint(version string, response interface{}) *MockServerBuilder {
    b.handlers["/storage/storage-units"] = func(w http.ResponseWriter, r *http.Request) {
        if validateAPIVersion(r, version) {
            writeJSONResponse(w, response)
        } else {
            w.WriteHeader(http.StatusNotAcceptable)
        }
    }
    return b
}

func (b *MockServerBuilder) Build() *httptest.Server {
    return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if handler, ok := b.handlers[r.URL.Path]; ok {
            handler(w, r)
        } else {
            w.WriteHeader(http.StatusNotFound)
        }
    }))
}
```

**Helper Functions**:

```go
// LoadTestData loads test data from a file
func LoadTestData(t *testing.T, filename string) []byte {
    t.Helper()
    data, err := os.ReadFile(filename)
    require.NoError(t, err, "Failed to read test data file: %s", filename)
    return data
}

// validateAPIVersion checks if the request has the correct API version header
func validateAPIVersion(r *http.Request, expectedVersion string) bool {
    acceptHeader := r.Header.Get("Accept")
    return strings.Contains(acceptHeader, fmt.Sprintf("version=%s", expectedVersion))
}

// writeJSONResponse writes a JSON response to the ResponseWriter
func writeJSONResponse(w http.ResponseWriter, data interface{}) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(data)
}
```

**Usage Example**:

```go
// In test files
func TestSomething(t *testing.T) {
    // Build test config with fluent interface
    cfg := testutil.NewTestConfig().
        WithAPIVersion("13.0").
        WithServerURL(server.URL).
        WithAPIKey("test-key").
        Build()
    
    // Build mock server with fluent interface
    server := testutil.NewMockServer().
        WithJobsEndpoint("13.0", jobsResponse).
        WithStorageEndpoint("13.0", storageResponse).
        Build()
    defer server.Close()
    
    // Load test data
    data := testutil.LoadTestData(t, "testdata/jobs.json")
}
```

**Rationale**:

- Eliminates duplicate helper functions across test files
- Provides consistent, maintainable test infrastructure
- Fluent interface improves test readability
- Reduces cognitive load when writing new tests
- Follows builder pattern for complex object construction

### 10. Reduced Cognitive Complexity in Tests

**Issue**: `TestNewNbuCollectorAutomaticDetection` has cognitive complexity of 24 (threshold: 15)

**Solution**: Extract helper functions to reduce complexity

**Helper Functions**:

```go
// createVersionMockServer creates a mock server that responds to version detection requests
func createVersionMockServer(t *testing.T, responses map[string]int) *httptest.Server {
    return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        version := extractVersionFromHeader(r.Header.Get("Accept"))
        handleVersionResponse(w, version, responses)
    }))
}

// extractVersionFromHeader extracts the API version from the Accept header
func extractVersionFromHeader(acceptHeader string) string {
    for _, v := range []string{"13.0", "12.0", "3.0"} {
        if strings.Contains(acceptHeader, fmt.Sprintf("version=%s", v)) {
            return v
        }
    }
    return ""
}

// handleVersionResponse writes the appropriate response based on version and configured responses
func handleVersionResponse(w http.ResponseWriter, version string, responses map[string]int) {
    if statusCode, ok := responses[version]; ok {
        w.WriteHeader(statusCode)
        if statusCode == http.StatusOK {
            json.NewEncoder(w).Encode(map[string]interface{}{"data": []interface{}{}})
        }
    } else {
        w.WriteHeader(http.StatusNotAcceptable)
    }
}

// assertCollectorResult validates the collector creation result
func assertCollectorResult(t *testing.T, collector *NbuCollector, err error, tt testCase) {
    if tt.expectError {
        assert.Error(t, err)
        assert.Nil(t, collector)
    } else {
        require.NoError(t, err)
        require.NotNil(t, collector)
        assert.Equal(t, tt.expectedVersion, collector.cfg.NbuServer.APIVersion)
    }
}
```

**Refactored Test**:

```go
func TestNewNbuCollectorAutomaticDetection(t *testing.T) {
    tests := []struct {
        name            string
        serverResponses map[string]int
        expectedVersion string
        expectError     bool
    }{
        // test cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            server := createVersionMockServer(t, tt.serverResponses)
            defer server.Close()
            
            cfg := createTestConfigWithServer(server)
            collector, err := NewNbuCollector(cfg)
            
            assertCollectorResult(t, collector, err, tt)
        })
    }
}
```

**Rationale**:

- Reduces cognitive complexity from 24 to below 15
- Improves test readability and maintainability
- Makes test intent clearer
- Easier to debug test failures

### 11. Enhanced Error Context

**Pattern**: Add more context to error messages for better debugging

**Before**:

```go
if err := json.Unmarshal(resp.Body(), target); err != nil {
    return fmt.Errorf("failed to unmarshal JSON response: %w", err)
}
```

**After**:

```go
if err := json.Unmarshal(resp.Body(), target); err != nil {
    return fmt.Errorf("failed to unmarshal JSON response from %s (status: %d, content-type: %s): %w", 
        url, resp.StatusCode(), resp.Header().Get("Content-Type"), err)
}
```

**Rationale**:

- Provides actionable debugging information
- Includes request context (URL, status, content-type)
- Helps identify root cause faster
- Follows best practices for error messages

### 12. Package-Level Documentation

**Pattern**: Add comprehensive package documentation to all internal packages

**Example for testutil package**:

```go
// Package testutil provides shared testing utilities and constants for the NBU exporter.
//
// This package centralizes common test constants, helper functions, and mock builders
// to reduce duplication across test files and improve test maintainability.
//
// Key Components:
//   - Constants: Shared test values (API keys, endpoints, error messages)
//   - TestConfigBuilder: Fluent interface for building test configurations
//   - MockServerBuilder: Fluent interface for creating mock HTTP servers
//   - Helper Functions: Common test utilities (data loading, assertions)
//
// Usage Example:
//
// cfg := testutil.NewTestConfig().
//     WithAPIVersion("13.0").
//     WithServerURL(server.URL).
//     Build()
//
// server := testutil.NewMockServer().
//     WithJobsEndpoint("13.0", jobsResponse).
//     Build()
// defer server.Close()
package testutil
```

**Rationale**:

- Improves discoverability of package functionality
- Provides usage examples for developers
- Documents design patterns and conventions
- Follows Go documentation best practices

## Future Enhancements

### Potential Improvements (Out of Scope)

1. Builder pattern for NbuClient (if complexity increases)
2. Strategy pattern for samplers (if more strategies needed)
3. Reduce allocations in header carrier (micro-optimization)
4. Add structured logging with consistent fields

These are deferred as they provide marginal benefits and increase complexity.
