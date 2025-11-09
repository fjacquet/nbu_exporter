# NBU Exporter - Prometheus Exporter for Veritas NetBackup

A production-ready Prometheus exporter that collects backup job statistics and storage metrics from Veritas NetBackup REST API, exposing them for monitoring and visualization in Grafana.

![CI](https://github.com/fjacquet/nbu_exporter/actions/workflows/ci.yml/badge.svg)
![Code Analysis](https://github.com/fjacquet/nbu_exporter/actions/workflows/codeql-analysis.yml/badge.svg)
[![Code Smells](https://sonarcloud.io/api/project_badges/measure?project=fjacquet_nbu_exporter&metric=code_smells)](https://sonarcloud.io/summary/new_code?id=fjacquet_nbu_exporter)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=fjacquet_nbu_exporter&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=fjacquet_nbu_exporter)
[![Documentation](https://img.shields.io/badge/docs-github%20pages-blue)](https://fjacquet.github.io/nbu_exporter/)

## Features

- **Job Metrics Collection**: Aggregates backup job statistics by type, policy, and status
- **Storage Monitoring**: Tracks storage unit capacity (free/used bytes) for disk-based storage
- **Prometheus Integration**: Native Prometheus metrics exposition via HTTP endpoint
- **OpenTelemetry Tracing**: Optional distributed tracing for performance analysis and troubleshooting
- **Multi-Version API Support**: Automatic detection of NetBackup API versions (3.0, 12.0, 13.0)
- **Configurable Scraping**: Adjustable time windows for historical job data collection
- **Health Checks**: Built-in `/health` endpoint for monitoring exporter status
- **Graceful Shutdown**: Proper signal handling with configurable shutdown timeout
- **Security**: Configurable TLS verification, API key masking in logs
- **Performance**: HTTP client connection pooling, context-aware operations, minimal tracing overhead

## Quick Start

### Prerequisites

- Go 1.25 or later
- Veritas NetBackup 10.0 or later (see version support matrix below)
- Access to NetBackup REST API
- NetBackup API key (generated from NBU UI)

### Version Support Matrix

| NetBackup Version | API Version | Support Status | Notes |
|-------------------|-------------|----------------|-------|
| 11.0+             | 13.0        | ✅ Fully Supported | Latest version with all features |
| 10.5              | 12.0        | ✅ Fully Supported | Current stable version |
| 10.0 - 10.4       | 3.0         | ✅ Legacy Support | Basic functionality maintained |

**Automatic Version Detection**: The exporter automatically detects the highest supported API version available on your NetBackup server using intelligent fallback logic (13.0 → 12.0 → 3.0). No manual configuration required unless you need to override the detected version for testing or performance optimization.

### Installation

```bash
# Clone the repository
git clone https://github.com/fjacquet/nbu_exporter.git
cd nbu_exporter

# Build the binary
make cli

# Or build manually
go build -o bin/nbu_exporter .
```

### Configuration

Create a `config.yaml` file:

```yaml
server:
    host: "localhost"
    port: "2112"
    uri: "/metrics"
    scrapingInterval: "1h"  # Time window for job collection
    logName: "log/nbu-exporter.log"

nbuserver:
    scheme: "https"
    uri: "/netbackup"
    domain: "my.domain"
    domainType: "NT"
    host: "master.my.domain"
    port: "1556"
    apiVersion: "13.0"  # Optional: NetBackup API version (13.0, 12.0, or 3.0)
                        # If omitted, the exporter will automatically detect the version
    apiKey: "your-api-key-here"
    contentType: "application/vnd.netbackup+json; version=13.0"
    insecureSkipVerify: false  # Set to true only for testing environments

# Optional: OpenTelemetry distributed tracing (see OpenTelemetry Integration section)
# opentelemetry:
#     enabled: true
#     endpoint: "localhost:4317"
#     insecure: true
#     samplingRate: 0.1
```

**Important**: Replace `your-api-key-here` with your actual NetBackup API key.

### Running

```bash
# Run with configuration file
./bin/nbu_exporter --config config.yaml

# Run with debug logging
./bin/nbu_exporter --config config.yaml --debug

# Run via Makefile
make run-cli
```

The exporter will start and expose metrics at `http://localhost:2112/metrics`.

**Optional**: To enable distributed tracing with OpenTelemetry, see the [OpenTelemetry Integration](#opentelemetry-integration) section below.

## Usage

### Command Line Options

```bash
./bin/nbu_exporter --help

Flags:
  -c, --config string   Path to configuration file (required)
  -d, --debug           Enable debug mode
  -h, --help            Help for nbu_exporter
```

### Endpoints

- **Metrics**: `http://localhost:2112/metrics` - Prometheus metrics endpoint
- **Health**: `http://localhost:2112/health` - Health check endpoint

### Prometheus Configuration

Add to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'netbackup'
    static_configs:
      - targets: ['localhost:2112']
    scrape_interval: 60s
    scrape_timeout: 30s
```

## Metrics Exposed

### Job Metrics

- `nbu_jobs_count{action, policy_type, status}` - Number of jobs by type, policy, and status
- `nbu_jobs_size_bytes{action, policy_type, status}` - Total bytes transferred by jobs
- `nbu_jobs_status_count{action, status}` - Job count aggregated by action and status

### Storage Metrics

- `nbu_storage_bytes{name, type, size}` - Storage unit capacity
  - `size="free"` - Available capacity in bytes
  - `size="used"` - Used capacity in bytes

### System Metrics

- `nbu_api_version{version}` - Currently active NetBackup API version (13.0, 12.0, or 3.0)

**Note**: Tape storage units are excluded from metrics collection.

## OpenTelemetry Integration

The exporter supports optional distributed tracing via OpenTelemetry, enabling you to diagnose slow scrapes, identify performance bottlenecks in NetBackup API calls, and understand the complete request flow through your monitoring infrastructure.

### Features

- **Distributed Tracing**: Track the complete lifecycle of each Prometheus scrape
- **API Call Instrumentation**: Detailed spans for every NetBackup API request
- **W3C Trace Context Propagation**: Automatic trace context injection into NetBackup API requests
- **Performance Analysis**: Identify slow endpoints and optimize scraping intervals
- **Trace Correlation**: Link logs to traces for comprehensive troubleshooting
- **Zero Overhead When Disabled**: No performance impact when tracing is not enabled
- **Configurable Sampling**: Control tracing overhead with adjustable sampling rates
- **Graceful Degradation**: Continues operating normally if tracing initialization fails

### Quick Start

1. **Enable OpenTelemetry in your configuration:**

```yaml
opentelemetry:
    enabled: true
    endpoint: "localhost:4317"  # Your OTLP collector endpoint
    insecure: true              # Use false for production with TLS
    samplingRate: 0.1           # Trace 10% of scrapes
```

2. **Start an OpenTelemetry Collector** (see Docker Compose example below)

3. **Restart the exporter** and view traces in your observability backend (Jaeger, Tempo, etc.)

### Configuration Options

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `enabled` | bool | No | false | Enable or disable OpenTelemetry tracing |
| `endpoint` | string | Yes* | - | OTLP gRPC endpoint (e.g., "localhost:4317") |
| `insecure` | bool | No | false | Use insecure connection (no TLS) for OTLP export |
| `samplingRate` | float | No | 1.0 | Sampling rate (0.0 to 1.0) for trace collection |

*Required when `enabled` is `true`

### Sampling Rate Behavior

The `samplingRate` controls what percentage of scrape cycles are traced:

- **1.0** (100%): Trace all scrapes - useful for debugging and development
- **0.1** (10%): Trace 10% of scrapes - recommended for production monitoring
- **0.01** (1%): Trace 1% of scrapes - minimal overhead for high-frequency scraping

**Performance Impact:**
- Tracing disabled: 0% overhead
- samplingRate 0.1: < 2% overhead
- samplingRate 1.0: < 5% overhead

### Trace Hierarchy

Each Prometheus scrape creates a trace with the following span structure:

```
prometheus.scrape (root span)
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

Traces include rich attributes for analysis:

**Resource Attributes (all spans):**
- `service.name`: Service identifier ("nbu-exporter")
- `service.version`: Exporter version
- `host.name`: Hostname where exporter is running
- `netbackup.server`: NetBackup master server hostname

**Root Span (prometheus.scrape):**
- `scrape.duration_ms`: Total scrape duration
- `scrape.storage_metrics_count`: Number of storage metrics collected
- `scrape.job_metrics_count`: Number of job metrics collected
- `scrape.status`: "success" or "partial_failure"

**Storage Fetch (netbackup.fetch_storage):**
- `netbackup.endpoint`: API endpoint path
- `netbackup.storage_units`: Number of storage units retrieved
- `netbackup.api_version`: API version used

**Job Fetch (netbackup.fetch_jobs):**
- `netbackup.endpoint`: API endpoint path
- `netbackup.time_window`: Scraping interval
- `netbackup.total_jobs`: Total jobs retrieved
- `netbackup.total_pages`: Number of pages fetched

**Job Page Fetch (netbackup.fetch_job_page):**
- `netbackup.page_offset`: Page offset for pagination
- `netbackup.page_number`: Calculated page number
- `netbackup.jobs_in_page`: Number of jobs in this page

**Version Detection (netbackup.detect_version):**
- `netbackup.attempted_versions`: List of API versions attempted
- `netbackup.detected_version`: Successfully detected API version
- Span events for each version attempt with success/failure status

**HTTP Request (http.request):**
- `http.method`: HTTP method (GET, POST, etc.)
- `http.url`: Full request URL
- `http.status_code`: HTTP response status code
- `http.duration_ms`: Request duration in milliseconds
- `http.request_content_length`: Request body size in bytes
- `http.response_content_length`: Response body size in bytes

### Example Configurations

**Development (trace everything):**

```yaml
opentelemetry:
    enabled: true
    endpoint: "localhost:4317"
    insecure: true
    samplingRate: 1.0
```

**Production (sample 10%):**

```yaml
opentelemetry:
    enabled: true
    endpoint: "otel-collector.prod.example.com:4317"
    insecure: false
    samplingRate: 0.1
```

**High-frequency scraping (minimal overhead):**

```yaml
opentelemetry:
    enabled: true
    endpoint: "otel-collector.example.com:4317"
    insecure: false
    samplingRate: 0.01
```

### Docker Compose with OpenTelemetry

See the complete example in the "Docker Compose with OpenTelemetry" section below.

### Analyzing Traces

**Identify slow API calls:**

1. Open your trace backend (Jaeger, Tempo, etc.)
2. Search for traces with service name "nbu-exporter"
3. Sort by duration to find slow scrapes
4. Drill down into spans to identify bottlenecks

**Common queries:**

- Find scrapes taking > 30 seconds: `duration > 30s`
- Find failed API calls: `http.status_code >= 400`
- Find specific endpoints: `netbackup.endpoint = "/admin/jobs"`
- Find high pagination: `netbackup.total_pages > 10`

**Example trace analysis:**

```
Trace: prometheus.scrape (45.2s total)
├── netbackup.fetch_storage (2.1s) ✓ Normal
└── netbackup.fetch_jobs (43.1s) ⚠️ Slow!
    ├── netbackup.fetch_job_page (15.2s) ⚠️ Bottleneck
    ├── netbackup.fetch_job_page (14.8s) ⚠️ Bottleneck
    └── netbackup.fetch_job_page (13.1s) ⚠️ Bottleneck
```

**Diagnosis**: Job pagination is slow. Consider:
- Reducing `scrapingInterval` to fetch fewer jobs
- Checking NetBackup server performance
- Verifying network latency

For comprehensive trace analysis techniques, query examples, and troubleshooting scenarios, see the [Trace Analysis Guide](docs/trace-analysis-guide.md).

### Troubleshooting

**Traces not appearing:**

1. Verify OpenTelemetry is enabled: `enabled: true`
2. Check collector endpoint is reachable: `telnet localhost 4317`
3. Enable debug logging: `./nbu_exporter --config config.yaml --debug`
4. Check exporter logs for connection errors
5. Verify sampling rate is not too low (try `samplingRate: 1.0` for testing)

**Connection refused errors:**

```
WARN[0001] Failed to initialize OpenTelemetry: connection refused
```

**Solution**: Ensure your OTLP collector is running and accessible at the configured endpoint.

**High overhead:**

If tracing is causing performance issues:

1. Reduce sampling rate: `samplingRate: 0.1` or lower
2. Verify batch span processor is configured (automatic)
3. Check collector performance and resource limits
4. Consider disabling tracing temporarily: `enabled: false`

**TLS certificate errors:**

```
ERROR: x509: certificate signed by unknown authority
```

**Solution**: Either:
- Install the collector's CA certificate on the exporter host
- Use `insecure: true` for development/testing (not recommended for production)
- Configure the collector with a valid TLS certificate

### Backward Compatibility

OpenTelemetry integration is fully backward compatible:

- **No configuration changes required**: Existing deployments work without modification
- **Opt-in feature**: Tracing is disabled by default
- **Zero overhead when disabled**: No performance impact on existing installations
- **Graceful degradation**: If initialization fails, the exporter continues operating normally

## Grafana Dashboard

A pre-built Grafana dashboard is included in the `grafana/` directory:

1. Import `grafana/NBU Statistics-1629904585394.json` into your Grafana instance
2. Configure the Prometheus datasource
3. View NetBackup job statistics and storage utilization

## Docker Deployment

### Build Docker Image

```bash
# Using Makefile
make docker

# Or manually
docker build -t nbu_exporter .
```

The Docker image uses a multi-stage build for optimal size and includes:
- Alpine Linux base for minimal footprint
- CA certificates for HTTPS connections
- Pre-configured log directory

### Run Container

```bash
# Using Makefile (runs on port 2112)
make run-docker

# Or manually with custom configuration
docker run -d \
  --name nbu_exporter \
  -p 2112:2112 \
  -v $(pwd)/config.yaml:/etc/nbu_exporter/config.yaml \
  -v $(pwd)/log:/var/log/nbu_exporter \
  nbu_exporter

# With automatic version detection
docker run -d \
  --name nbu_exporter \
  -p 2112:2112 \
  -v $(pwd)/config-auto-detect.yaml:/etc/nbu_exporter/config.yaml \
  nbu_exporter

# With environment variables (optional)
docker run -d \
  --name nbu_exporter \
  -p 2112:2112 \
  -e NBU_API_KEY=your-api-key \
  -v $(pwd)/config.yaml:/etc/nbu_exporter/config.yaml \
  nbu_exporter
```

### Docker Compose Example

```yaml
version: '3.8'
services:
  nbu_exporter:
    image: nbu_exporter:latest
    container_name: nbu_exporter
    ports:
      - "2112:2112"
    volumes:
      - ./config.yaml:/etc/nbu_exporter/config.yaml:ro
      - ./log:/var/log/nbu_exporter
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:2112/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
```

### Docker Compose with OpenTelemetry

Complete example with OpenTelemetry Collector and Jaeger for trace visualization:

```yaml
version: '3.8'

services:
  nbu_exporter:
    image: nbu_exporter:latest
    container_name: nbu_exporter
    ports:
      - "2112:2112"
    volumes:
      - ./config.yaml:/etc/nbu_exporter/config.yaml:ro
      - ./log:/var/log/nbu_exporter
    environment:
      - OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4317
    restart: unless-stopped
    depends_on:
      - otel-collector
    networks:
      - monitoring

  otel-collector:
    image: otel/opentelemetry-collector:latest
    container_name: otel-collector
    command: ["--config=/etc/otel-collector-config.yaml"]
    volumes:
      - ./otel-collector-config.yaml:/etc/otel-collector-config.yaml:ro
    ports:
      - "4317:4317"   # OTLP gRPC receiver
      - "4318:4318"   # OTLP HTTP receiver
      - "8888:8888"   # Prometheus metrics
      - "13133:13133" # Health check
    restart: unless-stopped
    depends_on:
      - jaeger
    networks:
      - monitoring

  jaeger:
    image: jaegertracing/all-in-one:latest
    container_name: jaeger
    ports:
      - "16686:16686" # Jaeger UI
      - "14250:14250" # gRPC
      - "14268:14268" # HTTP
    environment:
      - COLLECTOR_OTLP_ENABLED=true
    restart: unless-stopped
    networks:
      - monitoring

networks:
  monitoring:
    driver: bridge
```

**OpenTelemetry Collector Configuration** (`otel-collector-config.yaml`):

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318

processors:
  batch:
    timeout: 10s
    send_batch_size: 1024
  
  memory_limiter:
    check_interval: 1s
    limit_mib: 512

exporters:
  jaeger:
    endpoint: jaeger:14250
    tls:
      insecure: true
  
  logging:
    loglevel: info

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [memory_limiter, batch]
      exporters: [jaeger, logging]
```

**Configuration for OpenTelemetry** (`config.yaml`):

```yaml
server:
    host: "localhost"
    port: "2112"
    uri: "/metrics"
    scrapingInterval: "1h"
    logName: "log/nbu-exporter.log"

nbuserver:
    scheme: "https"
    host: "master.my.domain"
    port: "1556"
    apiKey: "your-api-key"
    # ... other settings

opentelemetry:
    enabled: true
    endpoint: "otel-collector:4317"
    insecure: true
    samplingRate: 1.0  # Trace all scrapes for testing
```

**Start the stack:**

```bash
docker-compose up -d
```

**Access the services:**

- **Prometheus Metrics**: http://localhost:2112/metrics
- **Jaeger UI**: http://localhost:16686
- **Collector Metrics**: http://localhost:8888/metrics
- **Collector Health**: http://localhost:13133

### Verify Container

```bash
# Check container logs
docker logs nbu_exporter

# Check metrics endpoint
curl http://localhost:2112/metrics

# Check health endpoint
curl http://localhost:2112/health
```

## Development

### Project Structure

```
nbu_exporter/
├── internal/
│   ├── exporter/
│   │   ├── client.go       # HTTP client with connection pooling
│   │   ├── metrics.go      # Structured metric key types
│   │   ├── netbackup.go    # NetBackup API data fetching
│   │   └── prometheus.go   # Prometheus collector implementation
│   ├── logging/
│   │   └── logging.go      # Centralized logging setup
│   ├── models/
│   │   ├── Config.go       # Configuration with validation
│   │   ├── Jobs.go         # Job data structures
│   │   └── Storage.go      # Storage data structures
│   └── utils/
│       ├── date.go         # Date/time conversion for NBU API
│       ├── file.go         # File operations and YAML parsing
│       └── pause.go        # Timing utilities
├── main.go                 # Application entry point
├── config.yaml             # Configuration file
└── Makefile                # Build automation
```

### Building

```bash
# Build CLI binary
make cli

# Run tests
make test

# Build Docker image
make docker

# Build all
make all

# Clean build artifacts
make clean
```

### Testing

The exporter includes comprehensive test coverage for multi-version API support:

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with race detection
go test -race ./...

# Run specific test suites
go test ./internal/exporter -run TestVersionDetection
go test ./internal/exporter -run TestBackwardCompatibility
go test ./internal/exporter -run TestEndToEnd

# Run OpenTelemetry integration tests
go test ./internal/exporter -run TestIntegration

# Run performance benchmarks
go test ./internal/exporter -bench=. -benchmem
```

**Test Coverage**:

- Unit tests for version detection with mock servers
- Integration tests for all API versions (3.0, 12.0, 13.0)
- Backward compatibility tests for existing configurations
- End-to-end workflow tests with fallback scenarios
- OpenTelemetry integration tests with mock collector
- Performance benchmarks for tracing overhead validation
- Metrics consistency tests across versions

**Benchmark Tests**:

The exporter includes benchmark tests to measure OpenTelemetry tracing overhead:

- `BenchmarkFetchData_WithoutTracing`: Baseline performance without tracing
- `BenchmarkFetchData_WithTracing_FullSampling`: Performance with 100% sampling
- `BenchmarkFetchData_WithTracing_PartialSampling`: Performance with 10% sampling
- `BenchmarkSpanCreation`: Overhead of span creation
- `BenchmarkAttributeRecording`: Overhead of attribute recording

Run benchmarks to verify performance on your system:

```bash
go test ./internal/exporter -bench=BenchmarkFetchData -benchmem -benchtime=10s
```

### Debugging

Install Delve debugger:

```bash
go install github.com/go-delve/delve/cmd/dlv@latest
```

Run with debugger:

```bash
dlv debug . -- --config config.yaml --debug
```

## API Version Configuration

### Automatic Version Detection

By default, the exporter automatically detects the highest supported API version available on your NetBackup server. This happens at startup and follows this detection order:

1. **Try API 13.0** (NetBackup 11.0+)
2. **Try API 12.0** (NetBackup 10.5)
3. **Try API 3.0** (NetBackup 10.0-10.4)

The exporter uses the first version that responds successfully. This allows you to deploy the same exporter binary across different NetBackup environments without configuration changes.

**Startup Log Example:**

```
INFO[0000] Starting NBU Exporter
INFO[0001] Attempting API version detection
DEBUG[0001] Trying API version 13.0
INFO[0002] Detected NetBackup API version: 13.0
INFO[0002] Successfully connected to NetBackup API
```

### Manual Version Configuration

You can explicitly specify the API version in your configuration file to:

- Skip automatic detection for faster startup
- Override detection for testing or troubleshooting
- Lock to a specific version for consistency

**Example configurations:**

```yaml
# NetBackup 11.0 - Explicit version
nbuserver:
    apiVersion: "13.0"
    # ... other settings

# NetBackup 10.5 - Explicit version
nbuserver:
    apiVersion: "12.0"
    # ... other settings

# NetBackup 10.0-10.4 - Explicit version
nbuserver:
    apiVersion: "3.0"
    # ... other settings

# Automatic detection - Omit apiVersion field
nbuserver:
    # apiVersion not specified - will auto-detect
    # ... other settings
```

### Version Detection Behavior

- **Detection Time**: Adds 1-3 seconds to startup (one lightweight API call per version attempt)
- **Retry Logic**: Automatically retries transient failures with exponential backoff (max 3 attempts)
- **Error Handling**: Distinguishes between version incompatibility (HTTP 406) and other errors
- **Authentication**: Fails immediately on authentication errors (HTTP 401) without trying other versions
- **Logging**: Each version attempt is logged at DEBUG level for troubleshooting
- **Fallback Order**: Tries versions in descending order: 13.0 → 12.0 → 3.0

## Configuration Reference

### Server Section

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `host` | string | Yes | - | Server bind address |
| `port` | string | Yes | - | Server port (1-65535) |
| `uri` | string | Yes | - | Metrics endpoint path |
| `scrapingInterval` | duration | Yes | - | Time window for job collection (e.g., "1h", "30m") |
| `logName` | string | Yes | - | Log file path |

### NBU Server Section

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `scheme` | string | Yes | - | Protocol (http/https) |
| `uri` | string | Yes | - | API base path |
| `domain` | string | Yes | - | NetBackup domain |
| `domainType` | string | Yes | - | Domain type (NT, vx, etc.) |
| `host` | string | Yes | - | NetBackup master server hostname |
| `port` | string | Yes | - | API port (typically 1556) |
| `apiVersion` | string | No | Auto-detect | NetBackup API version (13.0, 12.0, or 3.0). If omitted, automatically detects the highest supported version. |
| `apiKey` | string | Yes | - | NetBackup API key |
| `contentType` | string | Yes | - | API content type header |
| `insecureSkipVerify` | bool | No | false | Skip TLS certificate verification (not recommended for production) |

### OpenTelemetry Section (Optional)

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `enabled` | bool | No | false | Enable or disable OpenTelemetry distributed tracing |
| `endpoint` | string | Yes* | - | OTLP gRPC endpoint for trace export (e.g., "localhost:4317") |
| `insecure` | bool | No | false | Use insecure connection (no TLS) for OTLP export |
| `samplingRate` | float | No | 1.0 | Sampling rate for trace collection (0.0 to 1.0) |

*Required when `enabled` is `true`

## Security Considerations

### API Key Management

- Never commit API keys to version control
- Use environment variables or secure secret management
- API keys are masked in debug logs (shows only first/last 4 characters)

### TLS Configuration

- `insecureSkipVerify: false` (default) - Validates TLS certificates (recommended)
- `insecureSkipVerify: true` - Skips TLS validation (use only for testing)

### Network Security

- The exporter includes `ReadHeaderTimeout` to prevent Slowloris attacks
- Graceful shutdown with 10-second timeout prevents resource leaks

## Troubleshooting

### Common Issues

**Configuration file not found**

```bash
Error: config file not found: config.yaml
```

Solution: Ensure the config file path is correct and the file exists.

**Invalid configuration**

```bash
Error: invalid configuration: invalid port: must be between 1 and 65535
```

Solution: Check all configuration values meet validation requirements.

**TLS certificate verification failed**

```bash
Error: x509: certificate signed by unknown authority
```

Solution: Either install the CA certificate or set `insecureSkipVerify: true` (not recommended for production).

**API authentication failed**

```bash
Error: failed to fetch storage data: HTTP 401 Unauthorized
```

Solution: Verify your API key is valid and has appropriate permissions.

### Version-Related Issues

**Version detection failed**

```bash
ERROR: Failed to detect compatible NetBackup API version.
Attempted versions: 13.0, 12.0, 3.0
Last error: HTTP 406 Not Acceptable
```

**Possible causes:**

1. NetBackup server is running a version older than 10.0
2. Network connectivity issues between exporter and NetBackup server
3. API endpoint is not accessible or blocked by firewall
4. Authentication credentials are invalid

**Troubleshooting steps:**

1. **Verify NetBackup version:**

   ```bash
   # On NetBackup master server
   bpgetconfig -g | grep VERSION
   ```

2. **Check network connectivity:**

   ```bash
   curl -k https://nbu-master:1556/netbackup/
   ```

3. **Verify API accessibility:**

   ```bash
   curl -k -H "Authorization: YOUR_API_KEY" \
        -H "Accept: application/vnd.netbackup+json;version=13.0" \
        https://nbu-master:1556/netbackup/admin/jobs?page[limit]=1
   ```

4. **Try manual version configuration:**

   ```yaml
   nbuserver:
       apiVersion: "12.0"  # Try 12.0 or 3.0 based on your NetBackup version
   ```

**Configured version not supported**

```bash
ERROR: Configured API version 13.0 is not supported by the NetBackup server (HTTP 406 Not Acceptable).
```

**Solution:** Your NetBackup server doesn't support the configured API version. Either:

1. **Remove the apiVersion field** to enable automatic detection:

   ```yaml
   nbuserver:
       # apiVersion: "13.0"  # Comment out or remove this line
       host: "master.my.domain"
       # ... other settings
   ```

2. **Specify a compatible version** based on your NetBackup version:
   - NetBackup 11.0+ → `apiVersion: "13.0"`
   - NetBackup 10.5 → `apiVersion: "12.0"`
   - NetBackup 10.0-10.4 → `apiVersion: "3.0"`

**Version mismatch after NetBackup upgrade**

If you've upgraded NetBackup and the exporter is still using an old API version:

1. **Remove explicit version configuration** to enable auto-detection:

   ```yaml
   nbuserver:
       # Remove or comment out apiVersion field
   ```

2. **Restart the exporter** to trigger version detection

3. **Verify detected version** in startup logs:

   ```
   INFO[0002] Detected NetBackup API version: 13.0
   ```

**Slow startup with version detection**

If automatic version detection is causing slow startup (1-3 seconds):

1. **Configure explicit version** to skip detection:

   ```yaml
   nbuserver:
       apiVersion: "13.0"  # Use your NetBackup's API version
   ```

2. **Verify faster startup** - should connect immediately without detection attempts

### Debug Mode

Enable debug logging to troubleshoot issues:

```bash
./bin/nbu_exporter --config config.yaml --debug
```

Debug mode provides:

- Detailed API request/response logging
- Masked API key display
- Verbose error context
- Collection timing information

## Migration from Previous Versions

### Multi-Version API Support (Latest)

The exporter now supports NetBackup 10.0, 10.5, and 11.0 with automatic API version detection.

**Key Features:**

- **Automatic Detection**: No manual version configuration required
- **Multi-Version Support**: Works with API versions 3.0, 12.0, and 13.0
- **Backward Compatible**: Existing configurations continue to work
- **Intelligent Fallback**: Automatically tries versions in descending order (13.0 → 12.0 → 3.0)

**Migration Options:**

**Option 1: Use Automatic Detection (Recommended)**

Remove or comment out the `apiVersion` field to enable automatic detection:

```yaml
nbuserver:
    # apiVersion: "12.0"  # Remove or comment out
    host: "master.my.domain"
    # ... other settings
```

**Option 2: Keep Explicit Version**

Your existing configuration will continue to work without changes:

```yaml
nbuserver:
    apiVersion: "12.0"  # Explicit version still supported
    host: "master.my.domain"
    # ... other settings
```

**Option 3: Update to Latest Version**

For NetBackup 11.0 environments, update to the latest API version:

```yaml
nbuserver:
    apiVersion: "13.0"  # NetBackup 11.0+
    host: "master.my.domain"
    # ... other settings
```

**No Breaking Changes**: All existing configurations remain valid. The `apiVersion` field is now optional instead of required.

See the [Migration Guide](docs/netbackup-11-migration.md) for detailed upgrade instructions.

### Breaking Change: Configuration Typo Fix (Previous Version)

The configuration field `scrappingInterval` has been corrected to `scrapingInterval`.

**Update your config.yaml:**

```yaml
# OLD (will not work)
server:
    scrappingInterval: "1h"

# NEW (required)
server:
    scrapingInterval: "1h"
```

### Configuration Options History

- `insecureSkipVerify`: Optional TLS verification control (defaults to `false`)
- `apiVersion`: NetBackup API version specification (now optional with auto-detection, previously required)

See [CHANGELOG.md](CHANGELOG.md) for complete migration details.

## Architecture

### HTTP Client

- Reusable HTTP client with connection pooling
- Context support for cancellation and timeouts
- Configurable TLS verification
- 2-minute collection timeout to prevent hanging scrapes

### Metric Collection

- Structured metric keys for type safety
- Continues collection even if one source fails
- Pagination handling for large job datasets
- Time-based filtering for efficient job queries

### Error Handling

- Comprehensive error context with wrapped errors
- Proper error propagation without `os.Exit()` in libraries
- Graceful degradation when partial data is available

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests: `go test ./...`
5. Format code: `go fmt ./...`
6. Submit a pull request

## Documentation

- [CHANGELOG.md](CHANGELOG.md) - Version history and migration notes
- [OpenTelemetry Integration Guide](docs/opentelemetry-example.md) - Complete setup guide with Docker Compose examples
- [Trace Analysis Guide](docs/trace-analysis-guide.md) - Query and analyze traces for performance optimization
- [NetBackup 11.0 Migration Guide](docs/netbackup-11-migration.md) - Upgrade instructions for NetBackup 11.0
- [API 10.5 Migration Guide](docs/api-10.5-migration.md) - Upgrade instructions for NetBackup 10.5
- [Deployment Verification](docs/deployment-verification.md) - Deployment and rollback procedures
- [REFACTORING_SUMMARY.md](docs/REFACTORING_SUMMARY.md) - Recent refactoring details
- [NetBackup API Documentation](docs/veritas-10.5/getting-started.md) - API reference

## License

See [LICENSE](LICENSE) file for details.

## Support

For issues and questions:

- GitHub Issues: <https://github.com/fjacquet/nbu_exporter/issues>
- NetBackup API Documentation: <https://sort.veritas.com/public/documents/nbu/10.5/>

## Acknowledgments

Built with:

- [Prometheus Client for Go](https://github.com/prometheus/client_golang)
- [Resty HTTP Client](https://github.com/go-resty/resty)
- [Cobra CLI Framework](https://github.com/spf13/cobra)
- [Logrus Logging](https://github.com/sirupsen/logrus)
