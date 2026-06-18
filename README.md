# NBU Exporter — Prometheus Exporter for Veritas NetBackup

A Prometheus exporter that collects backup job statistics, storage metrics, tape health, and per-client lifecycle data from the Veritas NetBackup REST API.

[![CI](https://github.com/fjacquet/nbu_exporter/actions/workflows/ci.yml/badge.svg)](https://github.com/fjacquet/nbu_exporter/actions/workflows/ci.yml)
[![CodeQL](https://github.com/fjacquet/nbu_exporter/actions/workflows/codeql.yml/badge.svg)](https://github.com/fjacquet/nbu_exporter/actions/workflows/codeql.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/fjacquet/nbu_exporter)](https://goreportcard.com/report/github.com/fjacquet/nbu_exporter)
![Go Version](https://img.shields.io/github/go-mod/go-version/fjacquet/nbu_exporter)
[![Latest Release](https://img.shields.io/github/v/release/fjacquet/nbu_exporter)](https://github.com/fjacquet/nbu_exporter/releases/latest)
[![License: GPL](https://img.shields.io/github/license/fjacquet/nbu_exporter)](LICENSE)
[![Documentation](https://img.shields.io/badge/docs-github%20pages-blue)](https://fjacquet.github.io/nbu_exporter/)

## Features

- **Job metrics** — count, volume, duration, dedup ratio, queued jobs; by action, policy type, and status
- **Storage metrics** — free/used bytes per storage unit, capacity, concurrent job limits
- **Tape & disk-pool metrics** — drive status per robot, media inventory per pool, disk pool volume states (NBU 10.5+, opt-in)
- **Per-client lifecycle** — last successful job timestamp and job counts per client, for BACKUP / DUPLICATION / IMPORT phases (opt-in, allowlist or all clients)
- **Multi-site** — scrape several NetBackup primaries from one exporter, every metric labelled by `site`
- **Automatic API version detection** — probes 14.0 → 13.0 → 12.0 → 10.0 at startup; or pin explicitly
- **7 Grafana dashboards** — Overview, Jobs, Storage, Data Protection, Tape & Disks, Lifecycle, Multi-site
- **Prometheus alerting rules** — per-client compliance, tape health, inter-site divergence
- **Hot config reload** via SIGHUP / fsnotify file watcher
- **Optional OpenTelemetry** distributed tracing (OTLP gRPC)
- **Health check** endpoint at `/health`

## Full observability stack (recommended)

Deploy the exporter, Prometheus, and Grafana with one command:

```bash
git clone https://github.com/fjacquet/nbu_exporter.git
cd nbu_exporter

# Fill in your NBU master hostname and API key
cp .env.example .env
# Edit config.yaml with your nbuserver details (see Configuration below)

docker compose up -d
```

Then open:
- **Grafana** — `http://localhost:3000` (admin / admin) → Dashboards → NetBackup
- **Prometheus** — `http://localhost:9090`
- **Metrics** — `http://localhost:9440/metrics`

See [docs/deploy-stack.md](docs/deploy-stack.md) for the full step-by-step guide (API key generation, multi-site setup, alerting, production hardening).

## Configuration

```yaml
# config.yaml — minimal single-site example
server:
    host: "0.0.0.0"
    port: "9440"
    uri: "/metrics"
    scrapingInterval: "1h"
    logName: "log/nbu-exporter.log"

nbuserver:
    scheme: "https"
    uri: "/netbackup"
    host: "nbu-master.my.domain"   # NBU primary server FQDN
    port: "1556"
    apiKey: "your-api-key-here"    # generated from NBU Admin Console → API Keys
    insecureSkipVerify: false

collectors:
    tape:
        enabled: true    # NBU 10.5+ required; set false if older
    perClient:
        enabled: false   # opt-in; set true to track per-client compliance
```

For multi-site, replace `nbuserver:` with an `nbuservers:` list — see [`docs/config-examples/config-multisite.yaml`](docs/config-examples/config-multisite.yaml).

### Environment Variables

| Variable | Description |
|---|---|
| `NBU1_HOSTNAME` | NetBackup master hostname (single-site quickstart) |
| `NBU1_APIKEY` | NetBackup API key |

`config.yaml` supports `${VAR}` interpolation for secrets — never commit API keys.

## Dashboards

| Dashboard | What it shows |
|---|---|
| **Overview** | Site health tiles, backup success rate, storage usage, alert summary |
| **Jobs** | Success rate, volume, duration p50/p95, dedup ratio, queued jobs |
| **Storage** | Capacity gauges per storage unit |
| **Data Protection** | Alerts, malware scans, catalog SLOs |
| **Tape & Disks** | Drive status by robot, media inventory by pool, disk pool volumes |
| **Lifecycle** | Per-client compliance timeline, BACKUP/DUPLICATION/IMPORT success rates, overdue clients |
| **Multi-site** | Cross-site volume, replication rate, divergence ratio, per-site comparison |

All dashboards have a `$site` variable to filter by site or view all at once.

## Binary / CLI usage

```bash
# Build
make cli                    # outputs bin/nbu_exporter

# Run
./bin/nbu_exporter --config config.yaml
./bin/nbu_exporter --config config.yaml --debug   # verbose logging
./bin/nbu_exporter --config config.yaml --trace   # log every API response body

# macOS (Homebrew)
brew install fjacquet/tap/nbu_exporter
```

## Development

```bash
make tools         # install golangci-lint, govulncheck, cyclonedx-gomod (one-time)
make sure          # fmt + vet + test + build + lint
make ci            # full CI gate (fmt-check, vet, lint, test-race, govulncheck, coverage)
make test-coverage # tests with HTML coverage report
make docker        # build Docker image

# Regenerate Grafana dashboard JSONs after editing grafana/gen/*.py
PYTHONPATH=. python3 grafana/build_dashboards.py
```

## Documentation

Full docs at **[fjacquet.github.io/nbu_exporter](https://fjacquet.github.io/nbu_exporter/)**.

| Topic | Link |
|---|---|
| **Full stack deployment** | [docs/deploy-stack.md](docs/deploy-stack.md) |
| Installation | [Getting Started](https://fjacquet.github.io/nbu_exporter/getting-started/installation/) |
| Configuration | [Configuration Guide](https://fjacquet.github.io/nbu_exporter/getting-started/configuration/) |
| Metrics reference | [Metrics](https://fjacquet.github.io/nbu_exporter/metrics/) |
| Docker | [Docker](https://fjacquet.github.io/nbu_exporter/docker/) |
| OpenTelemetry | [Tracing Setup](https://fjacquet.github.io/nbu_exporter/opentelemetry-example/) |
| Changelog | [Changelog](https://fjacquet.github.io/nbu_exporter/changelog/) |

## Contributing

1. Fork the repository
2. Create a feature branch
3. Run `make ci` before submitting
4. Submit a pull request

## License

See [LICENSE](LICENSE) file for details.
