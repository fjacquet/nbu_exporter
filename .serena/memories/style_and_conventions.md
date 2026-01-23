# Style and Conventions for NBU Exporter

## Code Style

- Standard Go formatting (`go fmt`)
- golangci-lint for linting
- Use `internal/` for private packages

## Naming Conventions

- PascalCase for exported identifiers (types, functions, constants)
- camelCase for unexported identifiers
- Constants use SCREAMING_SNAKE_CASE for environment variables, otherwise PascalCase
- Test files: `*_test.go` in same package

## Documentation

- Package-level doc comments in each file
- Godoc comments for all exported symbols
- Include examples in doc comments where helpful

## Error Handling

- Always wrap errors with context: `fmt.Errorf("action failed: %w", err)`
- Never log API keys or sensitive data in errors
- Use structured logging with logrus

## Testing Patterns

- Use table-driven tests with `t.Run()`
- Use httptest for HTTP mocking
- Standard library testing preferred (no testify in some areas)
- Test file naming: `*_test.go`

## Configuration

- YAML configuration via `config.yaml`
- Validation in `Config.Validate()`
- Defaults set in `Config.SetDefaults()`

## Security

- TLS 1.2 minimum enforced
- API keys masked in logs (use `MaskAPIKey()`)
- InsecureSkipVerify requires explicit opt-in in config

## Metrics Pattern

- Pipe-delimited keys split into labels
- Storage: `name|type|size`
- Jobs: `action|policy_type|status`
