# Code Analysis: NBU Exporter - Improvement Recommendations

**Analysis Date**: November 8, 2025  
**Analyzed By**: Kiro AI Assistant

Based on comprehensive analysis of the codebase, here are specific, actionable improvements organized by category.

---

## 1. CRITICAL: Missing Test Coverage

**Issue**: No test files exist in the project, violating the 80% coverage requirement from quality standards.

**Impact**: High risk of regressions, difficult to refactor safely, production readiness compromised.

**Recommendations**:

- Create unit tests for all packages:
  - `internal/exporter/netbackup_test.go` - Test pagination, job fetching, storage fetching
  - `internal/exporter/client_test.go` - Test HTTP client with mocked responses
  - `internal/exporter/prometheus_test.go` - Test collector implementation
  - `internal/models/Config_test.go` - Test validation logic
  - `internal/utils/date_test.go` - Test date conversion

**Example test structure**:

```go
// internal/exporter/netbackup_test.go
func TestFetchStorage(t *testing.T) {
    tests := []struct {
        name          string
        mockResponse  models.Storages
        expectedCount int
        expectError   bool
    }{
        // table-driven test cases
    }
    // Use httptest.Server for mocking
}
```

---

## 2. Code Smells & Anti-Patterns

### 2.1 String-Based Metric Keys (Medium Priority)

**Location**: `internal/exporter/prometheus.go` lines 71-110

**Issue**: Using `strings.Split(key, "|")` to parse metric labels is fragile and error-prone.

**Current Code**:

```go
for key, value := range storageMetrics {
    labels := strings.Split(key, "|")  // Brittle!
    ch <- prometheus.MustNewConstMetric(...)
}
```

**Recommendation**: Use the `Labels()` method from `metrics.go` that already exists:

```go
// Create a parallel map structure
type StorageMetric struct {
    Key   StorageMetricKey
    Value float64
}

// In FetchStorage, return []StorageMetric instead of map[string]float64
// Then in Collect:
for _, metric := range storageMetrics {
    ch <- prometheus.MustNewConstMetric(
        c.nbuDiskSize,
        prometheus.GaugeValue,
        metric.Value,
        metric.Key.Labels()...,
    )
}
```

### 2.2 Global Variables in Logging Package

**Location**: `internal/logging/logging.go` lines 11-14

**Issue**: Global mutable state makes testing difficult and creates initialization order dependencies.

**Current Code**:

```go
var currentTime = time.Now()
var version = currentTime.Format("2006-01-02T15:04:05")
var programName = os.Args[0] + "-" + version
```

**Recommendation**: Remove globals, pass context through function parameters:

```go
type Logger struct {
    programName string
    version     string
}

func NewLogger(name, version string) *Logger {
    return &Logger{
        programName: name,
        version:     version,
    }
}

func (l *Logger) Info(msg string) {
    log.WithFields(log.Fields{"job": l.programName}).Info(msg)
}
```

### 2.3 Unused Logging Functions

**Location**: `internal/logging/logging.go`

**Issue**: Functions like `LogInfo`, `LogPanic`, `LogPanicMsg`, `HandleError`, `LogError` are defined but never used in the codebase. The code uses `log.Infof`, `log.Errorf` directly instead.

**Recommendation**: Either use these functions consistently or remove them to reduce maintenance burden.

---

## 3. Design Pattern Improvements

### 3.1 Missing Interface for NbuClient

**Location**: `internal/exporter/client.go`

**Issue**: `NbuClient` is a concrete type, making it difficult to mock for testing.

**Recommendation**: Define an interface:

```go
// internal/exporter/client.go
type HTTPClient interface {
    FetchData(ctx context.Context, url string, target interface{}) error
}

// Ensure NbuClient implements it
var _ HTTPClient = (*NbuClient)(nil)
```

Then use the interface in function signatures:

```go
func FetchStorage(ctx context.Context, client HTTPClient, storageMetrics map[string]float64) error
```

### 3.2 Improve Error Context

**Location**: Multiple files

**Issue**: Some errors lack sufficient context for debugging.

**Example in `netbackup.go` line 35**:

```go
if err := client.FetchData(ctx, url, &storages); err != nil {
    return fmt.Errorf("failed to fetch storage data: %w", err)
}
```

**Recommendation**: Add more context:

```go
if err := client.FetchData(ctx, url, &storages); err != nil {
    return fmt.Errorf("failed to fetch storage data from %s: %w", url, err)
}
```

---

## 4. Readability & Maintainability

### 4.1 Magic Numbers as Constants

**Location**: `internal/exporter/netbackup.go`

**Good**: Already using constants for most values. However, consider grouping related constants:

```go
// API pagination settings
const (
    defaultPageLimit = "100"
    defaultPageOffset = "0"
)

// Storage types
const (
    storageTypeTape = "Tape"
    storageTypeDisk = "Disk"
)

// Size types for metrics
const (
    sizeTypeFree = "free"
    sizeTypeUsed = "used"
)
```

### 4.2 Improve Function Documentation

**Location**: All exported functions

**Issue**: Some functions lack comprehensive godoc comments.

**Example - Current**:

```go
// FetchStorage retrieves and processes storage unit information from the NetBackup API.
func FetchStorage(...)
```

**Improved**:

```go
// FetchStorage retrieves storage unit information from the NetBackup API and populates
// the provided storageMetrics map with free and used capacity in bytes.
// Tape storage units are automatically filtered out.
//
// The function uses pagination with a limit of 100 items per request.
// Metrics are keyed by storage unit name, type, and size category (free/used).
//
// Returns an error if the API request fails or response parsing fails.
func FetchStorage(...)
```

### 4.3 Extract Configuration Validation

**Location**: `internal/models/Config.go` lines 28-67

**Issue**: The `Validate()` method is long and does multiple things.

**Recommendation**: Extract validation logic into smaller functions:

```go
func (c *Config) Validate() error {
    if err := c.validateServer(); err != nil {
        return err
    }
    if err := c.validateNBUServer(); err != nil {
        return err
    }
    return nil
}

func (c *Config) validateServer() error {
    if err := validatePort(c.Server.Port, "server"); err != nil {
        return err
    }
    // ... rest of server validation
}

func (c *Config) validateNBUServer() error {
    if err := validatePort(c.NbuServer.Port, "NBU server"); err != nil {
        return err
    }
    // ... rest of NBU server validation
}

func validatePort(port, context string) error {
    if port == "" {
        return fmt.Errorf("%s port is required", context)
    }
    p, err := strconv.Atoi(port)
    if err != nil || p < 1 || p > 65535 {
        return fmt.Errorf("invalid %s port: %s", context, port)
    }
    return nil
}
```

---

## 5. Performance Optimizations

### 5.1 Reduce Map Allocations

**Location**: `internal/exporter/prometheus.go` lines 56-59

**Issue**: Creating new maps on every scrape could be optimized.

**Current**:

```go
storageMetrics := make(map[string]float64)
jobsSize := make(map[string]float64)
jobsCount := make(map[string]float64)
jobsStatusCount := make(map[string]float64)
```

**Recommendation**: Pre-allocate with estimated capacity if you know typical sizes:

```go
storageMetrics := make(map[string]float64, 50)  // Typical storage unit count
jobsSize := make(map[string]float64, 100)       // Typical job type combinations
```

### 5.2 Context Cancellation in Pagination

**Location**: `internal/exporter/netbackup.go` lines 82-95

**Good**: Already checking `ctx.Done()` in the loop. Consider adding timeout logging:

```go
case <-ctx.Done():
    log.Warnf("Pagination cancelled: %v", ctx.Err())
    return ctx.Err()
```

---

## 6. Security Improvements

### 6.1 Sensitive Data Logging

**Location**: `main.go` line 145

**Issue**: API key is logged even when masked in debug mode.

**Current**:

```go
if debug {
    log.Infof("API Key: %s", cfg.MaskAPIKey())
}
```

**Recommendation**: Remove this entirely or only log that an API key is configured:

```go
if debug {
    log.Debug("API key configured: yes")
}
```

### 6.2 TLS Configuration

**Location**: `internal/exporter/client.go` line 36

**Issue**: `InsecureSkipVerify` is configurable but should have a warning.

**Recommendation**: Add logging when insecure mode is enabled:

```go
func NewNbuClient(cfg models.Config) *NbuClient {
    if cfg.NbuServer.InsecureSkipVerify {
        log.Warn("TLS certificate verification is disabled - this is insecure for production use")
    }
    // ... rest of implementation
}
```

---

## 7. Missing Features for Production Readiness

### 7.1 Metrics for Exporter Health

**Recommendation**: Add self-monitoring metrics:

```go
// In prometheus.go
nbuScrapeDuration: prometheus.NewDesc(
    "nbu_scrape_duration_seconds",
    "Duration of the scrape in seconds",
    nil, nil,
)
nbuScrapeErrors: prometheus.NewDesc(
    "nbu_scrape_errors_total",
    "Total number of scrape errors",
    []string{"component"}, nil,
)
```

### 7.2 Graceful Degradation

**Location**: `internal/exporter/prometheus.go` lines 61-66

**Good**: Already continuing on errors. Consider tracking error counts:

```go
var scrapeErrors int
if err := FetchStorage(ctx, c.client, storageMetrics); err != nil {
    log.Errorf("Failed to fetch storage metrics: %v", err)
    scrapeErrors++
}
// Expose scrapeErrors as a metric
```

### 7.3 Configuration Reload

**Recommendation**: Add support for SIGHUP to reload configuration without restart:

```go
// In main.go
func (s *Server) ReloadConfig(newCfg models.Config) error {
    s.cfg = newCfg
    log.Info("Configuration reloaded")
    return nil
}
```

---

## 8. Code Organization

### 8.1 Separate Concerns in Models

**Location**: `internal/models/Config.go`

**Issue**: Config struct mixes data structure with business logic (URL building, validation).

**Recommendation**: Consider moving URL building to a separate package:

```go
// internal/nbuapi/url_builder.go
type URLBuilder struct {
    baseURL string
}

func NewURLBuilder(cfg models.Config) *URLBuilder {
    return &URLBuilder{
        baseURL: fmt.Sprintf("%s://%s:%s%s", 
            cfg.NbuServer.Scheme, cfg.NbuServer.Host, 
            cfg.NbuServer.Port, cfg.NbuServer.URI),
    }
}

func (b *URLBuilder) Build(path string, params map[string]string) string {
    // URL building logic
}
```

---

## 9. Documentation Gaps

**Missing**:

- Architecture decision records (why certain patterns were chosen)
- API rate limiting considerations
- Error code meanings from NetBackup API
- Metric naming conventions documentation

**Recommendation**: Create `docs/architecture.md` and `docs/metrics.md`

---

## Priority Summary

### High Priority (Do First)

1. Add comprehensive test coverage (80%+ target)
2. Fix string-based metric key parsing
3. Define HTTPClient interface for testability
4. Remove unused logging functions

### Medium Priority

5. Refactor logging package to remove globals
6. Extract validation logic into smaller functions
7. Add exporter health metrics
8. Improve error context throughout

### Low Priority (Nice to Have)

9. Pre-allocate maps with capacity hints
10. Add configuration reload support
11. Separate URL building from Config model
12. Enhanced documentation

---

## Overall Assessment

The codebase is well-structured overall with good separation of concerns. The main gaps are in testing and some refinements to make the code more maintainable and production-ready. The architecture follows Go best practices in most areas, with clear package boundaries and appropriate use of interfaces where they exist.

**Key Strengths**:

- Clean separation between exporter, models, and utilities
- Good use of context for cancellation
- Proper error wrapping with context
- Constants used instead of magic numbers
- Graceful shutdown handling

**Key Weaknesses**:

- No test coverage (critical gap)
- Some anti-patterns (global variables, string-based parsing)
- Missing production observability features
- Inconsistent use of custom logging functions

Addressing the high-priority items will significantly improve code quality and production readiness.
