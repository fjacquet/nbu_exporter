# Testing Patterns

**Analysis Date:** 2026-01-22

## Test Framework

**Runner:**
- Framework: Go's standard `testing` package
- Config: No separate test configuration file; uses Go defaults
- Test discovery: Automatic file naming convention (`*_test.go`)

**Assertion Library:**
- Primary: `github.com/stretchr/testify/assert` and `testify/require`
- Custom helpers: `internal/testutil` package with `AssertNoError()`, `AssertContains()`, `AssertEqual()`, `AssertError()`

**Run Commands:**
```bash
go test ./...              # Run all tests
go test -race ./...        # Run all tests with race detection
go test ./... -cover       # Run all tests with coverage report
go test -coverprofile=coverage.out ./...  # Generate coverage profile
go tool cover -html=coverage.out  # View HTML coverage report
make test                  # Run all tests (via Makefile)
make test-coverage         # Run tests with coverage analysis
make sure                  # Run fmt, test, build, and golangci-lint
```

## Test File Organization

**Location:**
- Co-located with source: test files in same package as code being tested
- Pattern: `filename.go` paired with `filename_test.go` in same directory
- Example: `internal/exporter/prometheus.go` → `internal/exporter/prometheus_test.go`

**Naming:**
- Files: `<source>_test.go` (e.g., `Config_test.go`, `client_test.go`, `prometheus_test.go`)
- Functions: `Test<FunctionName>_<Scenario>` or `Test<FunctionName><Scenario>` (PascalCase, underscores separate concerns)
- Examples:
  - `TestNewNbuCollectorExplicitVersion`
  - `TestNewNbuCollectorAutomaticDetection`
  - `TestNewNbuCollectorDetectionFailure`
  - `TestNbuCollector_APIVersionMetric`
  - `TestStorageMetricsCollection`
  - `TestJobMetricsCollection`

**Structure:**
```
internal/
├── exporter/
│   ├── prometheus.go
│   ├── prometheus_test.go
│   ├── client.go
│   ├── client_test.go
│   ├── version_detector.go
│   ├── version_detector_test.go
│   ├── end_to_end_test.go
│   ├── integration_test.go
│   └── test_common.go                # Shared test constants
├── models/
│   ├── Config.go
│   ├── Config_test.go
│   ├── Jobs.go
│   ├── Jobs_test.go
│   └── test_common.go                # Shared test constants
├── testutil/
│   ├── helpers.go                    # MockServerBuilder, test helpers
│   ├── helpers_test.go               # Tests for testutil itself
│   └── constants.go                  # Shared test constants
└── utils/
    ├── file.go
    ├── file_test.go
    └── date_test.go
```

## Test Structure

**Suite Organization:**
- Table-driven tests as primary pattern for multiple scenarios
- Subtests using `t.Run()` for grouping related test cases
- Each test case in the table defines: name, inputs, expected outputs, and expected error status

**Table-Driven Pattern (from `prometheus_test.go`):**
```go
func TestNewNbuCollectorExplicitVersion(t *testing.T) {
	tests := []struct {
		name       string
		apiVersion string
	}{
		{
			name:       "API version 13.0",
			apiVersion: models.APIVersion130,
		},
		{
			name:       "API version 12.0",
			apiVersion: models.APIVersion120,
		},
		// More test cases...
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			server := httptest.NewTLSServer(...)
			defer server.Close()

			cfg := /* test config */

			// Execute
			collector, err := NewNbuCollector(cfg)

			// Assert
			require.NoError(t, err)
			require.NotNil(t, collector)
			assert.Equal(t, tt.apiVersion, collector.cfg.NbuServer.APIVersion)
		})
	}
}
```

**Patterns:**
- Setup: Create mock servers, test data, configuration
- Defer cleanup: `defer server.Close()` for resources
- Execute: Call function being tested
- Assert: Verify results using testify assertions

**Assertions:**
- `require.*`: Fails test immediately if assertion fails (use for setup/critical checks)
- `assert.*`: Records failure but continues test (use for multiple checks in same test)
- Examples:
  ```go
  require.NoError(t, err, "Collector creation should succeed")
  require.NotNil(t, collector, "Collector should not be nil")
  assert.Equal(t, expectedVersion, collector.cfg.NbuServer.APIVersion)
  assert.True(t, foundAPIVersionMetric, "API version metric should be collected")
  assert.Contains(t, err.Error(), "expected text")
  ```

## Mocking

**Framework:**
- `net/http/httptest` for HTTP server mocking (built-in)
- Custom `MockServerBuilder` from `internal/testutil` for fluent mock configuration

**Patterns (from `testutil/helpers.go`):**
```go
// Fluent builder pattern for HTTP mocks
server := testutil.NewMockServer().
    WithJobsEndpoint("13.0", jobsResponse).
    WithStorageEndpoint("13.0", storageResponse).
    Build()
defer server.Close()

// Version detection mocking
server := testutil.NewMockServer().
    WithVersionDetection(map[string]bool{
        "13.0": true,
        "12.0": false,
    }).
    Build()

// Error response mocking
server := testutil.NewMockServer().
    WithErrorResponse("/api/jobs", http.StatusUnauthorized).
    Build()
```

**What to Mock:**
- External HTTP services (NetBackup API via httptest)
- File system operations (use actual test files or in-memory equivalents)
- OpenTelemetry tracer provider (set to nil for tests without tracing)

**What NOT to Mock:**
- Configuration validation logic (test with real Config structs)
- Error wrapping and propagation (test with actual errors)
- Internal helper functions (test directly, don't mock)
- Package-level utilities from `internal/testutil` (use as-is for consistency)

## Fixtures and Factories

**Test Data:**
- Files: Raw test responses stored in project (e.g., `testdata/jobs-response.json`, referenced in integration tests)
- Loading: `testutil.LoadTestData(t, filename)` for file-based fixtures
- Helper functions: `createConfigWithAPIVersion()`, `createTestConfig()`, `createVersionMockServer()` in test files
- Constants: Centralized in `internal/testutil/constants.go` and aliased in `test_common.go` per package

**Example from `integration_test.go`:**
```go
// Helper function for test config creation
func createTestConfig(serverURL string, apiVersion string) models.Config {
    cfg := models.Config{}
    cfg.NbuServer.Scheme = "https"
    cfg.NbuServer.Host = extractHostFromURL(serverURL)
    cfg.NbuServer.Port = extractPortFromURL(serverURL)
    cfg.NbuServer.APIKey = testAPIKey
    cfg.NbuServer.APIVersion = apiVersion
    return cfg
}

// Using test data from testutil
func TestStorageMetricsCollection(t *testing.T) {
    storageResponse := loadStorageTestData(t)
    // ...
}
```

**Location:**
- Shared constants: `internal/testutil/constants.go` (API keys, endpoints, versions)
- Package-specific aliases: `test_common.go` in each package that needs constants (e.g., `internal/exporter/test_common.go`)
- Factories: Inline in test files or in `testutil` for reusable builders

**From `testutil/constants.go`:**
```go
const (
    // HTTP headers
    ContentTypeHeader   = "Content-Type"
    AcceptHeader        = "Accept"
    AuthorizationHeader = "Authorization"

    // Common test values
    TestAPIKey = "testkey123"

    // Test endpoints
    TestPathAdminJobs    = "/api/v4/admin/jobs"
    TestPathStorageUnits = "/api/v4/storage/storage-units"
    TestPathMetrics      = "/metrics"

    // API versions
    APIVersion30  = "3.0"
    APIVersion120 = "12.0"
    APIVersion130 = "13.0"
)
```

## Coverage

**Requirements:**
- Overall target: 70% total coverage (enforced via `.testcoverage.yml`)
- File/package targets: None enforced (set to 0)
- Gaps: Some complex integration points, OTEL tracing initialization

**View Coverage:**
```bash
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html  # Open in browser
go tool cover -func=coverage.out | grep total:      # Summary line
```

**Coverage Report Location:** Generated to `coverage.out` and `coverage.html` in project root

## Test Types

**Unit Tests:**
- Scope: Individual functions and methods in isolation
- Approach: Use table-driven tests with various inputs
- Examples: `TestConfigSetDefaults()`, `TestNewNbuClientWithVersionDetection()`, `TestNbuCollectorDescribe()`
- Mocking: Mock external HTTP services, accept nil tracer for OTEL
- Coverage: Majority of test files (15+ test files)

**Integration Tests:**
- Scope: Multiple components working together (collector, client, API responses)
- Approach: Use mock HTTP servers to simulate NetBackup API responses
- Files: `internal/exporter/integration_test.go`, `internal/exporter/end_to_end_test.go`
- Pattern: Load realistic test data, verify metric collection and transformation
- Example: `TestStorageMetricsCollection()`, `TestJobMetricsCollection()`

**E2E Tests:**
- Not used: No full end-to-end tests requiring real NetBackup server
- Alternative: Integration tests with httptest servers provide sufficient coverage
- File: `end_to_end_test.go` contains integration-level tests (not full E2E)

**Backward Compatibility Tests:**
- File: `internal/exporter/backward_compatibility_test.go`
- Purpose: Verify multiple API versions (3.0, 12.0, 13.0) work correctly
- Pattern: Separate test cases for each API version's response format

**Performance Tests:**
- File: `internal/exporter/performance_test.go`
- Benchmark tests: `BenchmarkXXX` function naming (Go standard)
- Run with: `go test -bench=. ./internal/exporter/`

## Common Patterns

**Helper Functions (using `t.Helper()`):**
- Marked with `t.Helper()` to report errors at caller's location, not helper location
- Examples from `helpers.go`:
  ```go
  func testAcceptedVersion(t *testing.T, server *httptest.Server) {
      t.Helper()
      // Test implementation
  }
  ```
- Used for: test setup, assertions, data loading

**Async Testing:**
```go
// From prometheus_test.go - using goroutine with channel
metricChan := make(chan prometheus.Metric, 10)
go func() {
    collector.Collect(metricChan)
    close(metricChan)
}()

// Verify metrics collected
for metric := range metricChan {
    desc := metric.Desc()
    // assertions
}
```

**Timeout Testing:**
```go
// Wait with timeout
done := make(chan bool)
go func() {
    defer func() { done <- true }()
    collector.Collect(ch)
}()

select {
case <-done:
    // Success
case <-time.After(5 * time.Second):
    t.Error("Collect() timed out")
}
```

**Error Testing:**
```go
// Expecting error
collector, err := NewNbuCollector(cfg)
if tt.expectError {
    assert.Error(t, err, "Should fail when versions unsupported")
    assert.Contains(t, err.Error(), "expected error text")
    assert.Nil(t, collector, "Collector should be nil on error")
}

// Expecting success
require.NoError(t, err)
require.NotNil(t, collector)
```

**Skipped Tests:**
```go
func TestNewNbuCollectorAutomaticDetection(t *testing.T) {
    t.Skip("Skipping automatic detection test - covered by version_detector_test.go")
    // Test body (skipped)
}
```

## Test Dependencies

**Direct:**
- `github.com/stretchr/testify` - Assertions
- `net/http/httptest` - HTTP mock servers
- Go standard `testing` package

**Internal:**
- `internal/testutil` - Mock builders, shared constants, helper functions
- `internal/models` - Test configuration creation

## Test Execution

**From Makefile:**
```bash
make test              # go test ./...
make test-coverage     # go test ./... -coverprofile=coverage.out -covermode=atomic
make sure              # go fmt ./..., go test ./..., go build ./..., golangci-lint run
```

**With Options:**
```bash
go test ./internal/exporter -run TestVersionDetection    # Specific test
go test -race ./...                                       # Race detector
go test -v ./...                                          # Verbose output
go test ./... -timeout 10m                                # Custom timeout
```

---

*Testing analysis: 2026-01-22*
