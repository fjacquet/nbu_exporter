---
phase: 04-test-coverage
plan: 04
subsystem: exporter-testing
tags: [testing, concurrency, race-detection, error-handling]
dependency-graph:
  requires: [04-01, 04-02, 04-03]
  provides: [concurrent-tests, client-edge-case-tests]
  affects: []
tech-stack:
  added: []
  patterns: [table-driven-tests, race-detection, network-mock-testing]
key-files:
  created:
    - internal/exporter/concurrent_test.go
  modified:
    - internal/exporter/client_test.go
decisions:
  - 10-goroutine concurrency level for collector tests
  - 100ms context timeout for network timeout tests
  - Table-driven tests for HTTP error codes
metrics:
  duration: 5 minutes
  completed: 2026-01-23
---

# Phase 04 Plan 04: Concurrent Tests & Client Edge Cases Summary

**One-liner:** Added concurrent collector access tests with race detection and comprehensive client error handling edge case tests.

## What Was Built

### 1. Concurrent Collector Access Tests (concurrent_test.go)
Created comprehensive concurrent access tests for the NbuCollector:

- **TestCollectorConcurrentCollect**: Verifies 10 goroutines can safely call Collect() simultaneously
- **TestCollectorConcurrentDescribe**: Verifies parallel Describe() calls are thread-safe
- **TestCollectorCollectDuringClose**: Tests graceful handling when Close() is called during Collect()
- **TestCollectorConcurrentCollectAndDescribe**: Tests mixed Collect/Describe concurrent operations
- **TestCollectorMultipleCloseAttempts**: Verifies Close() is idempotent (only succeeds once)
- **TestCollectorCloseWithActiveCollect**: Tests Close() waiting for active requests

### 2. Client Error Handling Edge Case Tests
Added comprehensive error handling tests to client_test.go:

- **TestClientNetworkTimeout**: Server doesn't respond within context timeout (100ms timeout vs 5s delay)
- **TestClientConnectionRefused**: Connection to unreachable server (localhost:65534)
- **TestClientHTTPErrorsComprehensive**: Table-driven test for 400, 401, 403, 404, 500, 502, 503
- **TestClientPartialResponse**: Truncated JSON responses (incomplete object, array, key)
- **TestClientEmptyResponseBody**: Empty response body handling
- **TestClientServerClosesDuringTransfer**: Server closes connection mid-transfer

## Key Patterns

### Concurrent Test Pattern
```go
const numGoroutines = 10
var wg sync.WaitGroup

for i := 0; i < numGoroutines; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()
        ch := make(chan prometheus.Metric, 100)
        go func() {
            collector.Collect(ch)
            close(ch)
        }()
        for range ch {
        }
    }(i)
}
wg.Wait()
```

### Network Timeout Test Pattern
```go
ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
defer cancel()
err := client.FetchData(ctx, slowServerURL, &result)
// Verify error indicates timeout
```

### Connection Close Test Pattern
```go
listener, _ := net.Listen("tcp", "127.0.0.1:0")
go func() {
    conn, _ := listener.Accept()
    conn.Read(buf)
    conn.Close()  // Close without response
}()
err := client.FetchData(ctx, testURL, &result)
// Verify error handling
```

## Test Results

### Race Detector Verification
- **Run 1**: PASS (27.203s)
- **Run 2**: PASS (27.051s)
- **Run 3**: PASS (27.104s)
- **Race conditions detected**: 0

### Coverage
- **Before**: ~88.1%
- **After**: 90.2%
- **Improvement**: +2.1%

### Test Duration
- All tests complete in ~27 seconds (well under 60 second limit)

## Commits

| Hash | Description |
|------|-------------|
| f3332af | test(04-04): add concurrent collector access tests |
| 93e4e5e | test(04-04): add client error handling edge case tests |

## Deviations from Plan

None - plan executed exactly as written.

## Requirements Addressed

- **TEST-04**: Concurrent collector access tested with race detector
- **TEST-05**: Client handles network timeouts and unusual HTTP responses correctly

## Files Changed

### Created
- `internal/exporter/concurrent_test.go` (368 lines)
  - 6 test functions for concurrent collector access

### Modified
- `internal/exporter/client_test.go` (+266 lines)
  - 6 new test functions for client edge cases

## Next Steps

Phase 04 Plan 04 complete. Ready to proceed with remaining Phase 04 plans or Phase 05.
