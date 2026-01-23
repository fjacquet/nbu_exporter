# Technology Stack

**Analysis Date:** 2026-01-22

## Languages

**Primary:**

- Go 1.25 - All application code and CLI tooling

**Secondary:**

- YAML - Configuration and Docker Compose definitions
- Shell - Build and deployment scripts (Makefile)

## Runtime

**Environment:**

- Go 1.25 runtime with standard library
- Runs as standalone compiled binary (no JVM, runtime dependencies, or interpreters required)

**Package Manager:**

- Go modules (go.mod)
- Lockfile: `go.sum` (present)
- Vendoring: `vendor/` directory populated with all dependencies

## Frameworks

**Core:**

- `github.com/prometheus/client_golang` v1.23.2 - Prometheus metrics client and HTTP exposition
- `github.com/go-resty/resty/v2` v2.17.1 - HTTP client with TLS and timeout management
- `github.com/spf13/cobra` v1.10.2 - CLI framework for command parsing

**Tracing & Observability:**

- `go.opentelemetry.io/otel` v1.39.0 - OpenTelemetry tracing SDK
- `go.opentelemetry.io/otel/sdk` v1.39.0 - OpenTelemetry SDK implementation
- `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc` v1.39.0 - OTLP gRPC exporter for traces
- `google.golang.org/grpc` v1.78.0 - gRPC protocol for trace export

**Logging:**

- `github.com/sirupsen/logrus` v1.9.4 - Structured logging with multiple output levels

**Configuration:**

- `gopkg.in/yaml.v2` v2.4.0 - YAML parsing for config files

**Testing:**

- `github.com/stretchr/testify` v1.11.1 - Assertion library and mock utilities

## Key Dependencies

**Critical:**

- `github.com/prometheus/client_golang` v1.23.2 - Core metrics collection and exposition for Prometheus scraping at `/metrics` endpoint
- `github.com/go-resty/resty/v2` v2.17.1 - Handles all NetBackup API communication with connection pooling and 2-minute timeout
- `go.opentelemetry.io/otel` v1.39.0 - Optional distributed tracing for debugging slow API calls and scrape cycles

**Infrastructure:**

- `github.com/spf13/cobra` v1.10.2 - CLI argument parsing and help text generation
- `github.com/sirupsen/logrus` v1.9.4 - Structured logging to file and console
- `gopkg.in/yaml.v2` v2.4.0 - Configuration file parsing
- `google.golang.org/grpc` v1.78.0 - gRPC protocol for OpenTelemetry trace export to collectors

**Transitive Dependencies:**

- `github.com/prometheus/common` v0.67.5 - Prometheus data types
- `github.com/prometheus/procfs` v0.19.2 - Linux /proc filesystem utilities
- `golang.org/x/net` v0.49.0 - HTTP/2 and TLS support
- `google.golang.org/protobuf` v1.36.11 - Protocol buffers for gRPC messages
- `github.com/cenkalti/backoff/v5` v5.0.3 - Retry logic for telemetry exporter

## Configuration

**Environment:**

- Configuration via YAML file specified at runtime: `--config config.yaml` flag
- No environment variable support; all settings via config file
- API key obtained from NetBackup UI, provided in `nbuserver.apiKey` field
- Optional OpenTelemetry configuration for distributed tracing

**Required Configuration Sections:**

- `server` - HTTP server binding (host, port, metrics URI `/metrics`, scraping interval, log file path)
- `nbuserver` - NetBackup master details (host, port, scheme, API key, domain, API version)
- `opentelemetry` - Optional (disabled by default)

**Build:**

- Makefile targets: `make cli` (build binary), `make docker` (build image), `make test`, `make sure` (full QA)
- Binary output: `bin/nbu_exporter`
- Docker multi-stage build: Golang builder stage → Alpine runtime stage
- Dockerfile includes `ca-certificates` for HTTPS to NetBackup servers

## Platform Requirements

**Development:**

- Go 1.25
- Make or compatible build tool
- Docker and Docker Compose (for containerized testing)
- golangci-lint (optional, for static analysis via `make sure`)

**Production:**

- Linux (Alpine-based Docker image), macOS, or Windows
- No runtime dependencies beyond standard Go library and ca-certificates
- Binary size: ~15MB (stripped with `-ldflags="-s -w"` for release builds)
- Memory: Minimal; scales with number of storage units and jobs in NetBackup (typical: 50-100MB)
- Network: HTTPS connectivity to NetBackup master server (port 1556 by default)
- Port binding: Configurable (default 2112 for `/metrics` endpoint)

**Docker Runtime:**

- Base image: Alpine Linux (minimal ~5MB)
- ca-certificates package for TLS certificate verification
- Health check endpoint: `GET /health` returns HTTP 200 OK

---

_Stack analysis: 2026-01-22_
