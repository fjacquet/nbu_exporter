---
phase: 04-test-coverage
plan: 01
subsystem: main
tags: [testing, integration, coverage, http, lifecycle]

dependency-graph:
  requires: [03-05]
  provides: [main-package-tests, coverage-baseline]
  affects: [04-02, 04-03, 04-04]

tech-stack:
  added: []
  patterns: [table-driven-tests, mock-server-testing, integration-tests]

key-files:
  created:
    - main_test.go
    - testdata/malformed_config.yaml
  modified:
    - testdata/valid_config.yaml
    - testdata/invalid_config.yaml
    - internal/testutil/helpers_test.go

decisions:
  - id: TEST-01
    decision: Use httptest.NewTLSServer for mocking NBU API
    rationale: Enables testing Server.Start() with actual collector creation
  - id: TEST-02
    decision: Use port 0 for test servers
    rationale: Let OS assign random available port to avoid conflicts

metrics:
  duration: 3 minutes
  completed: 2026-01-23
---

# Phase 04 Plan 01: Main Package Integration Tests Summary

**One-liner:** Comprehensive integration tests for main.go covering server lifecycle, config validation, and signal handling (60% coverage)

## What Was Built

### Test Coverage

Created `main_test.go` with 25 tests covering:

| Test Category | Tests | Coverage |
|---------------|-------|----------|
| Server Initialization | TestNewServer, TestNewServerWithOTel | NewServer function |
| Config Validation | TestValidateConfig_Success, FileNotFound, InvalidConfig, MalformedYAML | validateConfig function |
| Logging Setup | TestSetupLogging_Success, DebugMode, InvalidPath | setupLogging function |
| Health Endpoint | TestHealthHandler, AllMethods | healthHandler method |
| Signal Handling | TestWaitForShutdown_Signal, Error, NilError | waitForShutdown function |
| Server Lifecycle | TestServerStartShutdown_Integration | Start/Shutdown sequence |
| Error Channel | TestServerErrorChan, TestServerErrorPropagation | Error channel pattern |
| Middleware | TestExtractTraceContextMiddleware (3 variants) | Trace context extraction |
| Config Helpers | TestConfigGetServerAddress, GetNBUBaseURL | Config methods |
| Benchmarks | BenchmarkNewServer, HealthHandler, ValidateConfig | Performance baselines |

### Test Fixtures

- **testdata/valid_config.yaml** - Valid configuration for testing (updated with proper port 8080)
- **testdata/invalid_config.yaml** - Missing nbuserver section for validation error testing
- **testdata/malformed_config.yaml** - Invalid YAML syntax for parsing error testing

### Key Test Patterns

1. **Mock NBU Server** - Uses `httptest.NewTLSServer` to simulate NetBackup API
2. **Port 0 Pattern** - Uses OS-assigned ports to avoid test conflicts
3. **Signal Testing** - Sends SIGINT to own process to test signal handling
4. **Error Channel Pattern** - Verifies error propagation through buffered channels

## Coverage Achieved

```
github.com/fjacquet/nbu_exporter    coverage: 60.0% of statements
```

Target: 60%+ achieved exactly at 60.0%

## Commits

| Hash | Type | Description |
|------|------|-------------|
| 1fdb367 | test | Add configuration test fixtures |
| a33f943 | test | Add main package integration tests |
| a754832 | fix | Remove unused fmt import in testutil |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Fixed unused fmt import in testutil**

- **Found during:** Full test suite verification
- **Issue:** `internal/testutil/helpers_test.go` had unused `"fmt"` import causing build failure
- **Fix:** Removed unused import
- **Files modified:** internal/testutil/helpers_test.go
- **Commit:** a754832

## Verification

All success criteria met:

- [x] main_test.go exists with comprehensive tests (25 tests)
- [x] Test fixtures exist in testdata/ directory (3 files)
- [x] All tests pass with `go test -race -v .` (25/25 pass)
- [x] Coverage reaches 60%+ for main package (exactly 60.0%)

## Next Phase Readiness

Phase 04-02 (Exporter Package Tests) can proceed:

- Test patterns established
- Mock server infrastructure available
- No blockers identified
