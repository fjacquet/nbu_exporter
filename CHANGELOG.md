# Changelog

All notable changes to the NBU Exporter project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **OpenTelemetry Distributed Tracing**: Optional distributed tracing for NetBackup API calls and Prometheus scrape cycles
  - OTLP gRPC exporter for sending traces to OpenTelemetry Collector, Jaeger, or Tempo
  - Configurable sampling rates (0.0 to 1.0) for controlling trace collection overhead
  - Automatic trace context propagation using W3C Trace Context standard
  - Span hierarchy: `prometheus.scrape` → `netbackup.fetch_storage`/`netbackup.fetch_jobs` → `http.request`
  - Rich span attributes following OpenTelemetry semantic conventions
  - Resource attributes: service name, version, hostname, NetBackup server
  - Graceful degradation: continues operating if tracing initialization fails
  - Zero overhead when disabled
  - Minimal overhead when enabled: < 2% at 0.1 sampling, < 5% at 1.0 sampling
- `opentelemetry` configuration section with fields: `enabled`, `endpoint`, `insecure`, `samplingRate`
- Telemetry manager (`internal/telemetry/manager.go`) for OpenTelemetry lifecycle management
- Instrumented HTTP client with automatic span creation for all NetBackup API requests
- Instrumented Prometheus collector with root span for scrape cycles
- Span attributes for storage operations: endpoint, storage unit count, API version
- Span attributes for job operations: endpoint, time window, total jobs, pagination details
- Span attributes for HTTP requests: method, URL, status code, duration, request/response sizes
- Error recording in spans with detailed error messages and span status
- Trace context injection into outgoing NetBackup API requests
- Docker Compose example with OpenTelemetry Collector and Jaeger (`docker-compose-otel.yaml`)
- OpenTelemetry Collector configuration example (`otel-collector-config.yaml`)
- Comprehensive OpenTelemetry documentation in README and `docs/opentelemetry-example.md`
- Trace analysis guide at `docs/trace-analysis-guide.md`
- OpenTelemetry integration tests with mock collector
- OpenTelemetry benchmark tests for performance validation
- **Multi-Version API Support**: NetBackup 10.0 (API 3.0), 10.5 (API 12.0), and 11.0 (API 13.0)
- **Automatic Version Detection**: Intelligent fallback logic (13.0 → 12.0 → 3.0) with retry mechanism
- **Content-Type Validation**: Validates server responses are JSON before unmarshaling, preventing cryptic error messages
- `apiVersion` configuration field in `nbuserver` section (now optional, defaults to auto-detection)
- API version included in Accept header for all NetBackup API requests
- Version detector module (`internal/exporter/version_detector.go`) with exponential backoff retry logic
- `nbu_api_version` Prometheus metric to expose currently active API version
- Comprehensive test suites:
  - Version detection integration tests with mock servers
  - Backward compatibility tests for existing configurations
  - End-to-end workflow tests for all API versions
  - Performance validation tests
  - Metrics consistency tests across versions
  - API compatibility tests with real response fixtures
- Test fixtures for all API versions in `testdata/api-versions/`:
  - `jobs-response-v3.json`, `jobs-response-v12.json`, `jobs-response-v13.json`
  - `storage-response-v3.json`, `storage-response-v12.json`, `storage-response-v13.json`
  - `error-406-response.json` for version incompatibility testing
- Comprehensive migration guide at `docs/netbackup-11-migration.md` with multiple deployment scenarios
- Configuration examples for all supported versions in `docs/config-examples/`
- Enhanced error handling for 406 (Not Acceptable) responses with troubleshooting guidance
- Optional fields in storage data model: `storageCategory`, replication capabilities, snapshot flags, WORM support
- Optional field in jobs data model: `kilobytesDataTransferred`
- Configuration validation with `Config.Validate()` method
- Helper methods for configuration: `GetNBUBaseURL()`, `GetServerAddress()`, `GetScrapingDuration()`, `MaskAPIKey()`, `BuildURL()`
- `NbuClient` structure for reusable HTTP client with connection pooling
- Context support throughout for proper cancellation and timeout handling
- Structured metric key types (`StorageMetricKey`, `JobMetricKey`, `JobStatusKey`)
- Health check endpoint at `/health`
- `Server` structure for better application lifecycle management
- Graceful shutdown with configurable timeout (10 seconds)
- `insecureSkipVerify` configuration option for TLS verification
- API key masking in debug logs for security
- `ReadHeaderTimeout` to HTTP server for security
- Comprehensive error context with `fmt.Errorf` wrapping
- Collection timeout (2 minutes) to prevent hanging scrapes
- **Linting Configuration**: Added `.markdownlint.json` and `.prettierrc` for consistent code formatting
  - Markdown line length relaxed to 120 characters
  - Tables excluded from line length checks
  - Prettier configured with 120 character width and LF line endings

### Changed

- **BREAKING**: Fixed typo `scrappingInterval` → `scrapingInterval` in configuration
- Accept header format now includes API version: `application/vnd.netbackup+json;version=X.Y`
- Data models updated to support optional fields across all API versions
- Default API version changed from "12.0" to "13.0" (with automatic fallback)
- `apiVersion` configuration field is now **optional** (previously required)
- Minimum NetBackup version requirement: 10.0+ (supports API versions 3.0, 12.0, and 13.0)
- Refactored `main.go` with `Server` structure for better separation of concerns
- Improved error handling - functions now return errors instead of calling `os.Exit()`
- HTTP client now reused across requests for better performance
- TLS verification now configurable (defaults to secure mode)
- Metric collection continues even if one source fails (storage or jobs)
- Improved godoc comments throughout codebase
- Better variable naming following Go conventions
- Centralized URL construction in `Config.BuildURL()`
- Enhanced logging with structured fields and debug mode support
- **Code Quality Improvements**: Comprehensive refactoring for maintainability and SonarCloud compliance
  - Consolidated duplicate span creation logic into single reusable `createSpan` helper function
  - Centralized all OpenTelemetry span attribute keys as constants in `internal/telemetry/attributes.go`
  - Extracted complex error message templates to `internal/telemetry/errors.go` for easier maintenance
  - Batched span attribute recording for improved performance (single `SetAttributes` call)
  - Enhanced OpenTelemetry endpoint validation with format and port range checks
  - Extracted complex conditional logic to named helper functions for clarity
  - Renamed all test functions to follow Go conventions (removed underscores, e.g., `TestNbuClient_GetHeaders` → `TestNbuClientGetHeaders`)
  - Eliminated duplicate string literals by extracting to named constants (API versions, content types, test configuration values)
  - Reduced cognitive complexity in 8 test functions by extracting helper functions:
    - `TestNewNbuCollectorAutomaticDetection` (complexity 24 → <15)
    - `TestAPIVersionDetectorIntegration` (complexity 37 → <15)
    - `TestConfigValidateNbuServer` (complexity 23 → <15)
    - `TestConfigValidateOpenTelemetry` (complexity 16 → <15)
    - `TestConfigValidateServer` (complexity 17 → <15)
    - `TestConfigGetNBUBaseURL` (complexity 23 → <15)
    - `TestConfigValidate` (complexity 23 → <15)
    - `createMockServerWithFile` helper (complexity 22 → <15)
  - Centralized test helper functions in `internal/testutil` package with fluent builder interfaces
  - Added `TestConfigBuilder` for consistent test configuration creation
  - Added `MockServerBuilder` for simplified mock HTTP server setup
  - Enhanced error messages with additional context (URL, status code, content-type, response preview)
  - Added comprehensive package-level documentation with usage examples
  - All code now passes SonarCloud "Sonar Way" quality profile checks

### Removed

- `cmd.go` - Unused `ConfigCommand` structure
- `debug.go` - File containing only commented-out code
- Unused variables `programName` and `nbuRoot` from package level
- Obvious and redundant code comments
- Direct `os.Exit()` calls from utility functions

### Fixed

- **HTML Response Handling**: Server returning HTML instead of JSON now produces clear error messages instead of cryptic "invalid character '<'" errors
- Error handling in `ReadFile()` - now returns errors properly
- Missing error checks in metric collection
- Resource leaks from not reusing HTTP clients
- Potential Slowloris attack vector with `ReadHeaderTimeout`
- Configuration validation - ports, schemes, and durations now validated
- Graceful shutdown - now uses proper context with timeout
- JSON unmarshaling errors now include response preview for easier debugging

### Security

- TLS certificate verification now configurable and secure by default
- API keys masked in logs to prevent accidental exposure
- Added `ReadHeaderTimeout` to prevent slow header attacks
- Proper error context without exposing sensitive information

### Performance

- HTTP client connection pooling reduces overhead
- Context-aware operations allow early cancellation
- Reduced allocations from client reuse
- Better resource cleanup with proper context handling
- **Batched Job Pagination**: Jobs fetched in batches of 100 per API call (~100x fewer requests for large job sets)
- **Parallel Metric Collection**: Storage and job metrics collected concurrently with `errgroup`, reducing scrape time to `max(storage_time, jobs_time)` instead of sum
- **Pre-allocation Capacity Hints**: Maps and slices pre-allocated with expected sizes (100 job metrics, 50 status metrics) to reduce GC pressure

## Migration Notes

### OpenTelemetry Distributed Tracing (Optional)

**New Feature**: This version adds optional OpenTelemetry distributed tracing for diagnosing slow scrapes and identifying performance bottlenecks.

**Quick Start:**

1. Add OpenTelemetry configuration to your `config.yaml`:

```yaml
opentelemetry:
  enabled: true
  endpoint: "localhost:4317" # Your OTLP collector endpoint
  insecure: true # Use false for production with TLS
  samplingRate: 0.1 # Sample 10% of scrapes
```

2. Deploy OpenTelemetry Collector (see `docker-compose-otel.yaml` for example)
3. Restart the exporter
4. View traces in Jaeger, Tempo, or your observability backend

**Backward Compatibility**:

- OpenTelemetry is completely optional - existing deployments work without any changes
- When disabled or not configured, zero tracing overhead
- All existing Prometheus metrics remain unchanged
- No impact on scrape performance when disabled

**Use Cases**:

- Diagnose slow NetBackup API calls
- Identify performance bottlenecks in scrape cycles
- Track request flows through the exporter
- Correlate logs with traces for troubleshooting
- Monitor API version detection performance

See [docs/opentelemetry-example.md](docs/opentelemetry-example.md) for complete setup guide and [docs/trace-analysis-guide.md](docs/trace-analysis-guide.md) for trace analysis examples.

### Multi-Version API Support

**Important**: This version adds support for NetBackup 10.0, 10.5, and 11.0 with automatic version detection. See [docs/netbackup-11-migration.md](docs/netbackup-11-migration.md) for complete migration guide.

**Quick Start:**

1. Ensure NetBackup server is version 10.0 or later
2. **Optional**: Remove or update `apiVersion` field in config.yaml to enable automatic detection
3. Restart the exporter - it will automatically detect the highest supported API version
4. Verify detected version in logs: `INFO: Detected NetBackup API version: X.Y`
5. Verify metrics collection

**Backward Compatibility**:

- Existing configurations with explicit `apiVersion` continue to work without changes
- Existing configurations without `apiVersion` will now auto-detect (previously defaulted to "12.0")
- No breaking changes to Prometheus metrics - all metric names and labels remain consistent
- NetBackup 11.0 maintains backward compatibility with API versions 12.0 and 3.0

### Configuration File Changes

Update your `config.yaml`:

```yaml
# Change this:
server:
    scrappingInterval: "1h"

# To this:
server:
    scrapingInterval: "1h"

# API version configuration (all options are valid):

# Option 1: Automatic detection (recommended)
nbuserver:
    # apiVersion not specified - will auto-detect

# Option 2: Explicit version for NetBackup 11.0
nbuserver:
    apiVersion: "13.0"

# Option 3: Explicit version for NetBackup 10.5
nbuserver:
    apiVersion: "12.0"

# Option 4: Explicit version for NetBackup 10.0-10.4
nbuserver:
    apiVersion: "3.0"

# Optionally add (defaults to false if omitted):
nbuserver:
    insecureSkipVerify: false
```

### Testing Your Migration

```bash
# Validate configuration
./bin/nbu_exporter --config config.yaml --debug

# Check health endpoint
curl http://localhost:2112/health

# Check metrics endpoint
curl http://localhost:2112/metrics

# Verify detected API version
curl http://localhost:2112/metrics | grep nbu_api_version

# Check logs for version detection
tail -f log/nbu-exporter.log | grep "Detected NetBackup API version"
```

### Test Coverage

This release includes comprehensive test coverage:

- 80%+ code coverage for API client and version detection modules
- Unit tests for all three API versions (3.0, 12.0, 13.0)
- Integration tests with mock NetBackup servers
- Backward compatibility tests for existing configurations
- End-to-end workflow tests including fallback scenarios
- Performance validation tests
- Metrics consistency tests across all versions

## [Previous Versions]

No previous changelog entries available. This is the first documented release.
