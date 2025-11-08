# Technical Stack

## Language & Runtime

- **Go**: Version 1.23.4
- **Build System**: Go modules with Makefile automation

## Core Dependencies

### HTTP & API Client

- `go-resty/resty/v2`: HTTP client for NetBackup API interactions
  - Context7 Library: `/go-resty/docs` (209 snippets, Trust Score: 8.1)
  - Use for: REST API calls, TLS configuration, request/response handling
- Built-in `crypto/tls`: TLS configuration for HTTPS connections

### Metrics & Monitoring

- `prometheus/client_golang`: Prometheus client library for metrics exposition
  - Context7 Library: `/prometheus/client_golang` (8 snippets, Trust Score: 7.4)
  - Use for: Collector interface, gauge/counter metrics, HTTP handler
- Custom collector implementation for NetBackup-specific metrics

### CLI & Configuration

- `spf13/cobra`: Command-line interface framework
  - Context7 Library: `/spf13/cobra` (108 snippets, Version: v1.9.1)
  - Use for: Command structure, flags, subcommands
- `gopkg.in/yaml.v2`: YAML configuration file parsing

### Logging

- `sirupsen/logrus`: Structured logging
  - Context7 Library: `/sirupsen/logrus` (24 snippets, Trust Score: 10, Version: v1.9.3)
  - Use for: Log levels, structured fields, formatters

### NetBackup API

- **Veritas NetBackup REST API**: Proprietary API (no Context7 documentation)
  - Refer to official Veritas documentation
  - See `internal/exporter/netbackup.go` for established patterns
  - API endpoints: `/storage/storage-units`, `/admin/jobs`

## Common Commands

### Development

```bash
# Install dependencies
go mod download
go mod tidy

# Run tests
go test
go test -v                    # Verbose output
go test -race                 # With race detection

# Format code
go fmt ./...
goimports -w .
```

### Building

```bash
# Build CLI binary
make cli
# Output: bin/nbu_exporter

# Build Docker image
make docker

# Build all (CLI + tests + Docker)
make all

# Clean build artifacts
make clean
```

### Running

```bash
# Run locally with config file
./bin/nbu_exporter --config config.yaml

# Run with debug mode
./bin/nbu_exporter --config config.yaml --debug

# Run via Makefile
make run-cli

# Run Docker container
make run-docker
# Exposes metrics on http://localhost:8080/metrics
```

### Configuration

```bash
# Configuration file: config.yaml (required)
# Must specify:
# - Server settings (host, port, metrics URI, scraping interval)
# - NetBackup server details (host, port, API key)

# Example:
./nbu_exporter --config config.yaml
./nbu_exporter -c config.yaml --debug
```

## Docker Deployment

```bash
# Build image
docker build -t nbu_exporter .

# Run container
docker run -d -p 8080:8080 \
  -v $(pwd)/config.yaml:/config.yaml \
  nbu_exporter --config /config.yaml
```

## Project Structure Conventions

- Use Go modules exclusively for dependency management
- Pin dependency versions in go.mod for reproducible builds
- All code must pass `go fmt` checks before commit
- Follow standard Go project layout conventions
