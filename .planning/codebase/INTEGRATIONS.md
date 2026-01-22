# External Integrations

**Analysis Date:** 2026-01-22

## APIs & External Services

**NetBackup REST API:**
- Service: Veritas NetBackup backup infrastructure monitoring
- What it's used for: Fetching storage capacity metrics and job statistics for Prometheus exposition
- SDK/Client: Custom HTTP client via `github.com/go-resty/resty/v2`
- Auth: API key via `Authorization` header
- Endpoints:
  - `GET /netbackup/admin/jobs` - Job statistics with pagination
  - `GET /netbackup/admin/storages` - Storage unit capacity and usage
- Supported API Versions: 3.0 (NetBackup 10.0-10.4), 12.0 (NetBackup 10.5), 13.0 (NetBackup 11.0+)
- Auto-detection: Version detection at startup (`GET /netbackup/admin/jobs?page[limit]=1` with fallback chain)
- Configuration:
  - Host: `nbuserver.host` (e.g., "master.my.domain")
  - Port: `nbuserver.port` (default: 1556)
  - Scheme: `nbuserver.scheme` (http or https)
  - API Key: `nbuserver.apiKey` (required, no default)
  - TLS: Configurable verification skip via `nbuserver.insecureSkipVerify`
  - Timeout: 2-minute connection timeout, 1-minute request timeout

## Data Storage

**Databases:**
- None - exporter is stateless, no persistent storage

**File Storage:**
- Local log file: Path configured in `server.logName` (e.g., `log/nbu-exporter.log`)
- Structured logging via `github.com/sirupsen/logrus`

**Caching:**
- None - metrics fetched on-demand for each Prometheus scrape cycle
- No in-memory cache between scrapes

## Authentication & Identity

**Auth Provider:**
- Custom - API key authentication only
- Implementation: Bearer token in `Authorization` HTTP header
- Configuration: `nbuserver.apiKey` loaded from config file (no env var support)
- Lifecycle: Static token obtained manually from NetBackup UI, requires manual rotation

## Monitoring & Observability

**Error Tracking:**
- None - errors logged to file via logrus but not sent to external service

**Logs:**
- Approach: File-based structured logging (JSON and text formats supported)
- Output: `server.logName` file path from config
- Levels: INFO, DEBUG (when `--debug` CLI flag used), WARN, ERROR
- Persistence: Logs rotated manually by external tools (logrotate, Docker volume management)

**Distributed Tracing (Optional):**
- Framework: OpenTelemetry (disabled by default)
- Export: OTLP gRPC protocol
- Endpoint: `opentelemetry.endpoint` (e.g., "localhost:4317" or "otel-collector.example.com:4317")
- TLS: `opentelemetry.insecure` flag controls connection mode
- Sampling: Configurable via `opentelemetry.samplingRate` (0.0 to 1.0)
- Traces exported: Prometheus scrape cycle spans, NetBackup API call spans with HTTP semantics
- Integration: Jaeger, Tempo, Datadog, or any OTLP-compatible backend
- Connection: Non-blocking; failures don't block metric collection

## CI/CD & Deployment

**Hosting:**
- Flexible: Runs on any platform (Linux, macOS, Windows)
- Docker: Multi-stage build produces Alpine-based image `nbu_exporter:latest`
- Kubernetes: Compatible via Helm or manual manifests (examples in `docs/veritas-*/`)

**CI Pipeline:**
- GitHub Actions workflows in `.github/workflows/`
  - `build.yml` - Go build and test
  - `coverage.yml` - Test coverage reporting
  - `static.yml` - Linting and static analysis
  - `release.yml` - Release binary builds
- Test coverage: Configured thresholds in `.testcoverage.yml`

**Container Registry:**
- No push to external registry in base Makefile (manual docker push required)
- Dockerfile uses latest golang and alpine base images (no pinned versions)

## Environment Configuration

**Required env vars:**
- None - all configuration via YAML file

**Config file sections:**
```yaml
server:
  host: "0.0.0.0"          # Binding address
  port: "2112"             # HTTP port
  uri: "/metrics"          # Prometheus metrics endpoint
  scrapingInterval: "1h"   # Data collection window for job statistics
  logName: "log/nbu-exporter.log"  # Log file path

nbuserver:
  host: "master.my.domain" # NetBackup master hostname
  port: "1556"             # NetBackup REST API port
  scheme: "https"          # http or https
  uri: "/netbackup"        # API base path
  domain: "my.domain"      # Windows domain (for auth context)
  domainType: "NT"         # Domain type (NT = Windows, UNIX = Unix)
  apiKey: "secret-key"     # Required: obtained from NetBackup UI
  apiVersion: "13.0"       # Optional: auto-detected if omitted
  insecureSkipVerify: false # Skip TLS cert verification (dev only)

opentelemetry:
  enabled: false           # Enable distributed tracing
  endpoint: "localhost:4317"  # OTLP gRPC collector endpoint
  insecure: true           # Insecure connection (no TLS)
  samplingRate: 0.1        # Sample 10% of traces (0.0-1.0)
```

**Secrets location:**
- API key: Embedded in `config.yaml` (requires file-level access control)
- No environment variable support for secrets
- Recommendation: Use Docker secrets or Kubernetes secrets to mount config file

## Webhooks & Callbacks

**Incoming:**
- `/health` - Health check endpoint (returns HTTP 200 OK)
- `/metrics` - Prometheus metrics endpoint (configurable via `server.uri`)

**Outgoing:**
- None - exporter is passive (pull-based via Prometheus scrapes)

## Metrics Exposition

**Prometheus Integration:**
- Endpoint: `http://{host}:{port}{uri}` (default: `http://localhost:2112/metrics`)
- Format: Prometheus text-based exposition format
- Handler: `github.com/prometheus/client_golang/prometheus/promhttp`
- Metrics exposed:
  - `nbu_disk_bytes` - Storage capacity in bytes (labels: name, type, size)
  - `nbu_jobs_bytes` - Job data transferred in bytes (labels: action, policy_type, status)
  - `nbu_jobs_count` - Number of jobs (labels: action, policy_type, status)
  - `nbu_status_count` - Job status counts (labels: action, status)
  - `nbu_api_version` - NetBackup API version in use
  - `nbu_response_time_ms` - NetBackup API response time (if enabled)
- Scrape interval: Configurable in Prometheus config (typical: 5m)
- Timeout: Default 2 minutes for metric collection

---

*Integration audit: 2026-01-22*
