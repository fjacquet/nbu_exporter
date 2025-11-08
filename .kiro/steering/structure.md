# Project Structure

## Directory Layout

```
nbu_exporter/
├── internal/              # Private application code (not importable externally)
│   ├── exporter/         # Prometheus collector implementations
│   │   ├── netbackup.go  # NetBackup API client and data fetching
│   │   └── prometheus.go # Prometheus collector registration
│   ├── logging/          # Centralized logging setup
│   │   └── logging.go    # Log file initialization and configuration
│   ├── models/           # Core data structures
│   │   ├── Config.go     # Application configuration model
│   │   ├── Jobs.go       # NetBackup job data structures
│   │   ├── Storage.go    # Storage unit data structures
│   │   └── Storages.go   # Storage collection structures
│   └── utils/            # Shared utilities
│       ├── date.go       # Date/time conversion for NBU API
│       ├── file.go       # File operations and YAML parsing
│       └── pause.go      # Timing and delay utilities
├── grafana/              # Grafana dashboard definitions
│   └── NBU Statistics-*.json
├── bin/                  # Build output directory (generated)
├── log/                  # Log file directory (generated)
├── main.go               # Application entry point
├── cmd.go                # Cobra command definitions (if separate)
├── debug.go              # Debug utilities
├── config.yaml           # Configuration file (template/example)
├── go.mod                # Go module definition
├── go.sum                # Dependency checksums
├── Makefile              # Build automation
├── Dockerfile            # Container image definition
├── sonar-project.properties # SonarCloud configuration
└── README.md             # Project overview
```

## Architecture Patterns

### Collector Pattern

The exporter implements Prometheus's `Collector` interface:

```go
type Collector interface {
    Describe(chan<- *Desc)
    Collect(chan<- Metric)
}
```

**Key Components:**
1. `NbuCollector` in `internal/exporter/prometheus.go` - Implements collector interface
2. `fetchStorage()` - Retrieves storage unit metrics
3. `fetchAllJobs()` - Aggregates job statistics with pagination handling

### API Client Architecture

- **HTTP Client**: Resty-based client with TLS configuration
- **Pagination Handling**: Generic `handlePagination()` function for iterating API responses
- **URL Building**: Centralized `buildURL()` for consistent query parameter handling
- **Data Fetching**: Reusable `fetchData()` for HTTP GET requests with JSON unmarshaling

### Configuration Flow

1. **CLI Parsing**: Cobra command processes flags (`--config`, `--debug`)
2. **File Loading**: YAML configuration loaded via `utils.ReadFile()`
3. **Validation**: `checkParams()` ensures config file exists
4. **Initialization**: Logging setup, HTTP server configuration

### Metrics Collection Flow

1. **Registration**: Collector registered with Prometheus registry
2. **HTTP Endpoint**: `/metrics` endpoint exposes Prometheus format
3. **Scraping**: On each scrape request:
   - Fetch storage unit data from NetBackup API
   - Fetch job data with time-based filtering
   - Aggregate metrics by job type, policy, and status
   - Expose as Prometheus gauges

## Code Organization Principles

### Separation of Concerns

- **`main.go`**: CLI interface, HTTP server lifecycle, signal handling
- **`internal/exporter/`**: NetBackup API interaction and Prometheus metrics
- **`internal/models/`**: Data structures matching NetBackup API responses
- **`internal/utils/`**: Pure utility functions (date conversion, file I/O)
- **`internal/logging/`**: Logging configuration and setup

### Naming Conventions

- **Packages**: Short, lowercase, no underscores (e.g., `exporter`, `models`, `utils`)
- **Files**: Lowercase with underscores for multi-word names (e.g., `netbackup.go`)
- **Exported**: CamelCase starting with uppercase (e.g., `Config`, `FetchStorage`)
- **Unexported**: camelCase starting with lowercase (e.g., `createHTTPClient`, `buildURL`)

### Error Handling Patterns

- Wrap errors with context: `fmt.Errorf("context: %w", err)`
- Log errors before returning: `logging.LogError()`
- Return errors to caller for handling
- Use early returns for error conditions

## File Naming Patterns

- Main implementation: `<domain>.go` (e.g., `netbackup.go`, `prometheus.go`)
- Models: `<Entity>.go` with capitalized names (e.g., `Config.go`, `Jobs.go`)
- Utilities: `<function>.go` (e.g., `date.go`, `file.go`)

## Import Organization

Standard Go import order:
1. Standard library
2. External dependencies
3. Internal packages

Example:
```go
import (
    "crypto/tls"
    "encoding/json"
    "fmt"
    
    "github.com/go-resty/resty/v2"
    "github.com/prometheus/client_golang/prometheus"
    
    "github.com/fjacquet/nbu_exporter/internal/logging"
    "github.com/fjacquet/nbu_exporter/internal/models"
)
```

## Adding New Metrics

When adding new NetBackup metrics:

1. Define data structures in `internal/models/`
2. Implement fetch function in `internal/exporter/netbackup.go`
3. Register metric descriptors in `internal/exporter/prometheus.go`
4. Update collector's `Collect()` method to expose new metrics
5. Update Grafana dashboard JSON in `grafana/`
6. Document new metrics in README.md
