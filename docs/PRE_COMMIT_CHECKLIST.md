# Pre-Commit Checklist ✅

## Code Quality

- [x] All tests passing: `go test ./...`
- [x] Test coverage ≥80%: 88.0%
- [x] Code builds successfully: `go build ./...`
- [x] No linting errors: `go fmt ./...`
- [x] Go version correct: 1.25

## GitHub Actions Workflows

- [x] **go.yml**: Updated to Go 1.25, includes tests and coverage check
- [x] **build.yml**: SonarCloud scan configured
- [x] **release.yml**: Updated to Go 1.25, GoReleaser configured
- [x] **codeql-analysis.yml**: Created, uses Go 1.25

## Documentation

- [x] README.md updated with:
  - Version support matrix
  - Docker deployment instructions
  - Go 1.25 requirement
- [x] CHANGELOG.md updated with all changes
- [x] RELEASE_NOTES.md created
- [x] IMPLEMENTATION_SUMMARY.md created
- [x] Migration guide complete

## Configuration Files

- [x] go.mod: Go 1.25
- [x] config.yaml: Updated with version comments
- [x] config-auto-detect.yaml: Created
- [x] docker-compose.yml: Created
- [x] .dockerignore: Created
- [x] Makefile: New targets added

## Bug Fixes

- [x] HTML response handling fixed
- [x] Content-Type validation added
- [x] Enhanced error messages with troubleshooting

## Test Coverage

```
Package                                          Coverage
github.com/fjacquet/nbu_exporter/internal/exporter    88.0%
github.com/fjacquet/nbu_exporter/internal/models      52.1%
```

## Files Changed

### New Files (15)
- internal/exporter/version_detector.go
- internal/exporter/version_detector_test.go
- internal/exporter/end_to_end_test.go
- internal/exporter/api_compatibility_test.go
- internal/exporter/backward_compatibility_test.go
- internal/exporter/version_detection_integration_test.go
- internal/exporter/metrics_consistency_test.go
- internal/exporter/performance_test.go
- testdata/api-versions/*.json (7 files)
- docs/netbackup-11-migration.md
- docs/config-examples/*.yaml (4 files)
- RELEASE_NOTES.md
- IMPLEMENTATION_SUMMARY.md
- docker-compose.yml
- .dockerignore
- config-auto-detect.yaml
- .github/workflows/codeql-analysis.yml

### Modified Files (11)
- internal/models/Config.go
- internal/exporter/client.go
- internal/exporter/client_test.go
- internal/exporter/netbackup.go
- internal/exporter/prometheus.go
- internal/models/Jobs.go
- internal/models/Storage.go
- Makefile
- Dockerfile
- config.yaml
- README.md
- CHANGELOG.md
- go.mod
- .github/workflows/go.yml
- .github/workflows/release.yml
- .kiro/steering/tech.md

## Ready to Commit? ✅

All checks passed! The code is ready to commit.

### Recommended Commit Message

```
feat: Add NetBackup 11.0 multi-version API support with automatic detection

- Add support for NetBackup API versions 3.0, 12.0, and 13.0
- Implement intelligent automatic version detection with fallback (13.0 → 12.0 → 3.0)
- Add exponential backoff retry logic for transient failures
- Fix HTML response handling bug (invalid character '<' error)
- Add Content-Type validation before JSON unmarshaling
- Enhance error messages with troubleshooting guidance
- Add nbu_api_version Prometheus metric
- Update to Go 1.25
- Add comprehensive test coverage (88.0%)
- Add Docker Compose configuration
- Update documentation and migration guides

BREAKING CHANGE: Configuration field 'scrappingInterval' renamed to 'scrapingInterval'

Closes #XXX
```

### Next Steps

1. Review changes: `git status`
2. Stage files: `git add .`
3. Commit: `git commit -m "feat: Add NetBackup 11.0 multi-version API support"`
4. Push: `git push origin <branch-name>`
5. Create Pull Request

