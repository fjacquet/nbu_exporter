---
inclusion: fileMatch
fileMatchPattern: '*.go'
---

# Go Best Practices for camt-csv

## Code Formatting and Style

- All Go code must be formatted with `gofmt` and organized with `goimports`
- Follow idiomatic Go naming: CamelCase for exported, camelCase for unexported
- Use short receiver names (1-2 characters) and meaningful variable names
- All code must pass `golangci-lint` checks defined in `.golangci.yml`

## Project Structure Conventions

- **cmd/**: CLI command implementations using cobra pattern
- **internal/**: Private application code organized by domain
  - `models/`: Core data structures (Transaction, Money, etc.)
  - `parser/`: Parser interface and implementations
  - `categorizer/`: Transaction categorization logic
  - `store/`: YAML configuration management
- Place new parsers in `internal/<format>parser/` packages
- Each parser must implement the `parser.Parser` interface
- Create `adapter.go` files for parser constructors

## Interface Design

- All parsers implement: `Parse(r io.Reader) ([]models.Transaction, error)`
- Use interfaces for testability and extensibility
- Keep interfaces small and focused (single responsibility)
- Define interfaces in consuming packages, not implementing packages

## Error Handling

- Always check and propagate errors explicitly
- Wrap errors with context using `fmt.Errorf("context: %w", err)`
- Use custom error types in `internal/parsererror/` for domain errors
- Never use `panic` for recoverable errors
- Return errors as the last return value

## Financial Data Handling

- Use `github.com/shopspring/decimal.Decimal` for all monetary amounts (never float64)
- Design core models as immutable where possible
- Validate financial data at all boundaries
- Handle currency conversion explicitly

## Testing Requirements

- All significant logic requires unit tests with minimum 80% coverage
- Use table-driven tests for multiple scenarios
- Test files alongside source: `parser.go` → `parser_test.go`
- Use `github.com/stretchr/testify` for assertions
- Mock external dependencies (filesystem, APIs)
- Use `testdata/` subdirectories for test fixtures
- Run tests with race detector: `go test -race`

## Configuration Management

- Use `spf13/viper` for hierarchical configuration
- Precedence: CLI flags > Environment variables > Config files > Defaults
- Environment variables use `CAMT_` prefix
- Never hardcode or log sensitive data (API keys, credentials)
- Validate configuration at startup

## Logging Standards

- Use `github.com/sirupsen/logrus` for structured logging
- Levels: DEBUG (development), INFO (operations), WARN (issues), ERROR (failures)
- Include contextual information (file names, operation types)
- Never log sensitive data (account numbers, personal information)

## Dependency Management

- Use Go Modules exclusively with pinned versions
- Run `go mod tidy` to clean unused dependencies
- Justify new dependencies with clear business value
- Keep dependencies updated and secure