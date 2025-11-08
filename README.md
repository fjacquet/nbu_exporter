# NBU Exporter - Prometheus Exporter for Veritas NetBackup

A production-ready Prometheus exporter that collects backup job statistics and storage metrics from Veritas NetBackup REST API, exposing them for monitoring and visualization in Grafana.

![Code Analysis](https://github.com/fjacquet/nbu_exporter/actions/workflows/codeql-analysis.yml/badge.svg)
[![Code Smells](https://sonarcloud.io/api/project_badges/measure?project=fjacquet_nbu_exporter&metric=code_smells)](https://sonarcloud.io/summary/new_code?id=fjacquet_nbu_exporter)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=fjacquet_nbu_exporter&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=fjacquet_nbu_exporter)
![Go build](https://github.com/fjacquet/nbu_exporter/actions/workflows/go.yml/badge.svg)

## Features

- **Job Metrics Collection**: Aggregates backup job statistics by type, policy, and status
- **Storage Monitoring**: Tracks storage unit capacity (free/used bytes) for disk-based storage
- **Prometheus Integration**: Native Prometheus metrics exposition via HTTP endpoint
- **Configurable Scraping**: Adjustable time windows for historical job data collection
- **Health Checks**: Built-in `/health` endpoint for monitoring exporter status
- **Graceful Shutdown**: Proper signal handling with configurable shutdown timeout
- **Security**: Configurable TLS verification, API key masking in logs
- **Performance**: HTTP client connection pooling, context-aware operations

## Quick Start

### Prerequisites

- Go 1.23.4 or later
- Veritas NetBackup 10.5 or later (API version 12.0+)
- Access to NetBackup REST API
- NetBackup API key (generated from NBU UI)

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
    apiVersion: "12.0"  # NetBackup API version (12.0 for NetBackup 10.5+)
    apiKey: "your-api-key-here"
    contentType: "application/vnd.netbackup+json; version=3.0"
    insecureSkipVerify: false  # Set to true only for testing environments
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

**Note**: Tape storage units are excluded from metrics collection.

## Grafana Dashboard

A pre-built Grafana dashboard is included in the `grafana/` directory:

1. Import `grafana/NBU Statistics-1629904585394.json` into your Grafana instance
2. Configure the Prometheus datasource
3. View NetBackup job statistics and storage utilization

## Docker Deployment

### Build Docker Image

```bash
make docker

# Or manually
docker build -t nbu_exporter .
```

### Run Container

```bash
# Using Makefile
make run-docker

# Or manually
docker run -d \
  -p 8080:8080 \
  -v $(pwd)/config.yaml:/config.yaml \
  nbu_exporter --config /config.yaml
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

```bash
# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with race detection
go test -race ./...
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
| `apiVersion` | string | Yes | "12.0" | NetBackup API version (12.0 for NetBackup 10.5+) |
| `apiKey` | string | Yes | - | NetBackup API key |
| `contentType` | string | Yes | - | API content type header |
| `insecureSkipVerify` | bool | No | false | Skip TLS certificate verification (not recommended for production) |

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

### Breaking Change: Configuration Typo Fix

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

### New Configuration Options

- `insecureSkipVerify`: Optional TLS verification control (defaults to `false`)
- `apiVersion`: NetBackup API version specification (required for NetBackup 10.5+, defaults to "12.0")

### NetBackup 10.5 API Support

The exporter now supports NetBackup 10.5 with API version 12.0. Key changes:

- **Required Configuration**: Add `apiVersion: "12.0"` to the `nbuserver` section
- **Version Header**: The exporter includes the API version in the Accept header for all requests
- **Backward Compatibility**: Existing configurations will default to version "12.0"
- **Minimum Version**: NetBackup 10.5 or later is required

**Migration Steps:**

1. Ensure your NetBackup server is version 10.5 or later
2. Add `apiVersion: "12.0"` to your `config.yaml` under the `nbuserver` section
3. Restart the exporter
4. Verify metrics collection is working correctly

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
