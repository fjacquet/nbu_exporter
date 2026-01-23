---
inclusion: always
---

# Quality Standards

## Production Readiness Requirements

**NON-NEGOTIABLE**: Never declare code production-ready without meeting ALL quality standards:

### Testing Requirements

- Minimum 80% test coverage overall, 100% for critical paths (parsing, validation)
- All tests must pass before any changes are considered complete
- Unit tests for all significant logic using table-driven patterns
- Integration tests for CLI commands and external service interactions
- Use `go test -race` to detect data races
- Mock external dependencies (filesystem, APIs, network calls)

### Security Requirements

- All inputs must be validated and sanitized
- Never hardcode secrets, API keys, or credentials
- Use environment variables for sensitive configuration
- Run security scanners (`gosec`, `govulncheck`) and address findings
- Validate file paths to prevent directory traversal attacks
- Use HTTPS only for external API calls

### Code Quality Standards

- All code must pass `golangci-lint` checks
- Follow Go conventions (`go fmt`, `goimports`)
- All exported functions/types require godoc comments
- Error handling: always check and propagate errors with context
- Use `github.com/shopspring/decimal` for monetary amounts (never float64)
- Implement proper resource cleanup with `defer`

### Architecture Compliance

- All parsers must implement the `parser.Parser` interface
- Follow single responsibility principle
- Use dependency injection for testability
- Maintain separation between pure business logic and I/O operations
- Follow the established directory structure in `internal/`

## Quality Gates

Before marking any feature complete:

1. All tests pass with race detection enabled
2. Security scan shows no critical vulnerabilities
3. Code coverage meets minimum thresholds
4. Linting passes without warnings
5. Documentation is updated for any public APIs
6. Integration tests verify end-to-end functionality

**Remember**: Quality is not optional. These standards protect users' financial data and ensure system reliability.
