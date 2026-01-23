---
phase: 05-performance-optimizations
plan: 02
subsystem: exporter
tags: [errgroup, parallelism, goroutines, sync, performance]

# Dependency graph
requires:
  - phase: 03-architecture-improvements
    provides: Structured metric types (StorageMetricValue, JobMetricValue)
provides:
  - Parallel metric collection in collectAllMetrics
  - errgroup coordination for storage/job fetch
  - Graceful degradation with concurrent fetches
affects: [performance monitoring, scrape latency, collector tests]

# Tech tracking
tech-stack:
  added: [golang.org/x/sync v0.19.0]
  patterns: [errgroup.WithContext, graceful degradation with nil returns]

key-files:
  created: []
  modified: [internal/exporter/prometheus.go, go.mod, go.sum]

key-decisions:
  - "Always return nil from g.Go() to preserve graceful degradation behavior"
  - "Pass gCtx (group context) to both collectors for proper cancellation propagation"
  - "Errors tracked in separate variables rather than errgroup error return"

patterns-established:
  - "Parallel collection pattern: g.Go() with nil return for concurrent-but-independent operations"
  - "Graceful degradation: failure in one fetch path does not affect the other"

# Metrics
duration: 4min
completed: 2026-01-23
---

# Phase 5 Plan 02: Parallel Collection with errgroup Summary

**Parallel storage and job metric collection using errgroup, reducing scrape time from sum to max of individual fetch times**

## Performance

- **Duration:** 4 min
- **Started:** 2026-01-23T21:48:00Z
- **Completed:** 2026-01-23T21:52:00Z
- **Tasks:** 3
- **Files modified:** 3 (prometheus.go, go.mod, go.sum)

## Accomplishments

- Storage and job metrics now fetched concurrently with errgroup
- Total scrape time reduced from sum(storage_time + jobs_time) to max(storage_time, jobs_time)
- Graceful degradation preserved: storage failure does not cancel job fetching and vice versa
- All existing tests pass with race detector (no concurrency issues introduced)

## Task Commits

Each task was committed atomically:

1. **Task 1: Add errgroup Dependency** - `f16f48b` (chore)
2. **Task 2: Implement Parallel Collection in collectAllMetrics** - `58fd633` (feat)
3. **Task 3: Run Full Test Suite** - No commit needed (verification only)

## Files Created/Modified

- `internal/exporter/prometheus.go` - Added errgroup import, refactored collectAllMetrics for parallel execution
- `go.mod` - Added golang.org/x/sync v0.19.0 dependency
- `go.sum` - Updated checksums for new dependency

## Decisions Made

1. **Always return nil from g.Go():** Errors are tracked in separate variables (storageErr, jobsErr) rather than using errgroup's error return. This ensures that one failure doesn't cancel the other goroutine via context cancellation, preserving the existing graceful degradation behavior.

2. **Use gCtx (group context) for collectors:** Both collectStorageMetrics and collectJobMetrics receive the errgroup's derived context (gCtx) rather than the original context. This ensures proper cancellation propagation if the parent context is cancelled (e.g., scrape timeout).

3. **Closures capture result variables safely:** Each goroutine writes to different result variables (storageMetrics vs jobsSize/jobsCount/jobsStatusCount), so there are no data races even without additional synchronization.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Sync vendor directory after adding dependency**
- **Found during:** Task 1 (Add errgroup Dependency)
- **Issue:** Project uses vendoring; `go build` failed with "inconsistent vendoring" error
- **Fix:** Ran `go mod vendor` to sync vendor directory
- **Files modified:** vendor/ (gitignored)
- **Verification:** Build succeeds, tests pass
- **Committed in:** Not committed (vendor is gitignored)

---

**Total deviations:** 1 auto-fixed (blocking issue with vendoring)
**Impact on plan:** Minor operational fix for vendored dependencies. No scope creep.

## Issues Encountered

None - plan executed smoothly after vendor sync.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Parallel collection in place, ready for next performance optimizations
- 05-03 (Response Caching) can proceed with confidence in concurrent access patterns
- All concurrent tests validate thread safety of the parallel implementation

---
*Phase: 05-performance-optimizations*
*Completed: 2026-01-23*
