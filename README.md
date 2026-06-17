# NBU Exporter - Prometheus Exporter for Veritas NetBackup

A Prometheus exporter that collects backup job statistics and storage metrics from Veritas NetBackup REST API.

[![CI](https://github.com/fjacquet/nbu_exporter/actions/workflows/ci.yml/badge.svg)](https://github.com/fjacquet/nbu_exporter/actions/workflows/ci.yml)
[![CodeQL](https://github.com/fjacquet/nbu_exporter/actions/workflows/codeql.yml/badge.svg)](https://github.com/fjacquet/nbu_exporter/actions/workflows/codeql.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/fjacquet/nbu_exporter)](https://goreportcard.com/report/github.com/fjacquet/nbu_exporter)
![Go Version](https://img.shields.io/github/go-mod/go-version/fjacquet/nbu_exporter)
[![Latest Release](https://img.shields.io/github/v/release/fjacquet/nbu_exporter)](https://github.com/fjacquet/nbu_exporter/releases/latest)
[![License: GPL](https://img.shields.io/github/license/fjacquet/nbu_exporter)](LICENSE)
[![Documentation](https://img.shields.io/badge/docs-github%20pages-blue)](https://fjacquet.github.io/nbu_exporter/)

## Features

- Job metrics collection by type, policy, and status
- Storage unit capacity monitoring (free/used bytes)
- Multi-site: scrape several NetBackup primaries from one exporter, labelled by `site`
  (background snapshot collection loop; see [`config-multisite.yaml`](docs/config-examples/config-multisite.yaml))
- Automatic NetBackup API version detection (14.0, 13.0, 12.0, 10.0)
- Optional OpenTelemetry distributed tracing
- Hot config reload via SIGHUP / file watcher
- Health check endpoint
- Docker support

## Quick Start

On macOS, install via Homebrew:

```bash
brew install fjacquet/tap/nbu_exporter
```

Or build from source (macOS/Linux):

```bash
make cli
```

Then configure and run:

```bash
# Configure — supply credentials via environment variables (never commit secrets)
cp .env.example .env        # fill in NBU1_HOSTNAME and NBU1_APIKEY
cp config.yaml config.local.yaml  # optional: edit server settings

# Run
NBU1_HOSTNAME=nbu.example.com NBU1_APIKEY=mykey nbu_exporter --config config.yaml
# or with a .env file sourced into your shell, simply:
nbu_exporter --config config.yaml
```

`host` and `apiKey` in `config.yaml` support `${VAR}` interpolation — see
[Configuration Guide](https://fjacquet.github.io/nbu_exporter/getting-started/configuration/) for details.

Metrics are exposed at `http://localhost:9440/metrics`.

### Environment Variables / .env

| Variable | Description | Default in compose |
|----------|-------------|-------------------|
| `NBU1_HOSTNAME` | NetBackup master server hostname | `master.my.domain` |
| `NBU1_APIKEY` | NetBackup API key | _(empty)_ |

`NBU1_*` is a single-server quickstart convenience — `config.yaml` is always the source of truth.
For multiple servers add more entries directly in `config.yaml` (literal values or additional env refs).

## Documentation

Full documentation is available at **[fjacquet.github.io/nbu_exporter](https://fjacquet.github.io/nbu_exporter/)**.

| Topic | Link |
|-------|------|
| Installation | [Getting Started](https://fjacquet.github.io/nbu_exporter/getting-started/installation/) |
| Configuration | [Configuration Guide](https://fjacquet.github.io/nbu_exporter/getting-started/configuration/) |
| Metrics Reference | [Metrics](https://fjacquet.github.io/nbu_exporter/metrics/) |
| Docker Deployment | [Docker](https://fjacquet.github.io/nbu_exporter/docker/) |
| OpenTelemetry | [Tracing Setup](https://fjacquet.github.io/nbu_exporter/opentelemetry-example/) |
| Migration Guides | [NetBackup 11.0](https://fjacquet.github.io/nbu_exporter/netbackup-11-migration/) |
| Changelog | [Changelog](https://fjacquet.github.io/nbu_exporter/changelog/) |

## Development

```bash
make sure          # fmt, test, build, lint
make test          # run tests
make test-coverage # tests with HTML coverage report
make docker        # build Docker image
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Run `make sure` before submitting
4. Submit a pull request

## License

See [LICENSE](LICENSE) file for details.
