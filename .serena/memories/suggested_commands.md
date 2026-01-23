# Suggested Commands for NBU Exporter

## Build
```bash
make cli                    # Build binary to bin/nbu_exporter
go build ./...              # Build all packages
```

## Test
```bash
make test                   # Run all tests
go test ./...               # Run all tests
go test ./... -race         # Run with race detector
go test ./... -cover        # Run with coverage
go test ./internal/exporter -run TestName  # Run specific test
```

## Code Quality
```bash
make sure                   # Format, test, build, lint (full quality check)
go fmt ./...                # Format code
golangci-lint run           # Run linter
```

## Run
```bash
./bin/nbu_exporter --config config.yaml       # Run exporter
./bin/nbu_exporter -c config.yaml -d          # Run with debug mode
```

## Docker
```bash
make docker                 # Build Docker image
make run-docker             # Run container on port 2112
```

## Coverage
```bash
make test-coverage          # Generate coverage.html
```

## Clean
```bash
make clean                  # Remove build artifacts
```

## System Commands (macOS/Darwin)
- Use `ls`, `cat`, `grep`, `find` as normal
- `git` for version control
- `go` for Go toolchain commands
