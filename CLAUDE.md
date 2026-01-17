# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A Prometheus exporter for Veritas NetBackup (NBU), written in Go 1.25. Exposes backup job statistics and storage metrics via HTTP for Prometheus scraping, with optional OpenTelemetry distributed tracing.

## Build & Development Commands

```bash
# Build the binary
make cli                    # Outputs to bin/nbu_exporter

# Run tests
make test                   # Run all tests
go test ./internal/exporter -run TestVersionDetection  # Run specific test
go test ./... -cover        # With coverage
go test -race ./...         # With race detection

# Code quality (format, test, build, lint)
make sure

# Run the exporter
./bin/nbu_exporter --config config.yaml
./bin/nbu_exporter -c config.yaml -d    # with debug mode

# Docker
make docker                 # Build image
make run-docker             # Run container on port 2112

# Clean
make clean
```

## Architecture

### Entry Point
- `main.go` - Cobra CLI, HTTP server with `/metrics` and `/health` endpoints, Prometheus registry, OpenTelemetry initialization
- `Server` struct manages lifecycle: HTTP server, Prometheus registry, telemetry manager

### Internal Packages (`internal/`)

**exporter/** - Core Prometheus collector
- `prometheus.go` - `NbuCollector` implementing `prometheus.Collector`. Metrics: `nbu_disk_bytes`, `nbu_jobs_bytes`, `nbu_jobs_count`, `nbu_status_count`, `nbu_api_version`
- `netbackup.go` - `FetchStorage()` and `FetchAllJobs()` for NBU API calls with pagination via `handlePagination()`
- `client.go` - Reusable HTTP client with connection pooling, TLS config, 2-minute timeout
- `version_detector.go` - Auto-detects API version (13.0 → 12.0 → 3.0 fallback)
- `tracing.go` - OpenTelemetry span creation helpers

**telemetry/** - OpenTelemetry integration
- `manager.go` - TracerProvider lifecycle, OTLP gRPC exporter, sampling configuration
- `attributes.go` - Span attribute constants

**models/** - Data structures
- `Config.go` - YAML config with `Validate()` method, `BuildURL()` helper
- `Jobs.go`, `Storage.go`, `Storages.go` - NBU API response structs

**testutil/** - Shared test helpers and constants

### Metrics Labels Pattern
Metrics use pipe-delimited keys split into labels:
- Storage: `name|type|size` (e.g., "pool1|AdvancedDisk|free")
- Jobs: `action|policy_type|status` (e.g., "BACKUP|Standard|0")

### Configuration
Requires `config.yaml` with sections: `server`, `nbuserver`, optional `opentelemetry`. API key obtained from NetBackup UI.

## Key Patterns

- **Graceful degradation**: Collector continues with partial metrics if storage or jobs fetch fails
- **Context propagation**: All API calls use context for cancellation/timeout
- **Version detection**: Auto-detects highest supported NBU API version at startup
- **Span hierarchy**: `prometheus.scrape` → `netbackup.fetch_storage` / `netbackup.fetch_jobs` → `netbackup.fetch_job_page`

## Key Dependencies
- `github.com/prometheus/client_golang` - Prometheus client
- `github.com/go-resty/resty/v2` - HTTP client
- `github.com/spf13/cobra` - CLI framework
- `github.com/sirupsen/logrus` - Logging
- `go.opentelemetry.io/otel` - OpenTelemetry tracing
