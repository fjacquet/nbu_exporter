# Changelog

All notable changes to the NBU Exporter project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **Per-client lifecycle metrics & three new dashboards** (reconciled from PR #39, thanks
  **@cbijon**): per-client **lifecycle** metrics behind the `collectors.perClient` opt-in +
  allowlist — `nbu_client_jobs_count{client,action,status}` and
  `nbu_client_last_job_success_seconds{client,policy,action}` covering the full
  BACKUP → DUPLICATION → IMPORT lifecycle, derived from the main jobs scrape and bounded to
  allowlisted clients — plus three generated dashboards on the same `site`-wired generator:
  **Lifecycle** (`nbu-lifecycle`), **Tape & Disk Pools** (`nbu-tape`), and **Multi-site**
  (`nbu-multisite`). #39's tape metrics were folded into the existing `collectors.tape` as a
  superset (two extra endpoints — see Tape metrics below), and its richer tape label schema was
  adopted (raw `DRIVE_STATUS_*` `status`; media keyed by `pool`/`media_type`/`robot_type`).
- **Multi-site Grafana dashboards**: every generated dashboard now carries a `site` template
  variable (multi-value + "All", sourced from `label_values(nbu_up, site)`) as the first selector,
  and every panel query is filtered by `site=~"$site"` with `site` threaded into each `by (…)`
  grouping and legend — so series from multiple NetBackup primaries no longer collapse together or
  double-count. The Overview gains a per-site availability row (one UP/DOWN tile per primary,
  repeated over `$site`) so a down master is obvious. The filter/grouping is centralized in the
  `grafana/gen/` generator (`panels.with_site`), and `build_dashboards.py` now fails the build if any
  dashboard is missing the `site` selector or leaves a query unfiltered. See the Multi-Site
  Dashboards design spec.
- **Alerting rules** (`deploy/prometheus/`): three optional, site-aware Prometheus rule files —
  `rules-perclient.yml` (`NbuClientBackupStale` >25h, `NbuClientBackupCritical` >48h,
  `NbuClientNoRecentTapeCopy` >26h, `NbuClientNoRecentReplication` >28h,
  `NbuClientBackupFailureRate` >20%), `rules-tape.yml` (`NbuTapeDriveDown` / `NbuTapeDriveDisabled`,
  `NbuTapePoolScratchLow`, `NbuDiskPoolVolumeDegraded`), and `rules-multisite.yml`
  (`NbuInterSiteDivergence`) — each with promtool unit tests, plus a `make check-rules` target. The
  generic `nbu.rules.yml` is unchanged.
- **Per-client last-successful-backup metric** (opt-in `collectors.perClient`, default off):
  `nbu_client_last_successful_backup_timestamp_seconds{site,client}` for each allowlisted client,
  from a targeted `/admin/jobs` query (latest `status=0` BACKUP, `sort=-endTime`, `page[limit]=1`).
  An exact-name allowlist bounds the `client` cardinality; an empty allowlist emits nothing. Enables
  a "no backup in N hours" alert. See the Feature 3 design spec.
- **Tape / drive metrics** (opt-in `collectors.tape`, default off): `nbu_tape_drives_count`
  (by drive_type/robot_type/raw `status`), `nbu_tape_drive_info` (per drive), `nbu_tape_media_count`
  (by pool/media_type/robot_type), `nbu_tape_robot_device_hosts`, plus `nbu_tape_pool_partially_full`
  (`/storage/tape-volume-pools`) and `nbu_disk_pool_volume_count` (`/storage/disk-pools`), collected
  over REST — drives on NBU 10.0+, the rest on 10.5+ (API v12.0+) — with per-endpoint graceful
  degradation and the `site` label. No CLI shell-out. See the Feature 2 design spec.
- **Multi-site support**: a single exporter can scrape multiple NetBackup primary servers via
  a `nbuservers:` list, each with a unique `site`; every metric series carries a `site` label
  (first label). Collection adopts the family **snapshot model** — a background loop polls
  every site on `server.collectionInterval` (default 5m) and publishes an immutable snapshot,
  decoupling backend API load from Prometheus scrape frequency. `/metrics` serves immediately
  (empty until the first cycle); a down site reports only `nbu_up{site=…}=0` without affecting
  the others. A legacy single `nbuserver:` block is auto-mapped to a one-entry list (`site`
  defaults to the host), so existing configs keep working unchanged. Opt-in sub-collectors
  (alerts/malware/catalog/SLO) are per-site and carry the `site` label too. See
  [ADR-0004](https://github.com/fjacquet/nbu_exporter/blob/main/docs/adr/0004-multisite-snapshot-collection-model.md) and
  `docs/config-examples/config-multisite.yaml`.
- **NetBackup 10.x support**: the exporter negotiates API media-type `version=10.0` for
  NetBackup 10.0–10.4 (replacing the never-valid `3.0`); the auto-detect ladder is now
  `14.0 → 13.0 → 12.0 → 10.0`. Includes the NBU 10.3 and 11.2 OpenAPI bundles used for
  validation ([#34](https://github.com/fjacquet/nbu_exporter/pull/34)).

### Fixed

- **Job metrics on all modern NetBackup**: `GET /admin/jobs` is cursor-paginated (API ≥ 9.0:
  `page[after]`/`page[before]`, string `next`/`prev`). The exporter sent `page[offset]` and
  parsed `next` as an int, so the jobs response failed to unmarshal and **all `nbu_jobs_*`
  metrics were silently dropped** on NBU 10.x/10.5/11.0/11.2. Reworked the jobs pagination
  model and loop to follow cursors (storage pagination unchanged)
  ([#34](https://github.com/fjacquet/nbu_exporter/pull/34)).
- Restored API-version auto-detection when `apiVersion` is omitted — a forced `14.0` default
  in `SetDefaults()` had silently disabled it, hard-failing against NetBackup < 11.2
  ([#34](https://github.com/fjacquet/nbu_exporter/pull/34)).

### Changed

- Repo hygiene: removed a 22 MB committed `go test -c` binary and empty doc stubs, untracked
  `.planning/` and `.serena/` tool state (now gitignored), and fixed a stale `3.0` reference
  in `config-auto-detect.yaml` ([#35](https://github.com/fjacquet/nbu_exporter/pull/35)).

## [3.0.1] - 2026-06-14

### Added

- **Helm chart** for Kubernetes deployment, with lockstep version publishing alongside
  container images ([#33](https://github.com/fjacquet/nbu_exporter/pull/33)).

## [3.0.0] - 2026-06-14

### Changed

- **BREAKING:** the canonical metrics/exporter port is now **9440** (previously 2112).
  Update Prometheus scrape configs, compose files, Helm values, and firewall rules
  accordingly ([#32](https://github.com/fjacquet/nbu_exporter/pull/32)).

### Added

- Node Exporter Full (Grafana dashboard 1860) companion dashboard for host-level metrics
  alongside the NBU dashboards ([#32](https://github.com/fjacquet/nbu_exporter/pull/32)).

## [2.14.1] - 2026-06-14

### Fixed

- Demo compose stack: log to stdout and allow overriding host ports
  ([#30](https://github.com/fjacquet/nbu_exporter/pull/30)).

## [2.14.0] - 2026-06-14

### Added

- **Observability quickstart demo stack** — exporter + Prometheus + Grafana with datasource
  and dashboard provisioning, runnable with one `docker compose up`
  ([#29](https://github.com/fjacquet/nbu_exporter/pull/29)).
- **Redesigned Grafana dashboards** — focused, generator-produced (`grafana/gen/`),
  cross-linked; the legacy hardcoded dashboard was retired
  ([#28](https://github.com/fjacquet/nbu_exporter/pull/28)).

## [2.13.0] - 2026-06-14

### Added

- **NetBackup 11.2 (API version 14.0) support**: auto-detection probes `14.0` first
  (`14.0 → 13.0 → 12.0 → 3.0`), with v14 response fixtures.
- **Opt-in sub-collector framework** ([ADR-0002](https://github.com/fjacquet/nbu_exporter/blob/main/docs/adr/0002-opt-in-sub-collector-framework.md))
  with four collectors, all **default-off** via a new `collectors:` config section: alerts
  (`nbu_alerts_count`), malware scan results, catalog posture, and SLO compliance
  (`nbu_slo_count`). Each degrades gracefully and never affects `nbu_up`.
- Grafana panels for the new alerts/malware/catalog/SLO metrics.

## [2.12.0] - 2026-06-12

### Added

- Extended NetBackup job and storage metrics — job state, files, dedup ratio, queued reason,
  a duration histogram, and storage-capability info — plus templated dashboard and alerting
  rules ([#26](https://github.com/fjacquet/nbu_exporter/pull/26)).

## [2.11.1] - 2026-06-12

### Fixed

- Docker image copies the CA bundle from the build stage instead of `apk add`, for a smaller
  and more reproducible image ([#25](https://github.com/fjacquet/nbu_exporter/pull/25)).

## [2.11.0] - 2026-06-12

### Added

- Native `.env` loading at startup with no-override semantics (a real environment variable
  always wins over the `.env` file).

## [2.10.0] - 2026-06-11

### Changed

- Aligned `.gitignore` with the exporter-family canonical template.

## [2.9.0] - 2026-06-11

### Added

- `--trace` flag to log NetBackup API **response bodies** for live-appliance payload
  validation. It never logs request headers or credentials.

## [2.8.0] - 2026-06-05

### Changed

- Migrated the Homebrew distribution from a **formula** (`brews`) to a
  **cask** (`homebrew_casks`), following GoReleaser's deprecation of `brews`.
  The cask references the darwin-only archive (`binary "nbu_exporter"`) and
  includes a `postflight` hook that strips the Gatekeeper quarantine bit, since
  the binary is not notarized. `brew install fjacquet/tap/nbu_exporter` is
  unchanged for users. The tap's old `Formula/nbu_exporter.rb` is removed and a
  `tap_migrations.json` entry migrates existing formula installs to the cask.

## [2.7.0] - 2026-06-05

### Added

- Re-introduced the Homebrew tap, **scoped to macOS only**. The GoReleaser
  build is split into a darwin-only `macos` build/archive and a `others`
  (linux + windows) build/archive; the formula references the macOS archive,
  so it generates a `depends_on :macos` formula with no `on_linux` block.
  Linux users should use a GitHub Release binary, Docker, or build from source.

## [2.6.0] - 2026-06-05

### Removed

- Dropped Homebrew tap distribution ([#22](https://github.com/fjacquet/nbu_exporter/pull/22)).
  Removed the GoReleaser `brews` block, the orphaned `TAP_GITHUB_TOKEN` env in the
  release workflow, and the Homebrew install instructions from the README and
  installation guide. Install via build-from-source, Docker, or GitHub Releases.

## [2.5.0] - 2026-06-05

Tooling/CI/security baseline sync with the sibling `pflex_exporter` (see
[ADR-0001](https://github.com/fjacquet/nbu_exporter/blob/main/docs/adr/0001-tooling-baseline-sync-with-pflex.md), [#19](https://github.com/fjacquet/nbu_exporter/pull/19)).

### Added

- **CycloneDX SBOMs and signed releases**: each release archive now ships a
  CycloneDX SBOM (syft), and the checksums file is signed with cosign keyless
  (bundle format).
- **Consolidated CI** (`ci.yml`): golangci-lint v2.12.2, `go vet`, race-enabled
  tests, `govulncheck`, Semgrep, and a CycloneDX module SBOM — alongside the
  existing CodeQL analysis. The 70% coverage gate (`.testcoverage.yml`) is retained.
- **Makefile quality/security targets**: `tools`, `fmt-check`, `vet`, `lint`,
  `test-race`, `vuln`, `sbom`, and an aggregate `ci` target.
- **Architecture Decision Records** under `docs/adr/`.

### Changed

- **Go 1.26** (from 1.25); the `go` directive is pinned to `1.26.4` so CI installs
  a toolchain that includes stdlib security fixes.
- **Docker image runs as non-root** (uid 10001); the builder is pinned to
  `golang:1.26` and the binary is built statically (`Dockerfile` and
  `Dockerfile.goreleaser`).
- All GitHub Actions are **SHA-pinned** to their latest releases and the Semgrep
  container image is **digest-pinned** for reproducible, supply-chain-hardened CI.

### Security

- `govulncheck` runs in CI; pinning Go to 1.26.4 resolves 16 Go standard-library
  vulnerabilities present in 1.26.0.
- Removed the committed `.vscode/` editor configuration and added it to
  `.gitignore`.

## [2.2.1] - 2026-02-18

### Fixed

- **Double URL-encoding of job filter query parameter** ([#13](https://github.com/fjacquet/nbu_exporter/issues/13)): The `endTime` filter for job metrics was pre-encoded with `%20` for spaces, but `BuildURL()` applies URL encoding automatically via `url.Values.Encode()`. This caused double-encoding (`%20` → `%2520`), making the NetBackup API return HTTP 400 errors. Job metrics (`nbu_jobs_bytes`, `nbu_jobs_count`, `nbu_status_count`) were not collected as a result. Storage metrics were unaffected.

## [2.2.0] - 2026-01-24

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
- **Storage Metrics Caching**: TTL-based caching for storage metrics reduces NetBackup API load
  - Configurable cache TTL via `cacheTTL` field in config (default: 5 minutes)
  - Uses patrickmn/go-cache for thread-safe caching with automatic expiration
  - Cache hit returns instantly without API call, cache miss triggers fetch
  - Metric HELP string documents caching behavior per Prometheus best practices
- **Health Check with Connectivity Verification**: Enhanced `/health` endpoint verifies NetBackup API connectivity
  - Returns HTTP 200 when NBU API is reachable
  - Returns HTTP 503 with "UNHEALTHY" message when NBU API is unreachable
  - Uses lightweight version endpoint for connectivity test (5-second timeout)
  - Returns "OK (starting)" during startup phase before collector initialization
- **Staleness Tracking Metrics**: New metrics for monitoring data freshness
  - `nbu_up`: Gauge metric (1 = healthy, 0 = unhealthy) reflecting NBU API connectivity
  - `nbu_last_scrape_timestamp_seconds`: Unix timestamp of last successful collection with `source` label
- **Dynamic Configuration Reload**: Configuration changes can be applied without restarting the exporter
  - SafeConfig wrapper provides thread-safe config access via RWMutex
  - SIGHUP signal handler triggers manual config reload
  - fsnotify file watcher detects config file changes (watches directory for vim/emacs atomic saves)
  - Storage cache automatically flushed when NBU server address changes

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

1. Deploy OpenTelemetry Collector (see `docker-compose-otel.yaml` for example)
2. Restart the exporter
3. View traces in Jaeger, Tempo, or your observability backend

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

See [docs/opentelemetry-example.md](https://github.com/fjacquet/nbu_exporter/blob/main/docs/opentelemetry-example.md) for complete setup guide and [docs/trace-analysis-guide.md](https://github.com/fjacquet/nbu_exporter/blob/main/docs/trace-analysis-guide.md) for trace analysis examples.

### Multi-Version API Support

**Important**: This version adds support for NetBackup 10.0, 10.5, and 11.0 with automatic version detection. See [docs/netbackup-11-migration.md](https://github.com/fjacquet/nbu_exporter/blob/main/docs/netbackup-11-migration.md) for complete migration guide.

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

## [2.1.0] - 2025-11-09

Code quality improvements and refactoring for maintainability and SonarCloud compliance.

## [2.0.0] - 2025-11-09

OpenTelemetry integration and distributed tracing support.

## [1.2.2] - 2025-11-08

Patch release.

## [1.2.1] - 2025-11-08

GitHub Actions workflow for GitHub Pages deployment.

## [1.2.0] - 2025-11-08

API 11.0 integration support.

## [1.1.0] - 2025-11-08

Support for API 10.5 of NetBackup.

## [1.0.0] - 2025-11-08

Initial build with test server support.
