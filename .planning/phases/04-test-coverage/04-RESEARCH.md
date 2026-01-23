# Phase 4: Test Coverage - Research

**Researched:** 2026-01-23
**Domain:** Go testing patterns for CLI applications, HTTP servers, and concurrent systems
**Confidence:** HIGH

## Summary

This phase focuses on increasing test coverage across critical packages: main.go (0% to 60%+), testutil (51.9% to 80%+), and telemetry (76.6% to 90%+). Additionally, tests must verify concurrent collector access without race conditions and cover client error handling edge cases.

The Go ecosystem provides robust built-in testing tools that the project already leverages: the `testing` package, `httptest` for mock HTTP servers, and `testify` for assertions. The primary challenge is testing main.go with its Cobra CLI, HTTP server lifecycle, and signal handling - patterns that require specific techniques like constructor patterns, programmatic signal sending, and careful goroutine coordination.

Key recommendations: Use the constructor function pattern for Cobra command testing, leverage httptest.Server for HTTP integration tests, employ `-race` flag for all concurrent tests, and use programmatic signals (syscall.Kill) for shutdown testing.

**Primary recommendation:** Apply table-driven tests with httptest.Server for HTTP endpoints, constructor pattern for Cobra CLI testing, and mandatory `-race` flag for concurrent tests.

## Standard Stack

The established libraries/tools for Go testing in this project:

### Core

| Library                       | Version | Purpose                   | Why Standard                      |
| ----------------------------- | ------- | ------------------------- | --------------------------------- |
| `testing`                     | stdlib  | Test framework            | Go standard, zero dependencies    |
| `net/http/httptest`           | stdlib  | Mock HTTP servers/clients | Official Go testing utility       |
| `github.com/stretchr/testify` | v1.11.1 | Assertions and mocking    | Industry standard, already in use |

### Supporting

| Library     | Version | Purpose                       | When to Use                 |
| ----------- | ------- | ----------------------------- | --------------------------- |
| `os/signal` | stdlib  | Signal handling               | Testing graceful shutdown   |
| `syscall`   | stdlib  | Send signals programmatically | Signal-based shutdown tests |
| `context`   | stdlib  | Timeout/cancellation          | Integration test timeouts   |
| `sync`      | stdlib  | Goroutine coordination        | Concurrent tests            |

### Alternatives Considered

| Instead of           | Could Use         | Tradeoff                                  |
| -------------------- | ----------------- | ----------------------------------------- |
| testify              | Custom assertions | testify already in use, consistent API    |
| httptest             | Third-party mock  | httptest is official, battle-tested       |
| Manual race checking | -race flag        | Race detector is comprehensive, zero code |

**Installation:**
Already available - no additional dependencies required.

## Architecture Patterns

### Recommended Test Structure

```
main_test.go                    # Integration tests for main.go
internal/
  testutil/
    helpers_test.go             # Existing, expand coverage
    mockserver_test.go          # Add tests for builder patterns
  telemetry/
    manager_test.go             # Existing, add edge cases
    attributes_test.go          # Test attribute constants
  exporter/
    concurrent_test.go          # New: concurrent access tests
    client_edge_cases_test.go   # New: client error edge cases
```

### Pattern 1: Cobra Command Constructor Testing

**What:** Create testable Cobra commands by returning `*cobra.Command` from constructor functions
**When to use:** Testing CLI commands in main.go
**Example:**

```go
// Source: https://gianarb.it/blog/golang-mockmania-cli-command-with-cobra
// Create constructor that returns command
func NewRootCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "myapp",
        RunE: func(cmd *cobra.Command, args []string) error {
            // Command logic
            return nil
        },
    }
    return cmd
}

// Test the command
func TestRootCmd(t *testing.T) {
    cmd := NewRootCmd()

    // Capture output
    buf := new(bytes.Buffer)
    cmd.SetOut(buf)
    cmd.SetErr(buf)

    // Set test arguments
    cmd.SetArgs([]string{"--config", "test.yaml"})

    // Execute
    err := cmd.Execute()
    assert.NoError(t, err)
}
```

### Pattern 2: HTTP Server Integration Testing

**What:** Test full HTTP server startup, request handling, and shutdown
**When to use:** main.go Server struct tests
**Example:**

```go
// Source: https://pkg.go.dev/net/http/httptest
func TestServerStartupShutdown(t *testing.T) {
    // Create test config
    cfg := createTestConfig()

    // Start server
    server := NewServer(cfg)
    err := server.Start()
    require.NoError(t, err)

    // Give server time to start
    time.Sleep(100 * time.Millisecond)

    // Test endpoint
    resp, err := http.Get("http://" + cfg.GetServerAddress() + "/health")
    require.NoError(t, err)
    assert.Equal(t, http.StatusOK, resp.StatusCode)

    // Graceful shutdown
    err = server.Shutdown()
    assert.NoError(t, err)
}
```

### Pattern 3: Signal Handling Testing

**What:** Test graceful shutdown via SIGTERM/SIGINT
**When to use:** Testing waitForShutdown function
**Example:**

```go
// Source: https://go.dev/doc/articles/race_detector + https://gobyexample.com/signals
func TestSignalHandling(t *testing.T) {
    // Setup signal channel
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGTERM)
    defer signal.Stop(sigChan)

    // Start goroutine that waits for signal
    done := make(chan error, 1)
    go func() {
        done <- waitForShutdown(sigChan)
    }()

    // Send signal programmatically
    syscall.Kill(syscall.Getpid(), syscall.SIGTERM)

    // Verify clean shutdown
    select {
    case err := <-done:
        assert.NoError(t, err)
    case <-time.After(5 * time.Second):
        t.Fatal("Timeout waiting for signal handling")
    }
}
```

### Pattern 4: Concurrent Access Testing

**What:** Verify thread-safe access to shared resources
**When to use:** Collector concurrent access tests
**Example:**

```go
// Source: https://go.dev/doc/articles/race_detector
func TestCollectorConcurrentAccess(t *testing.T) {
    collector := createTestCollector()

    var wg sync.WaitGroup
    const numGoroutines = 10

    for i := 0; i < numGoroutines; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            ch := make(chan prometheus.Metric, 100)
            go func() {
                collector.Collect(ch)
                close(ch)
            }()
            // Drain channel
            for range ch {
            }
        }()
    }

    wg.Wait()
}
```

### Anti-Patterns to Avoid

- **Time-based synchronization:** Never use `time.Sleep` for goroutine coordination; use channels or sync primitives
- **Shared loop variables in goroutines:** Always pass loop variables as parameters to avoid races
- **Skipping -race flag:** Always run `go test -race` for concurrent code
- **Testing against real servers:** Use httptest.Server, not real external services
- **Ignoring context timeouts:** Always use context.WithTimeout for integration tests

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem            | Don't Build             | Use Instead                           | Why                                   |
| ------------------ | ----------------------- | ------------------------------------- | ------------------------------------- |
| Mock HTTP servers  | Custom HTTP listener    | `httptest.NewServer`                  | Handles port allocation, cleanup, TLS |
| Race detection     | Manual mutex audits     | `go test -race`                       | Comprehensive, catches runtime races  |
| Assertion helpers  | Custom t.Error wrappers | `testify/assert`                      | Rich API, consistent, diff output     |
| Test configuration | Hardcoded values        | `testutil` constants                  | Centralized, consistent across tests  |
| Signal sending     | Manual process spawning | `syscall.Kill(syscall.Getpid(), sig)` | Same-process, reliable                |

**Key insight:** Go's standard library provides excellent testing utilities. The httptest package specifically handles edge cases (port conflicts, cleanup, TLS certificates) that would be error-prone to implement manually.

## Common Pitfalls

### Pitfall 1: Flaky Signal Tests

**What goes wrong:** Signal tests pass locally but fail in CI due to timing issues
**Why it happens:** Signal delivery timing varies across systems; tests rely on implicit timing
**How to avoid:** Use buffered channels (capacity 1) and explicit synchronization; set reasonable timeouts
**Warning signs:** Tests that need `time.Sleep` before assertions

### Pitfall 2: Race Conditions in Test Setup

**What goes wrong:** Tests pass without `-race` but fail with it
**Why it happens:** Shared test fixtures modified concurrently; loop variables captured by reference
**How to avoid:** Always run `go test -race`; use t.Parallel() carefully; copy loop variables
**Warning signs:** Intermittent test failures, tests that fail when run in parallel

### Pitfall 3: Port Conflicts in Integration Tests

**What goes wrong:** Tests fail with "address already in use" errors
**Why it happens:** Fixed port numbers conflict when tests run in parallel
**How to avoid:** Use httptest.Server (allocates free ports) or `:0` for random port
**Warning signs:** Tests that work alone but fail when run together

### Pitfall 4: Leaked Goroutines

**What goes wrong:** Tests hang or leak resources
**Why it happens:** Goroutines not properly terminated; channels not closed
**How to avoid:** Always close channels; use defer for cleanup; use context cancellation
**Warning signs:** Increasing memory usage during test runs; test hangs

### Pitfall 5: Incomplete Server Shutdown Testing

**What goes wrong:** Server shutdown tests pass but miss edge cases
**Why it happens:** Only testing happy path; not testing in-flight requests
**How to avoid:** Test shutdown during active requests; verify all resources cleaned up
**Warning signs:** Resource leaks in production; connection pool exhaustion

## Code Examples

Verified patterns from official sources:

### Testing HTTP Handler Response

```go
// Source: https://pkg.go.dev/net/http/httptest
func TestHealthHandler(t *testing.T) {
    req := httptest.NewRequest(http.MethodGet, "/health", nil)
    w := httptest.NewRecorder()

    server := &Server{}
    server.healthHandler(w, req)

    resp := w.Result()
    assert.Equal(t, http.StatusOK, resp.StatusCode)

    body, _ := io.ReadAll(resp.Body)
    assert.Contains(t, string(body), "OK")
}
```

### Testing with Context Timeout

```go
// Source: Go stdlib patterns
func TestServerWithTimeout(t *testing.T) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    server := createTestServer()
    err := server.StartWithContext(ctx)
    require.NoError(t, err)

    // Test operations...

    // Shutdown respects context
    err = server.ShutdownWithContext(ctx)
    assert.NoError(t, err)
}
```

### Testing Client Error Handling

```go
// Source: existing client_test.go patterns
func TestClientNetworkTimeout(t *testing.T) {
    // Server that delays response
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        time.Sleep(10 * time.Second) // Longer than client timeout
    }))
    defer server.Close()

    cfg := createTestConfig()
    client := NewNbuClient(cfg)
    client.client.SetTimeout(100 * time.Millisecond) // Short timeout

    ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
    defer cancel()

    var result interface{}
    err := client.FetchData(ctx, server.URL, &result)

    assert.Error(t, err)
    assert.Contains(t, err.Error(), "timeout")
}
```

### Table-Driven Test Template

```go
// Source: Go testing best practices
func TestValidateConfig(t *testing.T) {
    tests := []struct {
        name      string
        configPath string
        wantErr   bool
        errMsg    string
    }{
        {
            name:       "valid config",
            configPath: "testdata/valid.yaml",
            wantErr:    false,
        },
        {
            name:       "missing file",
            configPath: "nonexistent.yaml",
            wantErr:    true,
            errMsg:     "config file not found",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            cfg, err := validateConfig(tt.configPath)

            if tt.wantErr {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.errMsg)
                assert.Nil(t, cfg)
            } else {
                assert.NoError(t, err)
                assert.NotNil(t, cfg)
            }
        })
    }
}
```

## State of the Art

| Old Approach           | Current Approach               | When Changed      | Impact                    |
| ---------------------- | ------------------------------ | ----------------- | ------------------------- |
| Manual port allocation | httptest.Server auto-ports     | Go 1.0+           | Eliminates port conflicts |
| t.Error + manual diffs | testify/assert                 | Mature since 2015 | Better error messages     |
| Manual race audits     | `-race` flag                   | Go 1.1            | Catches runtime races     |
| httptest.NewRequest    | httptest.NewRequestWithContext | Go 1.23           | Context-aware testing     |

**Deprecated/outdated:**

- None for core testing patterns; Go testing has been stable

## Open Questions

Things that couldn't be fully resolved:

1. **Main package test file location**
   - What we know: main_test.go can exist alongside main.go
   - What's unclear: Whether to use build constraints to exclude from production
   - Recommendation: Place main_test.go in root; no build constraints needed since test files are excluded automatically

2. **OpenTelemetry test coverage without collector**
   - What we know: Telemetry tests currently work with graceful degradation
   - What's unclear: How to test actual span export without running collector
   - Recommendation: Use noop tracer provider for unit tests; mock OTLP endpoint for integration tests if needed

## Sources

### Primary (HIGH confidence)

- [Go Race Detector](https://go.dev/doc/articles/race_detector) - Race detection patterns, enabling, common issues
- [httptest package](https://pkg.go.dev/net/http/httptest) - HTTP testing utilities
- [os/signal package](https://pkg.go.dev/os/signal) - Signal handling
- [Go by Example: Signals](https://gobyexample.com/signals) - Signal handling patterns

### Secondary (MEDIUM confidence)

- [Graceful Shutdown in Go: Practical Patterns](https://victoriametrics.com/blog/go-graceful-shutdown/) - Production patterns (May 2025)
- [Testing Cobra CLI](https://gianarb.it/blog/golang-mockmania-cli-command-with-cobra) - Cobra testing patterns
- [Testing Golang with httptest](https://speedscale.com/blog/testing-golang-with-httptest/) - httptest best practices

### Tertiary (LOW confidence)

- GitHub discussions on Cobra testing patterns - community consensus

## Metadata

**Confidence breakdown:**

- Standard stack: HIGH - Using Go stdlib and established testify
- Architecture: HIGH - Patterns verified against official documentation
- Pitfalls: HIGH - Based on race detector documentation and production experience

**Research date:** 2026-01-23
**Valid until:** 2026-02-23 (30 days - stable domain)

## Test Coverage Targets Summary

| Package     | Current | Target | Strategy                                                |
| ----------- | ------- | ------ | ------------------------------------------------------- |
| main (root) | 0%      | 60%+   | Integration tests: server lifecycle, signals, CLI       |
| testutil    | 51.9%   | 80%+   | Test MockServerBuilder methods, LoadTestData edge cases |
| telemetry   | 76.6%   | 90%+   | Test TracerProvider, edge cases in Initialize           |
| exporter    | 88.1%   | 90%+   | Concurrent access tests, add -race to CI                |

### Specific Functions to Test in main.go

1. `NewServer()` - Constructor with various configs
2. `Server.Start()` - Startup success/failure paths
3. `Server.Shutdown()` - Graceful shutdown, timeout handling
4. `Server.ErrorChan()` - Error propagation
5. `validateConfig()` - Valid/invalid config paths
6. `setupLogging()` - Debug mode toggling
7. `waitForShutdown()` - Signal handling (SIGINT, SIGTERM)
8. `healthHandler()` - HTTP response verification
9. `extractTraceContextMiddleware()` - Trace context extraction

### Testing Commands

```bash
# Run all tests with race detection
go test -race ./...

# Run with coverage report
go test -cover -coverprofile=coverage.out ./...

# View coverage in browser
go tool cover -html=coverage.out

# Run specific package tests
go test -race -v ./internal/testutil/...
go test -race -v ./internal/telemetry/...

# Run main package tests
go test -race -v .
```
