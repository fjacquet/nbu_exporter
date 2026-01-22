# Architecture

**Analysis Date:** 2026-01-22

## Pattern Overview

**Overall:** Layered Prometheus Exporter with Pull-Based Metrics Collection

**Key Characteristics:**
- On-demand metric collection triggered by Prometheus scrape requests
- Graceful degradation for partial API failures
- Optional OpenTelemetry distributed tracing support
- API version auto-detection with fallback strategy
- Interface-driven HTTP client for testability

## Layers

**Entry Point / Server Lifecycle:**
- Purpose: HTTP server setup, CLI parsing, graceful shutdown coordination
- Location: `main.go`
- Contains: `Server` struct managing HTTP server lifecycle, Prometheus registry, telemetry manager
- Depends on: `exporter`, `telemetry`, `models`, `logging`
- Used by: Entry point executable

**Prometheus Collector:**
- Purpose: Implements `prometheus.Collector` interface to expose NetBackup metrics on-demand
- Location: `internal/exporter/prometheus.go`
- Contains: `NbuCollector` struct with `Describe()` and `Collect()` methods
- Depends on: `NbuClient`, `models`, `telemetry`, OpenTelemetry trace
- Used by: Prometheus registry on each `/metrics` scrape

**NetBackup API Communication Layer:**
- Purpose: HTTP client management, API requests, pagination, request tracing
- Location: `internal/exporter/client.go`, `internal/exporter/netbackup.go`
- Contains:
  - `NbuClient` - Resty-based HTTP client with TLS, timeout, auth headers
  - `FetchStorage()` - Retrieves storage unit capacity data
  - `FetchJobDetails()` - Retrieves job data with pagination
- Depends on: `models`, go-resty, OpenTelemetry trace
- Used by: `NbuCollector.Collect()`

**Data Models and Configuration:**
- Purpose: YAML config parsing, API response DTOs, validation
- Location: `internal/models/`
- Contains:
  - `Config` - Server and NBU server settings with validation
  - `Jobs`, `Storages` - JSON:API response structures
  - Metric keys: `StorageMetricKey`, `JobMetricKey`, `JobStatusKey`
- Depends on: Standard library (yaml parsing)
- Used by: All layers for config and data structures

**Telemetry / Observability:**
- Purpose: Optional OpenTelemetry integration for distributed tracing
- Location: `internal/telemetry/`
- Contains: `Manager` - Lifecycle for TracerProvider, OTLP exporter, sampling
- Depends on: OpenTelemetry SDK/API, gRPC
- Used by: `Server`, `NbuCollector`, `NbuClient` for span creation

**Cross-Cutting Concerns:**
- Logging: `internal/logging/` - Logrus wrapper with JSON formatting and file output
- Utilities: `internal/utils/` - File operations, date parsing, pause helpers
- Testing: `internal/testutil/` - Shared test fixtures and constants

## Data Flow

**Prometheus Scrape Cycle:**

1. Prometheus sends HTTP GET to `/metrics`
2. Server's `Prometheus.HandlerFor()` calls `NbuCollector.Collect(ch)`
3. `Collect()` creates root span `prometheus.scrape` with:
   - `FetchStorage()` → span `netbackup.fetch_storage` → child spans for data processing
   - `FetchJobDetails()` with pagination → span `netbackup.fetch_job_page` for each page
4. Data accumulated in maps with pipe-delimited keys (`name|type|size`, `action|policy_type|status`)
5. Metrics converted to `prometheus.Metric` and sent to channel
6. Prometheus client formats and exposes via `/metrics` endpoint

**State Management:**

- **Configuration State**: Immutable `models.Config` loaded at startup, validated, passed to all components
- **Metric State**: Ephemeral maps in `Collect()` method - created, populated, and discarded per scrape
- **HTTP Client State**: Single `NbuClient` instance per collector with connection pooling via Resty
- **Tracer State**: Singleton `trace.Tracer` from OpenTelemetry global provider (thread-safe)
- **OpenTelemetry State**: Optional singleton `Manager` with global `TracerProvider` registration

## Key Abstractions

**Prometheus Collector Interface:**
- Purpose: Standardized metric exposition for Prometheus
- Examples: `NbuCollector` implements `prometheus.Collector`
- Pattern: `Describe()` lists metrics, `Collect()` computes and sends metrics

**NetBackupClient Interface:**
- Purpose: Abstract HTTP communication for testing without real API
- Examples: `FetchData()`, `DetectAPIVersion()`, `Close()`
- Pattern: Enables mock implementations in unit tests; `NbuClient` is primary implementation

**Metric Key Structures:**
- Purpose: Organize and parse composite metric labels from pipe-delimited strings
- Examples:
  - `StorageMetricKey{Name: "disk-pool-1", Type: "MEDIA_SERVER", Size: "free"}` → `"disk-pool-1|MEDIA_SERVER|free"`
  - `JobMetricKey{Action: "BACKUP", PolicyType: "VMWARE", Status: "0"}` → `"BACKUP|VMWARE|0"`
- Pattern: Keys and labels are reversible (string methods for maps, labels methods for Prometheus)

**Configuration Builder:**
- Purpose: Centralize URL construction with base paths and query parameters
- Examples: `BuildURL("/admin/jobs", map[string]string{"page[limit]": "100"})`
- Pattern: Eliminates URL construction duplication across API methods

## Entry Points

**HTTP Server Entry Point:**
- Location: `main.go`
- Triggers: Application startup via Cobra CLI command
- Responsibilities:
  - Validate and load configuration from YAML
  - Initialize logging (logrus with JSON format)
  - Create and start HTTP server
  - Register Prometheus collector with metric registry
  - Extract W3C trace context from incoming requests (if OTEL enabled)
  - Handle graceful shutdown on SIGINT/SIGTERM

**/metrics Endpoint Entry Point:**
- Location: `main.go` HTTP handler (via `promhttp.HandlerFor`)
- Triggers: Prometheus scrape request (typically every 15-60s)
- Responsibilities:
  - Call `NbuCollector.Collect()` to fetch fresh metrics
  - Convert to Prometheus text format
  - Return metrics with `Content-Type: text/plain`

**/health Endpoint Entry Point:**
- Location: `main.go` - `Server.healthHandler()`
- Triggers: External health checks (load balancers, Kubernetes)
- Responsibilities: Return HTTP 200 OK (simple liveness check)

## Error Handling

**Strategy:** Graceful degradation with partial metric exposition

**Patterns:**

- **Collector-Level Graceful Degradation**: If storage fetch fails, continue with job metrics. If job fetch fails, continue with storage metrics. Errors logged but don't prevent exposition.
  - Example: `FetchStorage()` error recorded in span but `Collect()` continues to `FetchJobDetails()`

- **Timeout Protection**:
  - Collection timeout: 2 minutes per `Collect()` call
  - Version detection: 30 seconds
  - Context-based cancellation propagated to all API calls

- **Span Error Recording** (when OTEL enabled):
  - `span.RecordError(err)` captures error details
  - `span.SetStatus(codes.Error, msg)` marks span as failed
  - Errors visible in trace UI for debugging

- **Telemetry Initialization Failure**:
  - If OpenTelemetry initialization fails, manager disables tracing and logs warning
  - Application continues normally without tracing (no startup failure)

- **Version Detection Fallback**:
  - Tries API versions in order: 13.0 → 12.0 → 3.0
  - Returns first successful response
  - If all fail, returns error and blocks collector creation

## Cross-Cutting Concerns

**Logging:** Logrus with JSON formatter and dual output (stdout + file)
- Format: Pretty-printed JSON with timestamp, level, message
- Levels: DEBUG (with `-d` flag), INFO, WARN, ERROR, FATAL
- Location: Configured via `internal/logging/PrepareLogs()`

**Validation:** Configuration validation with specific error messages
- Happens in `Config.Validate()` before collector creation
- Validates: ports (1-65535), schemes (http/https), API version format, duration parsing
- Blocks startup on validation failure

**Authentication:** API key header-based (Bearer token)
- Header: `Authorization: Bearer {APIKey}`
- Masking: `Config.MaskAPIKey()` shows only first/last 4 chars for safe logging

**OpenTelemetry Tracing:**
- Global propagator: W3C Trace Context
- Root spans: `prometheus.scrape` from `/metrics` endpoint
- Child spans: `netbackup.fetch_storage`, `netbackup.fetch_jobs`, `netbackup.fetch_job_page`
- Attributes: Endpoint, API version, record counts, error details
- Sampling: Configurable rate (0.0-1.0) for load control

---

*Architecture analysis: 2026-01-22*
