---
inclusion: always
---

# camt-csv Constitution

## Core Principles

### Idiomatic Go

- Follow Go Proverbs and standard conventions (`go fmt`, `goimports`)
- Use short names for small scopes, descriptive names for larger scopes
- CamelCase for exported identifiers, lowercase for unexported
- Short receiver names (1-2 characters)
- Prioritize clarity and simplicity over cleverness

### Error Handling

- Always check and propagate errors
- Wrap errors with `fmt.Errorf` and `%w` for context
- Use `errors.Is` and `errors.As` for error inspection
- Define custom error types in `internal/parsererror/` for domain-specific errors
- Return errors as the last return value
- Fail-fast for unrecoverable errors, provide graceful degradation otherwise

### Testing (NON-NEGOTIABLE)

- All significant logic requires unit tests
- Use table-driven tests for multiple scenarios
- Assertions via `github.com/stretchr/testify`
- Test files alongside source: `parser.go` → `parser_test.go`
- Use `testdata/` subdirectories for fixtures
- Mock external dependencies (filesystem, APIs)
- Minimum 80% coverage overall, 100% for critical paths (parsing, validation)
- All tests must pass before changes are complete
- Integration tests for CLI commands with filesystem/external service interactions

### Concurrency

- Use `sync.WaitGroup` and channels for goroutine coordination
- Use `context.Context` for cancellation and request-scoped values
- Avoid race conditions and deadlocks
- Run `go test -race` to detect data races

### CLI Architecture

- Use `spf13/cobra` for command structure, `spf13/viper` for configuration
- Organize commands as subcommands in `cmd/` directory
- Flags: kebab-case, clear descriptions, explicit defaults, mark required flags
- Standard streams: informational → `Stdout`, errors/warnings → `Stderr`
- Exit codes: 0 for success, non-zero for failure
- Configuration precedence: CLI flags > Env vars > Config file > Defaults

### Parser Interface

- All parsers implement `parser.Parser` interface from `internal/parser/parser.go`
- Interface signature: `Parse(r io.Reader) ([]models.Transaction, error)`
- Each parser in its own package: `internal/<format>parser/`
- Create `adapter.go` with constructor for each parser
- Register new parsers in factory pattern if applicable

### Single Responsibility

- Each package/function/type has one well-defined responsibility
- Parsers: format-specific parsing logic
- Models: data structure definitions
- Categorizer: transaction categorization
- Store: YAML configuration and category storage
- Common: shared utilities (CSV I/O, date/currency utils)

### Financial Data Handling

- Use `github.com/shopspring/decimal` for all monetary amounts (never float64)
- Design core models as immutable where possible
- Create new instances rather than modifying in-place
- Validate all financial data at boundaries

### Hybrid Categorization System

Three-tier approach (in order):

1. **Direct Mapping**: Exact matches from `database/creditors.yaml` and `database/debtors.yaml`
2. **Keyword Matching**: Local rules from `database/categories.yaml`
3. **AI Fallback**: Google Gemini API (optional, with auto-learning)

This prioritizes performance, privacy, and cost control. Use `github.com/gocarina/gocsv` for CSV processing.

### Configuration Management

- Use `spf13/viper` for hierarchical configuration
- Precedence: CLI flags > Env vars (`CAMT_` prefix) > Config file > Defaults
- Load env vars via `github.com/joho/godotenv`
- Config file locations: `~/.camt-csv/config.yaml` or `./.camt-csv/config.yaml`
- Never hardcode or log sensitive data (API keys, credentials)
- Validate configuration at startup

## Documentation & Logging

### Documentation

- All exported types, functions, and methods require godoc comments
- Explain _why_ something is done, not just _what_
- Document non-obvious behavior, edge cases, and assumptions
- Keep comments up-to-date with code changes

### Logging

- Use `github.com/sirupsen/logrus` for structured logging
- Levels: DEBUG (development), INFO (operations), WARN (degraded), ERROR (failures), FATAL (unrecoverable)
- Format: JSON for machine parsing, plain text for human readability
- Include contextual information (transaction IDs, file names, operation types)
- Never log sensitive data (API keys, full account numbers, personal information)

## Code Quality

### Linting

- Enforce `golangci-lint` via pre-commit hooks and CI/CD
- Configuration in `.golangci.yml`
- All code must pass linting before merge

### Dependencies

- Use Go Modules exclusively
- Pin versions in `go.mod` for reproducible builds
- Run `go mod tidy` to clean unused dependencies
- Document rationale for major dependencies

## Design Patterns

### Patterns to Use

- **Strategy Pattern**: Different parser implementations
- **Adapter Pattern**: Interface bridging (e.g., `adapter.go` files)
- **Factory Pattern**: Object creation (parser factory)
- **Template Method**: Common logic with customizations

### Anti-Patterns to Avoid

- God Objects (large, do-everything types)
- Tight Coupling (prefer interfaces)
- Magic Numbers/Strings (use named constants)
- Premature Optimization (profile first)

## Performance

### Optimization Approach

- Profile before optimizing (`go test -bench`, `pprof`)
- Focus on: minimizing allocations, streaming large files, efficient data structures
- Use benchmarks for critical paths
- Properly manage resources (file handles, memory) with `defer` for cleanup

### Resource Management

- Close file handles promptly
- Use `defer` for cleanup operations
- Stream large files rather than loading entirely into memory
- Consider memory usage for batch operations

## Security

### Input Validation

- Validate all user input (flags, arguments, config values)
- Sanitize file paths to prevent directory traversal
- Validate data formats before processing

### Sensitive Data

- Never hardcode secrets, API keys, or credentials
- Use environment variables for sensitive configuration
- Set appropriate file permissions (0600 for config, 0755 for binaries)
- Use HTTPS only for external API calls

### Vulnerability Management

- Run security scanners (`gosec`, `govulncheck`)
- Keep dependencies updated
- Monitor security advisories

## Adding New Parsers

When adding support for a new financial format:

1. Create package: `internal/<format>parser/`
2. Implement `parser.Parser` interface
3. Create `adapter.go` with constructor
4. Add tests in `<format>parser_test.go` with sample data in `testdata/`
5. Create CLI command in `cmd/<format>/`
6. Update documentation in `README.md` and `docs/user-guide.md`
7. Add sample files to `samples/<format>/`

## Release Process

### Versioning

- Follow Semantic Versioning (MAJOR.MINOR.PATCH)
- Tag releases in git: `v1.2.3`
- Update version in code and documentation

### Build & Deploy

- Automate via CI/CD (GitHub Actions)
- Cross-platform builds: Linux, macOS, Windows (amd64, arm64)
- Build flags: `-ldflags="-s -w"` for production
- Test all platforms before release

## Governance

- This constitution supersedes other project practices
- All code changes must comply with these principles
- Complexity requires justification
- See `docs/coding-standards.md` for additional runtime guidance

**Version**: 1.3.0 | **Last Updated**: 2025-11-01
