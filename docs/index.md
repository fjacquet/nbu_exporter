# NBU Exporter

A production-ready Prometheus exporter that collects backup job statistics and storage metrics from Veritas NetBackup REST API, exposing them for monitoring and visualization in Grafana.

![CI](https://github.com/fjacquet/nbu_exporter/actions/workflows/build.yml/badge.svg)
![Coverage](https://github.com/fjacquet/nbu_exporter/actions/workflows/coverage.yml/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/fjacquet/nbu_exporter)](https://goreportcard.com/report/github.com/fjacquet/nbu_exporter)

## Features

- **Job Metrics Collection** - Aggregates backup job statistics by type, policy, and status
- **Storage Monitoring** - Tracks storage unit capacity (free/used bytes) for disk-based storage
- **Prometheus Integration** - Native Prometheus metrics exposition via HTTP endpoint
- **OpenTelemetry Tracing** - Optional distributed tracing for performance analysis and troubleshooting
- **Multi-Version API Support** - Automatic detection of NetBackup API versions (3.0, 12.0, 13.0)
- **Health Checks** - Built-in `/health` endpoint for monitoring exporter status
- **Graceful Shutdown** - Proper signal handling with configurable shutdown timeout
- **Security** - Configurable TLS verification, API key masking in logs

## Quick Start

```bash
# Clone and build
git clone https://github.com/fjacquet/nbu_exporter.git
cd nbu_exporter
make cli

# Run with configuration
./bin/nbu_exporter --config config.yaml
```

See the [Installation guide](getting-started/installation.md) for full details.

## Version Support Matrix

| NetBackup Version | API Version | Support Status |
|-------------------|-------------|----------------|
| 11.0+             | 13.0        | Fully Supported |
| 10.5              | 12.0        | Fully Supported |
| 10.0 - 10.4       | 3.0         | Legacy Support |

The exporter automatically detects the highest supported API version (13.0 -> 12.0 -> 3.0).

## Metrics at a Glance

| Metric | Description |
|--------|-------------|
| `nbu_jobs_count` | Number of jobs by type, policy, and status |
| `nbu_jobs_size_bytes` | Total bytes transferred by jobs |
| `nbu_jobs_status_count` | Job count by action and status |
| `nbu_storage_bytes` | Storage unit capacity (free/used) |
| `nbu_api_version` | Active NetBackup API version |

See the [Metrics Reference](metrics.md) for full details.
