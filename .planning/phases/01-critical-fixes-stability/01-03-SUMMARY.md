---
phase: 01-critical-fixes-stability
plan: 03
subsystem: http-client
tags: [go, http, connection-pooling, graceful-shutdown, resource-cleanup, concurrency]

# Dependency graph
requires:
  - phase: 01-01
    provides: Immutable API version detector
provides:
  - Graceful connection draining with configurable timeout
  - Thread-safe connection tracking for active requests
  - Idempotent Close() preventing double-close errors
affects: [02-security, 03-architecture, main.go shutdown]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "sync.Mutex with atomic counters for connection tracking"
    - "Local channel references to prevent race conditions"
    - "Context-based timeout control for shutdown operations"

key-files:
  created: []
  modified:
    - internal/exporter/client.go
    - internal/exporter/client_test.go

key-decisions:
  - "Use atomic int32 for activeReqs counter to minimize lock contention"
  - "Store local channel reference before releasing lock to prevent data race"
  - "Default 30-second timeout for Close() balances graceful shutdown vs hang prevention"
  - "CloseWithContext() provides custom timeout control for advanced use cases"

patterns-established:
  - "Connection tracking pattern: increment on request start, decrement on defer, signal on zero"
  - "Graceful shutdown pattern: mark closed, wait for drain with timeout, then force cleanup"
  - "Race-free channel usage: store local reference while holding lock, then use local ref after release"

# Metrics
duration: 10min
completed: 2026-01-23
---

# Phase 01 Plan 03: Resource Cleanup Summary

**Graceful HTTP client shutdown with connection draining, atomic request tracking, and race-free timeout handling**

## Performance

- **Duration:** 10 min
- **Started:** 2026-01-23T03:20:37Z
- **Completed:** 2026-01-23T03:30:28Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments

- NbuClient now tracks active requests with atomic counter and mutex-protected state
- Close() waits up to 30 seconds for active requests to complete before forcing cleanup
- CloseWithContext() allows custom timeout control for advanced shutdown scenarios
- All tests pass with race detector, including concurrent close scenarios
- Fixed blocking compilation issue from plan 01-01 (NewAPIVersionDetector signature)

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement proper Close() with connection tracking** - `67cf9c9` (feat)
2. **Task 2: Update interface and add cleanup tests** - `e14e191` (test)

## Files Created/Modified

- `internal/exporter/client.go` - Added connection tracking fields (activeReqs, closed, closeChan), updated FetchData() to track request lifecycle, implemented Close() and CloseWithContext() with timeout
- `internal/exporter/client_test.go` - Added TestNbuClientCloseIdempotent, TestNbuClientCloseWaitsForActiveRequests, TestNbuClientFetchDataRejectsAfterClose, TestNbuClientCloseTimeout

## Decisions Made

- **Atomic counter for activeReqs:** Using atomic.Int32 minimizes lock contention since FetchData is called frequently. Mutex only protects closed flag and closeChan, not the hot path.
- **Local channel reference pattern:** Close() stores `ch := c.closeChan` while holding lock, then releases lock before reading from `ch`. This prevents race condition where FetchData's defer closes the channel while Close() is reading it.
- **30-second default timeout:** Balances graceful shutdown (most requests complete in < 2 minutes per existing timeout) vs preventing indefinite hangs on exporter restart.
- **Separate CloseWithContext():** Allows callers to specify custom timeout (e.g., shorter for testing, longer for production) without changing Close() semantics.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Fixed NewAPIVersionDetector call signature**

- **Found during:** Task 1 (building after adding connection tracking)
- **Issue:** Plan 01-01 changed NewAPIVersionDetector to accept (client, baseURL, apiKey) but calls in client.go and test files still used old (client, cfg) signature. Build failed with "not enough arguments" error.
- **Fix:** Updated performVersionDetectionIfNeeded() to extract baseURL and apiKey from config and pass individually. Updated test file calls similarly.
- **Files modified:** internal/exporter/client.go (line 156-157), internal/exporter/version_detection_integration_test.go (line 121, 166)
- **Verification:** `go build ./...` succeeds, all tests pass
- **Committed in:** 67cf9c9 (Task 1 commit)

**2. [Rule 1 - Bug] Fixed data race in Close() channel access**

- **Found during:** Task 2 (running race detector on tests)
- **Issue:** Close() created closeChan, released lock, then read from c.closeChan. FetchData's defer could close and nil out c.closeChan concurrently, causing data race detected by `go test -race`.
- **Fix:** Store local reference `ch := c.closeChan` while holding lock, then read from `ch` after releasing lock. Applied to both Close() and CloseWithContext().
- **Files modified:** internal/exporter/client.go (Close and CloseWithContext methods)
- **Verification:** `go test ./internal/exporter/... -race` passes with no race conditions
- **Committed in:** 67cf9c9 (Task 1 commit)

---

**Total deviations:** 2 auto-fixed (1 blocking, 1 bug)
**Impact on plan:** Blocking fix from previous plan was necessary to compile. Race fix was caught by tests and essential for correctness. No scope creep.

## Issues Encountered

None - implementation followed plan with only necessary bug fixes.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Proper resource cleanup complete, ready for security hardening in Phase 2
- Connection tracking pattern can be applied to other resources (telemetry manager, prometheus registry)
- No blockers for next phase

**Note for Phase 02 (Security):** If implementing connection limits or rate limiting, consider adding maxActiveReqs field to NbuClient to reject new requests when at capacity.

---
*Phase: 01-critical-fixes-stability*
*Completed: 2026-01-23*
