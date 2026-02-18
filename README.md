# NBU Exporter - Prometheus Exporter for Veritas NetBackup

A Prometheus exporter that collects backup job statistics and storage metrics from Veritas NetBackup REST API.

![CI](https://github.com/fjacquet/nbu_exporter/actions/workflows/build.yml/badge.svg)
![Coverage](https://github.com/fjacquet/nbu_exporter/actions/workflows/coverage.yml/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/fjacquet/nbu_exporter)](https://goreportcard.com/report/github.com/fjacquet/nbu_exporter)
![Go Version](https://img.shields.io/github/go-mod/go-version/fjacquet/nbu_exporter)
[![Latest Release](https://img.shields.io/github/v/release/fjacquet/nbu_exporter)](https://github.com/fjacquet/nbu_exporter/releases/latest)
[![License: GPL](https://img.shields.io/github/license/fjacquet/nbu_exporter)](LICENSE)
[![Documentation](https://img.shields.io/badge/docs-github%20pages-blue)](https://fjacquet.github.io/nbu_exporter/)

## Features

- Job metrics collection by type, policy, and status
- Storage unit capacity monitoring (free/used bytes)
- Automatic NetBackup API version detection (13.0, 12.0, 3.0)
- Optional OpenTelemetry distributed tracing
- Hot config reload via SIGHUP / file watcher
- Health check endpoint
- Docker support

## Quick Start

```bash
# Build
make cli

# Configure
cp config.yaml.example config.yaml  # edit with your NBU server details

# Run
./bin/nbu_exporter --config config.yaml
```

Metrics are exposed at `http://localhost:2112/metrics`.

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
