# Design Document

## Overview

This design document describes the integration of OpenTelemetry (OTel) distributed tracing and enhanced logging into the NBU Exporter. The integration will be optional, backward-compatible, and focused on instrumenting NetBackup API calls to enable performance diagnosis and troubleshooting. The design preserves the existing Prometheus metrics functionality while adding observability capabilities through traces and correlated logs.

### Design Goals

1. **Optional and Non-Breaking**: OpenTelemetry features are opt-in via configuration
2. **Minimal Overhead**: Tracing should not significantly impact scrape performance
3. **Production-Ready**: Robust error handling and graceful degradation
4. **Standards-Compliant**: Follow OpenTelemetry semantic conventions
5. **Operator-Friendly**: Clear configuration and actionable trace data

### Key Design Decisions

- **Preserve Prometheus Metrics**: Keep existing `prometheus/client_golang` for metrics exposition
- **Add OTel Tracing**: Instrument API calls with distributed tracing
- **Hybrid Logging**: Use OTel log bridge when tracing is enabled, fallback to logrus otherwise
- **OTLP gRPC Export**: Use standard OTLP protocol for trace export
- **Batch Processing**: Use batch span processor to minimize export overhead
- **Graceful Degradation**: Continue operating if OTel initialization or export fails

## Architecture

### Component Overview

```
┌─────────────────────────────────────────────────────────────┐
│                       NBU Exporter                          │
│                                                             │
│  ┌──────────────┐         ┌─────────────────┐             │
│  │   main.go    │────────▶│  OTel Manager   │             │
│  │              │         │  (telemetry/)   │             │
│  └──────────────┘         └─────────────────┘             │
│         │                          │                       │
│         │                          ├─ TracerProvider       │
│         │                          ├─ OTLP Exporter        │
│         │                          └─ Resource Attributes  │
│         │                                                   │
│         ▼                                                   │
│  ┌──────────────┐         ┌─────────────────┐             │
│  │   Server     │────────▶│ NbuCollector    │             │
│  │              │         │ (prometheus.go) │             │
│  └──────────────┘         └─────────────────┘             │
│                                    │                       │
│                                    ▼                       │
│                           ┌─────────────────┐             │
│                           │  NbuClient      │             │
│                           │ (instrumented)  │             │
│                           └─────────────────┘             │
│                                    │                       │
└────────────────────────────────────┼───────────────────────┘
                                     │
                                     ▼
                          ┌──────────────────┐
                          │  NetBackup API   │
                          └──────────────────┘
                                     │
                                     ▼
                          ┌──────────────────┐
                          │ OTel Collector   │
                          │  (OTLP gRPC)     │
                          └──────────────────┘
                                     │
                                     ▼
                          ┌──────────────────┐
                          │ Jaeger / Tempo   │
                          │  (Trace Backend) │
                          └──────────────────┘
```

### Data Flow

1. **Initialization Phase**:
   - Load configuration including optional OTel settings
   - Initialize OTel TracerProvider if enabled
   - Create OTLP gRPC exporter with configured endpoint
   - Set up resource attributes (service name, version, hostname)
   - Register global tracer provider

2. **Scrape Cycle**:
   - Prometheus scrapes `/metrics` endpoint
   - `NbuCollector.Collect()` creates root span "prometheus.scrape"
   - `FetchStorage()` creates child span "netbackup.fetch_storage"
   - HTTP client makes request with trace context injection
   - `FetchAllJobs()` creates child span "netbackup.fetch_jobs"
   - Pagination creates child spans "netbackup.fetch_job_page"
   - Spans record attributes (endpoint, status, duration, counts)
   - Spans are batched and exported to OTLP collector

3. **Shutdown Phase**:
   - Receive shutdown signal
   - Flush pending spans via `TracerProvider.Shutdown()`
   - Close HTTP server
   - Exit gracefully

## Components and Interfaces

### 1. Telemetry Manager (`internal/telemetry/manager.go`)

**Purpose**: Centralized management of OpenTelemetry lifecycle and configuration.

```go
package telemetry

import (
    "context"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/sdk/trace"
)

// Manager handles OpenTelemetry initialization and shutdown
type Manager struct {
    enabled        bool
    tracerProvider *trace.TracerProvider
    config         Config
}

// Config holds OpenTelemetry configuration
type Config struct {
    Enabled      bool
    Endpoint     string
    Insecure     bool
    SamplingRate float64
    ServiceName  string
    ServiceVersion string
}

// NewManager creates a new telemetry manager
func NewManager(cfg Config) (*Manager, error)

// Initialize sets up OpenTelemetry providers and exporters
func (m *Manager) Initialize(ctx context.Context) error

// Shutdown flushes pending telemetry and cleans up resources
func (m *Manager) Shutdown(ctx context.Context) error

// IsEnabled returns whether OpenTelemetry is enabled
func (m *Manager) IsEnabled() bool
```

**Key Responsibilities**:

- Initialize OTLP gRPC exporter with configured endpoint
- Create TracerProvider with batch span processor
- Configure sampling based on sampling rate
- Set resource attributes (service.name, service.version, host.name)
- Register global tracer provider
- Handle graceful shutdown with timeout

### 2. Instrumented HTTP Client (`internal/exporter/client.go`)

**Purpose**: Wrap HTTP client with automatic span creation for NetBackup API calls.

```go
package exporter

import (
    "context"
    "go.opentelemetry.io/otel/trace"
    "github.com/go-resty/resty/v2"
)

// NbuClient wraps the HTTP client with tracing capabilities
type NbuClient struct {
    cfg    models.Config
    client *resty.Client
    tracer trace.Tracer
}

// NewNbuClient creates an instrumented HTTP client
func NewNbuClient(cfg models.Config) *NbuClient

// FetchData makes an HTTP GET request with automatic span creation
func (c *NbuClient) FetchData(ctx context.Context, url string, result interface{}) error

// createSpan creates a new span for HTTP operations
func (c *NbuClient) createSpan(ctx context.Context, operation string) (context.Context, trace.Span)

// recordHTTPAttributes adds HTTP-specific attributes to span
func (c *NbuClient) recordHTTPAttributes(span trace.Span, method, url string, statusCode int, duration time.Duration)
```

**Key Responsibilities**:

- Create spans for each HTTP request
- Record HTTP semantic convention attributes (http.method, http.url, http.status_code)
- Inject trace context into outgoing requests
- Record errors and set span status
- Measure and record request duration

### 3. Instrumented Collector (`internal/exporter/prometheus.go`)

**Purpose**: Add tracing to the Prometheus collector's metric collection cycle.

```go
// Collect fetches metrics with optional tracing
func (c *NbuCollector) Collect(ch chan<- prometheus.Metric) {
    ctx, cancel := context.WithTimeout(context.Background(), collectionTimeout)
    defer cancel()

    // Create root span for scrape cycle if tracing is enabled
    ctx, span := c.createScrapeSpan(ctx)
    if span != nil {
        defer span.End()
    }

    // Existing collection logic with context propagation
    // ...
}

// createScrapeSpan creates a root span for the scrape cycle
func (c *NbuCollector) createScrapeSpan(ctx context.Context) (context.Context, trace.Span)
```

**Key Responsibilities**:

- Create root span "prometheus.scrape" for each collection cycle
- Propagate context to child operations
- Record scrape duration and metric counts
- Handle span lifecycle (creation, attributes, completion)

### 4. Instrumented Data Fetching (`internal/exporter/netbackup.go`)

**Purpose**: Add span creation to storage and job fetching operations.

```go
// FetchStorage retrieves storage data with tracing
func FetchStorage(ctx context.Context, client *NbuClient, storageMetrics map[string]float64) error {
    ctx, span := client.tracer.Start(ctx, "netbackup.fetch_storage",
        trace.WithSpanKind(trace.SpanKindClient),
    )
    defer span.End()

    // Record endpoint as attribute
    span.SetAttributes(
        attribute.String("netbackup.endpoint", storagePath),
    )

    // Existing fetch logic
    // ...

    // Record result attributes
    span.SetAttributes(
        attribute.Int("netbackup.storage_units", len(storages.Data)),
    )

    return nil
}

// FetchAllJobs retrieves job data with tracing
func FetchAllJobs(ctx context.Context, client *NbuClient, ...) error {
    ctx, span := client.tracer.Start(ctx, "netbackup.fetch_jobs",
        trace.WithSpanKind(trace.SpanKindClient),
    )
    defer span.End()

    // Record time window and endpoint
    span.SetAttributes(
        attribute.String("netbackup.endpoint", jobsPath),
        attribute.String("netbackup.time_window", scrapingInterval),
    )

    // Existing pagination logic with context propagation
    // ...

    return nil
}

// FetchJobDetails creates span for each page fetch
func FetchJobDetails(ctx context.Context, client *NbuClient, ..., offset int, ...) (int, error) {
    ctx, span := client.tracer.Start(ctx, "netbackup.fetch_job_page",
        trace.WithSpanKind(trace.SpanKindClient),
    )
    defer span.End()

    span.SetAttributes(
        attribute.Int("netbackup.page_offset", offset),
    )

    // Existing fetch logic
    // ...

    return nextOffset, nil
}
```

**Key Responsibilities**:

- Create child spans for each data fetching operation
- Record operation-specific attributes (endpoints, counts, time windows)
- Propagate context through pagination
- Record errors and set span status on failures

### 5. Configuration Extension (`internal/models/Config.go`)

**Purpose**: Extend configuration model to support OpenTelemetry settings.

```go
type Config struct {
    Server struct {
        // ... existing fields
    } `yaml:"server"`

    NbuServer struct {
        // ... existing fields
    } `yaml:"nbuserver"`

    OpenTelemetry struct {
        Enabled      bool    `yaml:"enabled"`
        Endpoint     string  `yaml:"endpoint"`
        Insecure     bool    `yaml:"insecure"`
        SamplingRate float64 `yaml:"samplingRate"`
    } `yaml:"opentelemetry"`
}

// Validate extends existing validation to check OTel config
func (c *Config) Validate() error {
    // ... existing validation

    // Validate OpenTelemetry configuration if enabled
    if c.OpenTelemetry.Enabled {
        if c.OpenTelemetry.Endpoint == "" {
            return errors.New("OpenTelemetry endpoint is required when enabled")
        }
        if c.OpenTelemetry.SamplingRate < 0 || c.OpenTelemetry.SamplingRate > 1 {
            return errors.New("OpenTelemetry sampling rate must be between 0 and 1")
        }
    }

    return nil
}
```

## Data Models

### Span Hierarchy

```
prometheus.scrape (root)
├── netbackup.fetch_storage
│   └── http.request (GET /storage/storage-units)
└── netbackup.fetch_jobs
    ├── netbackup.fetch_job_page (offset=0)
    │   └── http.request (GET /admin/jobs?offset=0)
    ├── netbackup.fetch_job_page (offset=1)
    │   └── http.request (GET /admin/jobs?offset=1)
    └── netbackup.fetch_job_page (offset=N)
        └── http.request (GET /admin/jobs?offset=N)
```

### Span Attributes

#### Root Span (prometheus.scrape)

- `scrape.duration_ms`: Total scrape duration
- `scrape.storage_metrics_count`: Number of storage metrics collected
- `scrape.job_metrics_count`: Number of job metrics collected
- `scrape.status`: "success" or "partial_failure"

#### Storage Fetch Span (netbackup.fetch_storage)

- `netbackup.endpoint`: "/storage/storage-units"
- `netbackup.storage_units`: Number of storage units retrieved
- `netbackup.api_version`: API version used

#### Job Fetch Span (netbackup.fetch_jobs)

- `netbackup.endpoint`: "/admin/jobs"
- `netbackup.time_window`: Scraping interval (e.g., "5m")
- `netbackup.start_time`: Start of time window (ISO 8601)
- `netbackup.total_jobs`: Total jobs retrieved
- `netbackup.total_pages`: Number of pages fetched

#### Job Page Span (netbackup.fetch_job_page)

- `netbackup.page_offset`: Pagination offset
- `netbackup.page_number`: Page number (calculated)
- `netbackup.jobs_in_page`: Number of jobs in this page

#### HTTP Request Span (http.request)

- `http.method`: "GET"
- `http.url`: Full request URL
- `http.status_code`: HTTP response status code
- `http.request_content_length`: Request size in bytes
- `http.response_content_length`: Response size in bytes
- `http.duration_ms`: Request duration in milliseconds

### Resource Attributes

Set once during initialization:

- `service.name`: "nbu-exporter"
- `service.version`: Application version from build info
- `host.name`: System hostname
- `netbackup.server`: Target NetBackup server hostname
- `netbackup.api_version`: Configured or detected API version

## Error Handling

### Initialization Errors

**Scenario**: OTel initialization fails (invalid endpoint, network issues)

**Handling**:

1. Log warning with error details
2. Set `Manager.enabled = false`
3. Continue application startup without tracing
4. Existing Prometheus metrics remain functional

```go
func (m *Manager) Initialize(ctx context.Context) error {
    if !m.config.Enabled {
        return nil
    }

    if err := m.setupExporter(ctx); err != nil {
        log.Warnf("Failed to initialize OpenTelemetry: %v. Continuing without tracing.", err)
        m.enabled = false
        return nil // Don't fail startup
    }

    return nil
}
```

### Export Errors

**Scenario**: OTLP collector is unavailable during span export

**Handling**:

1. Batch span processor retries with exponential backoff
2. Spans are dropped after retry limit to prevent memory buildup
3. Log export failures at DEBUG level to avoid log spam
4. Metric collection continues unaffected

### Span Creation Errors

**Scenario**: Tracer is nil or span creation fails

**Handling**:

1. Check if tracer is initialized before creating spans
2. Use nil-safe span operations
3. Continue operation without tracing
4. No impact on metric collection

```go
func (c *NbuClient) createSpan(ctx context.Context, operation string) (context.Context, trace.Span) {
    if c.tracer == nil {
        return ctx, nil
    }
    return c.tracer.Start(ctx, operation)
}
```

### Context Timeout

**Scenario**: Scrape timeout occurs during traced operation

**Handling**:

1. Context cancellation propagates to all child spans
2. Spans are ended with status "cancelled"
3. Partial spans are exported
4. Timeout error is logged and returned

## Testing Strategy

### Unit Tests

1. **Telemetry Manager Tests**
   - Test initialization with valid configuration
   - Test initialization with invalid endpoint
   - Test shutdown with pending spans
   - Test disabled mode (no-op behavior)

2. **Instrumented Client Tests**
   - Test span creation for HTTP requests
   - Test attribute recording (method, URL, status)
   - Test error recording and span status
   - Test operation without tracer (nil-safe)

3. **Collector Tests**
   - Test scrape span creation
   - Test context propagation to fetch operations
   - Test span attributes for successful scrape
   - Test span attributes for failed scrape

4. **Configuration Tests**
   - Test OTel config validation
   - Test default values
   - Test invalid sampling rates
   - Test missing endpoint when enabled

### Integration Tests

1. **End-to-End Tracing**
   - Start exporter with OTel enabled
   - Trigger Prometheus scrape
   - Verify span hierarchy in test collector
   - Verify span attributes match expectations

2. **Backward Compatibility**
   - Start exporter without OTel config
   - Verify normal operation
   - Verify no trace spans created
   - Verify Prometheus metrics work

3. **Error Scenarios**
   - Start with invalid OTLP endpoint
   - Verify graceful degradation
   - Verify metrics still collected
   - Verify warning logged

4. **Performance**
   - Benchmark scrape with tracing disabled
   - Benchmark scrape with tracing enabled (sampling=1.0)
   - Benchmark scrape with tracing enabled (sampling=0.1)
   - Verify overhead is < 5% for sampling=0.1

### Manual Testing

1. **Docker Compose Setup**
   - Run exporter with OTel Collector and Jaeger
   - Trigger scrapes and view traces in Jaeger UI
   - Verify trace hierarchy and attributes
   - Test slow API scenarios

2. **Configuration Validation**
   - Test various sampling rates
   - Test TLS vs insecure connections
   - Test missing optional fields
   - Test invalid configurations

## Dependencies

### New Go Modules

```go
require (
    go.opentelemetry.io/otel v1.32.0
    go.opentelemetry.io/otel/sdk v1.32.0
    go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.32.0
    go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.32.0
    go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.57.0
)
```

### Existing Dependencies (Unchanged)

- `github.com/prometheus/client_golang` - Prometheus metrics
- `github.com/go-resty/resty/v2` - HTTP client
- `github.com/sirupsen/logrus` - Logging (used when OTel disabled)
- `github.com/spf13/cobra` - CLI framework
- `gopkg.in/yaml.v2` - Configuration parsing

## Configuration Example

```yaml
server:
  port: "2112"
  host: "0.0.0.0"
  uri: "/metrics"
  scrapingInterval: "5m"
  logName: "nbu_exporter.log"

nbuserver:
  host: "nbu-master.example.com"
  port: "1556"
  scheme: "https"
  uri: "/netbackup"
  apiKey: "your-api-key-here"
  apiVersion: "13.0"
  insecureSkipVerify: false

# Optional: OpenTelemetry configuration
opentelemetry:
  enabled: true
  endpoint: "localhost:4317"  # OTLP gRPC endpoint
  insecure: true              # Use insecure connection (for development)
  samplingRate: 0.1           # Sample 10% of scrapes
```

## Deployment Considerations

### Docker Compose Example

```yaml
version: '3.8'

services:
  nbu-exporter:
    image: nbu-exporter:latest
    ports:
      - "2112:2112"
    volumes:
      - ./config.yaml:/config.yaml
    command: ["--config", "/config.yaml"]
    depends_on:
      - otel-collector

  otel-collector:
    image: otel/opentelemetry-collector:latest
    ports:
      - "4317:4317"  # OTLP gRPC
      - "4318:4318"  # OTLP HTTP
    volumes:
      - ./otel-collector-config.yaml:/etc/otel-collector-config.yaml
    command: ["--config=/etc/otel-collector-config.yaml"]

  jaeger:
    image: jaegertracing/all-in-one:latest
    ports:
      - "16686:16686"  # Jaeger UI
      - "14250:14250"  # gRPC
    environment:
      - COLLECTOR_OTLP_ENABLED=true
```

### Performance Impact

**Expected Overhead**:

- Tracing disabled: 0% overhead
- Tracing enabled (sampling=0.1): < 2% overhead
- Tracing enabled (sampling=1.0): < 5% overhead

**Memory Impact**:

- Batch span processor buffers up to 2048 spans
- Estimated memory: ~1-2 MB for span buffer
- Spans exported every 5 seconds or when buffer is full

### Monitoring

**Key Metrics to Monitor**:

- Scrape duration (existing Prometheus metric)
- Trace export success rate (OTel internal metrics)
- Span drop rate (OTel internal metrics)
- OTLP connection status

## Migration Path

### Phase 1: Add Infrastructure (Non-Breaking)

1. Add telemetry package with Manager
2. Extend configuration model
3. Add OTel dependencies to go.mod
4. No functional changes to existing code

### Phase 2: Instrument HTTP Client

1. Add tracer to NbuClient
2. Instrument FetchData method
3. Add span creation and attribute recording
4. Maintain backward compatibility (nil-safe)

### Phase 3: Instrument Collectors

1. Add span creation to Collect method
2. Instrument FetchStorage and FetchAllJobs
3. Add span attributes for operations
4. Test with and without OTel enabled

### Phase 4: Documentation and Examples

1. Update README with OTel configuration
2. Add docker-compose example
3. Document trace queries
4. Add troubleshooting guide

## Future Enhancements

### Potential Additions (Out of Scope)

1. **OTel Metrics**: Replace Prometheus client with OTel metrics SDK
   - Requires significant refactoring
   - May impact existing Prometheus integrations
   - Consider for v2.0

2. **OTel Logs**: Full log bridge implementation
   - Replace logrus entirely
   - Structured logging with trace correlation
   - Export logs via OTLP

3. **Custom Metrics**: Add OTel metrics for exporter internals
   - Scrape duration histogram
   - API call duration histogram
   - Error rates by endpoint

4. **Exemplars**: Link Prometheus metrics to traces
   - Requires Prometheus 2.26+
   - Enables jumping from metrics to traces
   - Useful for debugging specific scrapes
