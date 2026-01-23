# What to Do When a Task is Completed

## Before Committing
1. **Format code**: `go fmt ./...`
2. **Run tests**: `go test ./...` (or `go test ./... -race` for race detection)
3. **Build check**: `go build ./...`
4. **Lint**: `golangci-lint run`
5. **Full quality check**: `make sure` (runs all above)

## Commit Guidelines
- Use conventional commits: `feat:`, `fix:`, `refactor:`, `test:`, `docs:`, `perf:`, `chore:`
- Include scope when relevant: `feat(exporter):`, `fix(config):`
- Reference requirements if applicable
- Co-authored-by for AI assistance

## Test Coverage
- All new code should have tests
- Run `go test ./... -cover` to check coverage
- Critical paths should have high coverage

## Quality Checklist
- [ ] Tests pass (including race detector)
- [ ] Build succeeds
- [ ] Linter passes
- [ ] No sensitive data in logs/errors
- [ ] Documentation updated if needed
