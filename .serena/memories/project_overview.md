# NBU Exporter Project Overview

## Purpose
A Prometheus exporter for Veritas NetBackup (NBU). Exposes backup job statistics and storage metrics via HTTP for Prometheus scraping, with optional OpenTelemetry distributed tracing.

## Tech Stack
- **Language**: Go 1.25
- **HTTP Client**: go-resty/resty/v2
- **Metrics**: prometheus/client_golang
- **Tracing**: OpenTelemetry (otel)
- **CLI Framework**: spf13/cobra
- **Logging**: sirupsen/logrus
- **Testing**: stretchr/testify

## Architecture
- `main.go` - Entry point with Cobra CLI, HTTP server (`/metrics`, `/health`)
- `internal/exporter/` - Core Prometheus collector, HTTP client, version detection
- `internal/telemetry/` - OpenTelemetry integration
- `internal/models/` - Configuration and API response structs
- `internal/testutil/` - Shared test helpers

## Key Patterns
- Graceful degradation: Collector continues with partial metrics on errors
- Context propagation: All API calls use context for cancellation/timeout
- Version detection: Auto-detects highest supported NBU API version (13.0 → 12.0 → 3.0)
- Connection pooling: HTTP client with tuned pool settings
- Retry with backoff: 3 retries, 5-60s wait, handles 429/5xx

## Configuration
YAML config with sections: `server`, `nbuserver`, optional `opentelemetry`
