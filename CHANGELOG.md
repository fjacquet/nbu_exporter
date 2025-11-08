# Changelog

All notable changes to the NBU Exporter project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- NetBackup API 10.5 support (API version 12.0)
- `apiVersion` configuration field in `nbuserver` section (defaults to "12.0")
- API version included in Accept header for all NetBackup API requests
- Optional fields in storage data model: `storageCategory`, replication capabilities, snapshot flags, WORM support
- Optional field in jobs data model: `kilobytesDataTransferred`
- API version detection capability in HTTP client
- Enhanced error handling for 406 (Not Acceptable) responses
- Test fixtures for API 10.5 responses in `testdata/api-10.5/`
- Comprehensive migration guide at `docs/api-10.5-migration.md`
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

### Changed
- **BREAKING**: Fixed typo `scrappingInterval` → `scrapingInterval` in configuration
- Accept header format now includes API version: `application/vnd.netbackup+json;version=12.0`
- Data models updated to support new optional fields from API 10.5
- Minimum NetBackup version requirement: 10.5+ (for API version 12.0)
- Refactored `main.go` with `Server` structure for better separation of concerns
- Improved error handling - functions now return errors instead of calling `os.Exit()`
- HTTP client now reused across requests for better performance
- TLS verification now configurable (defaults to secure mode)
- Metric collection continues even if one source fails (storage or jobs)
- Improved godoc comments throughout codebase
- Better variable naming following Go conventions
- Centralized URL construction in `Config.BuildURL()`
- Enhanced logging with structured fields and debug mode support

### Removed
- `cmd.go` - Unused `ConfigCommand` structure
- `debug.go` - File containing only commented-out code
- Unused variables `programName` and `nbuRoot` from package level
- Obvious and redundant code comments
- Direct `os.Exit()` calls from utility functions

### Fixed
- Error handling in `ReadFile()` - now returns errors properly
- Missing error checks in metric collection
- Resource leaks from not reusing HTTP clients
- Potential Slowloris attack vector with `ReadHeaderTimeout`
- Configuration validation - ports, schemes, and durations now validated
- Graceful shutdown - now uses proper context with timeout

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

## Migration Notes

### NetBackup API 10.5 Upgrade

**Important**: This version adds support for NetBackup 10.5 (API version 12.0). See [docs/api-10.5-migration.md](docs/api-10.5-migration.md) for complete migration guide.

**Quick Start:**
1. Ensure NetBackup server is version 10.5 or later
2. Add `apiVersion: "12.0"` to `nbuserver` section in config.yaml (optional, defaults to "12.0")
3. Restart the exporter
4. Verify metrics collection

**Backward Compatibility**: Existing configurations will work with default API version "12.0". No breaking changes to Prometheus metrics.

### Configuration File Changes

Update your `config.yaml`:

```yaml
# Change this:
server:
    scrappingInterval: "1h"

# To this:
server:
    scrapingInterval: "1h"

# Add API version (optional, defaults to "12.0"):
nbuserver:
    apiVersion: "12.0"

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
```

## [Previous Versions]

No previous changelog entries available. This is the first documented release.
