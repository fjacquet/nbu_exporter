---
phase: 06-operational-features
plan: 02
subsystem: exporter
tags: [prometheus, health-check, metrics, observability]

# Dependency graph
requires:
  - phase: 06-01
    provides: Storage metrics caching foundation
provides:
  - nbu_up metric for connectivity alerting
  - nbu_last_scrape_timestamp_seconds for staleness detection
  - TestConnectivity method for health verification
  - Enhanced /health endpoint with NBU connectivity check
affects: [monitoring, alerting, kubernetes-probes, load-balancer-health]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Health check with connectivity verification
    - Scrape timestamp tracking with mutex protection
    - Graceful degradation (up=1 if any collection succeeds)

key-files:
  created:
    - internal/exporter/health.go
    - internal/exporter/health_test.go
  modified:
    - internal/exporter/prometheus.go
    - internal/exporter/prometheus_test.go
    - internal/exporter/concurrent_test.go
    - main.go
    - main_test.go

key-decisions:
  - "nbu_up=1 if ANY collection succeeds, 0 only if ALL fail (partial success = healthy)"
  - "Cache hit updates lastStorageScrapeTime (cached data is still valid data)"
  - "Health endpoint returns 503 on NBU unreachable with lightweight version endpoint test"
  - "Startup phase returns 200 'OK (starting)' before collector initialization"
  - "5-second default timeout for health check connectivity test"

patterns-established:
  - "Health endpoint with connectivity verification pattern"
  - "Scrape timestamp tracking for staleness metrics"

# Metrics
duration: 11min
completed: 2026-01-23
---

# Phase 6 Plan 02: Health Check & Up Metric Summary

**nbu_up metric with connectivity verification and nbu_last_scrape_timestamp_seconds for staleness detection, plus enhanced /health endpoint returning 503 when NBU unreachable**

## Performance

- **Duration:** 11 min
- **Started:** 2026-01-23T22:11:47Z
- **Completed:** 2026-01-23T22:22:57Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments

- Added nbu_up metric (1 if any collection succeeds, 0 if all fail)
- Added nbu_last_scrape_timestamp_seconds metric with source label (storage/jobs)
- Created TestConnectivity() method using lightweight version detection endpoint
- Enhanced /health endpoint to verify NBU connectivity with 5-second timeout
- Added IsHealthy() method for quick check without API call

## Task Commits

Each task was committed atomically:

1. **Task 1: Add nbu_up and nbu_last_scrape_timestamp_seconds metrics** - `9889357` (feat)
2. **Task 2: Create TestConnectivity and enhanced health endpoint** - `119f1de` (feat)

## Files Created/Modified

- `internal/exporter/prometheus.go` - Added nbu_up and nbu_last_scrape_timestamp_seconds metrics, timestamp tracking fields with mutex
- `internal/exporter/health.go` - New file with TestConnectivity() and IsHealthy() methods
- `internal/exporter/health_test.go` - Comprehensive tests for health check functionality
- `internal/exporter/prometheus_test.go` - Updated to expect 8 metric descriptors
- `internal/exporter/concurrent_test.go` - Updated descriptor count expectation
- `main.go` - Enhanced healthHandler with NBU connectivity verification
- `main_test.go` - Fixed CacheTTL field, updated test expectations for startup phase

## Decisions Made

1. **Partial success = healthy** - nbu_up=1 if either storage OR jobs collection succeeds. This follows Prometheus best practice where partial data is better than no data. Only when ALL collections fail does nbu_up=0.

2. **Cache hit updates timestamp** - When storage metrics come from cache, we still update lastStorageScrapeTime because the cached data is valid and recent. This prevents false staleness alerts.

3. **Startup phase handling** - Health endpoint returns 200 "OK (starting)" when collector is nil (before Start() completes). This allows load balancers to receive healthy status during application initialization.

4. **Lightweight connectivity test** - TestConnectivity uses DetectAPIVersion which calls `/admin/jobs?page[limit]=1`. This endpoint is:
   - Lightweight (returns minimal data)
   - Validates both connectivity AND authentication
   - Doesn't modify server state

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed test expectations for new metric count**
- **Found during:** Task 1 (Build verification)
- **Issue:** prometheus_test.go and concurrent_test.go expected 6 descriptors but we now have 8
- **Fix:** Updated expectedDescriptors array and count assertions
- **Files modified:** internal/exporter/prometheus_test.go, internal/exporter/concurrent_test.go
- **Verification:** Tests pass
- **Committed in:** 9889357 (Task 1 commit)

**2. [Rule 3 - Blocking] Fixed CacheTTL missing in main_test.go config struct**
- **Found during:** Task 2 (Full test suite verification)
- **Issue:** createTestConfig() in main_test.go didn't include CacheTTL field added in 06-01
- **Fix:** Added CacheTTL field to Server struct literal in test config
- **Files modified:** main_test.go
- **Verification:** Tests compile and pass
- **Committed in:** 119f1de (Task 2 commit)

**3. [Rule 1 - Bug] Fixed TestHealthHandler assertion for new behavior**
- **Found during:** Task 2 (Test verification)
- **Issue:** Test expected "OK\n" but healthHandler now returns "OK (starting)\n" when collector is nil
- **Fix:** Updated test assertion to expect new startup phase response
- **Files modified:** main_test.go
- **Verification:** Test passes
- **Committed in:** 119f1de (Task 2 commit)

---

**Total deviations:** 3 auto-fixed (2 bugs, 1 blocking)
**Impact on plan:** All auto-fixes necessary for test correctness. No scope creep.

## Issues Encountered

None - plan executed as specified with expected test updates.

## User Setup Required

None - no external service configuration required.

## Verification Results

1. **Build:** PASS - `go build ./...` succeeds
2. **Tests:** PASS - All exporter tests pass
3. **Race detector:** PASS - No race conditions detected

## Next Phase Readiness

- Health check foundation complete, ready for Kubernetes/load balancer integration
- nbu_up metric available for alerting rules (e.g., `alert: NBUDown if nbu_up == 0 for 5m`)
- nbu_last_scrape_timestamp_seconds enables staleness alerts (e.g., `alert: NBUStale if time() - nbu_last_scrape_timestamp_seconds > 600`)
- Phase 6 may have additional plans for configuration reload, metrics endpoint enhancements

---
*Phase: 06-operational-features*
*Completed: 2026-01-23*
