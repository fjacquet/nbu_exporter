# Coding Conventions

**Analysis Date:** 2026-01-22

## Naming Patterns

**Files:**

- Exported types use PascalCase: `Config.go`, `Jobs.go`, `Storages.go`
- Private/helper files use snake_case or descriptive names: `test_common.go`, `version_detector.go`, `client.go`
- Test files use `_test.go` suffix: `prometheus_test.go`, `Config_test.go`

**Functions & Methods:**

- Exported functions/methods use PascalCase: `NewNbuCollector()`, `Validate()`, `Describe()`, `ReadFile()`
- Private functions use camelCase: `validatePort()`, `performVersionDetectionIfNeeded()`, `createScrapeSpan()`
- Constructors use `New<TypeName>()` pattern: `NewNbuClient()`, `NewServer()`, `NewNbuCollector()`
- Validation functions use `validate<Field>()` pattern: `validateServerConfig()`, `validateAPIVersion()`, `validateNBUServerConfig()`
- Helper functions use descriptive verb-noun pattern: `createTestConfig()`, `loadStorageTestData()`, `extractVersionFromHeader()`

**Variables & Constants:**

- Constants use UPPER_SNAKE_CASE: `defaultTimeout`, `shutdownTimeout`, `collectionTimeout`, `readHeaderTimeout`
- Module-level constants group related values: `programName`, `shutdownTimeout`, `readHeaderTimeout` (together at top of main.go)
- Interface parameters and config constants: `testAPIKey`, `testPathMetrics`, `testPathNetBackup`
- Variable names are descriptive: `telemetryMgr`, `tracerProvider`, `storageMetrics`, `allJobs`

**Types:**

- Exported struct names use PascalCase: `Server`, `NbuCollector`, `Config`, `NbuClient`
- Struct fields use PascalCase: `cfg`, `client`, `tracer`, `httpSrv`, `registry`
- Struct embedding omits field names for anonymous embedded types
- Test table structs use lowercase field names: `name`, `apiVersion`, `expectedVersion`, `expectError`

## Code Style

**Formatting:**

- Tool: `go fmt` (enforced via Makefile target `make sure`)
- No external formatter config file (uses Go defaults)
- Indentation: tabs (Go standard)

**Linting:**

- Tool: `golangci-lint` (enforced via Makefile target `make sure`)
- No project-level `.golangci.yaml` (uses default linter config)
- Runs automatically in `make sure` target

**Line Length:**

- No hard line limit enforced; wrapped at logical boundaries for readability
- Long function documentation and error messages can extend naturally
- Descriptive names preferred over abbreviations

## Import Organization

**Order:**

1. Standard library imports: `context`, `fmt`, `net/http`, `os`, `time`
2. Internal imports: `github.com/fjacquet/nbu_exporter/internal/...`
3. Third-party imports: `github.com/prometheus/...`, `github.com/sirupsen/...`, `go.opentelemetry.io/...`
4. Blank line separating groups

**Path Aliases:**

- Logrus imported as `log`: `log "github.com/sirupsen/logrus"`
- Used as `log.Info()`, `log.Warn()`, `log.Fatal()`, `log.Debug()`
- No other path aliases in use

**Example from `main.go` (lines 19-39):**

```go
import (
 "context"
 "fmt"
 "net/http"
 "os"
 "os/signal"
 "syscall"
 "time"

 "github.com/fjacquet/nbu_exporter/internal/exporter"
 "github.com/fjacquet/nbu_exporter/internal/logging"
 "github.com/fjacquet/nbu_exporter/internal/models"
 "github.com/fjacquet/nbu_exporter/internal/telemetry"
 "github.com/fjacquet/nbu_exporter/internal/utils"
 "github.com/prometheus/client_golang/prometheus"
 "github.com/prometheus/client_golang/prometheus/promhttp"
 log "github.com/sirupsen/logrus"
 "github.com/spf13/cobra"
 "go.opentelemetry.io/otel"
 "go.opentelemetry.io/otel/propagation"
)
```

## Error Handling

**Patterns:**

- Use `errors.New()` for simple, constant error messages without context
- Use `fmt.Errorf()` with `%w` verb for error wrapping to preserve stack: `fmt.Errorf("failed to X: %w", err)`
- Descriptive error messages include context about what failed and expected format/values
- Validation errors include guidance (expected format, valid range, etc.)

**Examples from `Config.go`:**

```go
// Simple error
return errors.New("server port is required")

// Wrapped error with context
return fmt.Errorf("invalid scraping interval format '%s': %w (expected format: 5m, 1h, 30s)",
    c.Server.ScrapingInterval, err)

// Validation with helpful message
return fmt.Errorf("unsupported API version: %s (supported versions: %v)",
    c.NbuServer.APIVersion, SupportedAPIVersions)
```

**Error Propagation:**

- All functions that can fail return `error` as last return value
- Errors checked immediately after function calls: `if err != nil { return err }`
- Graceful degradation in collectors: errors logged as warnings, collection continues with partial metrics

## Logging

**Framework:** `logrus` imported as `log`

**Patterns:**

- `log.Info()` for informational messages (startup, configuration)
- `log.Warn()` / `log.Warnf()` for recoverable issues (initialization failures that allow graceful degradation)
- `log.Debug()` / `log.Debugf()` for detailed diagnostic info (enabled via `--debug` flag)
- `log.Fatalf()` for unrecoverable errors that prevent startup

**Example from `main.go`:**

```go
log.Infof("Starting %s...", programName)
log.Warnf("Failed to initialize OpenTelemetry: %v. Continuing without tracing.", err)
log.Debug("Debug mode enabled")
log.Fatalf("HTTP server error: %v", err)
```

## Comments

**When to Comment:**

- Package-level comments documenting the package purpose (required, at top of each package)
- Exported function/type comments explaining what it does and key behavior
- Non-obvious logic or complex algorithms
- Security-relevant code (e.g., TLS settings, API key handling)
- Parameter descriptions for exported functions

**JSDoc/TSDoc:**

- Not used; Go doesn't use JSDoc
- Function comments follow standard Go format: `// FunctionName does X.`
- Multi-line comments for detailed explanations, especially for exported functions

**Example from `prometheus.go` (lines 23-44):**

```go
// NbuCollector implements the Prometheus Collector interface for NetBackup metrics.
// It collects storage capacity and job statistics from the NetBackup API and exposes
// them as Prometheus metrics.
//
// The collector fetches:
//   - Storage unit capacity (free/used bytes) for disk-based storage
//   - Job statistics (count, bytes transferred) aggregated by type, policy, and status
//   - Job status counts aggregated by action and status code
//   - API version information
//
// Metrics are collected on-demand when Prometheus scrapes the /metrics endpoint.
type NbuCollector struct {
```

**Example from `main.go` (lines 52-60):**

```go
// Server encapsulates the HTTP server and its dependencies for serving Prometheus metrics.
// It manages the lifecycle of the HTTP server, Prometheus registry, NetBackup collector,
// and OpenTelemetry telemetry manager.
type Server struct {
 cfg              models.Config        // Application configuration
 httpSrv          *http.Server         // HTTP server instance
 registry         *prometheus.Registry // Prometheus metrics registry
 telemetryManager *telemetry.Manager   // OpenTelemetry telemetry manager (nil if disabled)
}
```

## Function Design

**Size:**

- No hard limit; functions are organized by logical responsibility
- Average function length: 20-50 lines for business logic
- Longer functions acceptable when they have single responsibility (e.g., validation with multiple checks)
- Helper methods extract reusable validation/formatting logic

**Parameters:**

- Context passed as first parameter for async operations: `func (c *NbuCollector) createScrapeSpan(ctx context.Context)`
- Configuration structs passed by value (immutable view): `NewNbuCollector(cfg models.Config)`
- Pointers used for mutable receivers or large structs
- Variadic args used for flexible message formatting in helpers: `msgAndArgs ...interface{}`

**Return Values:**

- Single return value for simple operations: `func FileExists(filename string) bool`
- Named returns when returning multiple values related to operation: `(context.Context, trace.Span)` for span creation
- Error as last return value (Go idiom): `(*NbuCollector, error)` for constructors
- Interfaces returned by value: `prometheus.Collector`, not `*prometheus.Collector`

**Nil Safety:**

- Methods check for nil receivers/fields before use (e.g., tracer nil-check in `createScrapeSpan`)
- Constructors may return nil receivers with non-nil error
- Defer used with nil-safe close: `defer func() { _ = f.Close() }()`

## Module Design

**Exports:**

- Exported types/functions documented with comments
- Private helpers exported only if useful across packages (else kept private)
- Constructors always exported: `New<Type>()` (public factory functions)

**Barrel Files:**

- Not used; no re-exports to simplify imports
- Import directly from source packages: `internal/exporter`, `internal/models`, `internal/testutil`

**Package Organization:**

- `internal/exporter/` - Prometheus collector, HTTP client, metric collection
- `internal/models/` - Configuration and API response structs
- `internal/telemetry/` - OpenTelemetry integration
- `internal/testutil/` - Shared test utilities and fixtures
- `internal/utils/` - File I/O and utility functions
- `internal/logging/` - Logging configuration

## Interface Satisfaction

- Explicit implementation of `prometheus.Collector` in `NbuCollector` type with `Describe()` and `Collect()` methods
- No interface comments; methods implement standard library interfaces (Go idiom)

---

_Convention analysis: 2026-01-22_
