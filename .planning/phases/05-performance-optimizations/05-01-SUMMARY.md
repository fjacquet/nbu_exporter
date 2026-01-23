---
phase: 05-performance-optimizations
plan: 01
subsystem: api-client
tags: [pagination, performance, netbackup, batch-processing]

dependency_graph:
  requires:
    - phase-03 (architecture-improvements)
    - phase-04 (test-coverage)
  provides:
    - batched-job-pagination
    - 100x-api-call-reduction
  affects:
    - future-performance-plans
    - prometheus-scrape-latency

tech_stack:
  added: []
  patterns:
    - batch-pagination
    - range-iteration

key_files:
  created:
    - internal/exporter/netbackup_test.go
  modified:
    - internal/exporter/netbackup.go
    - internal/exporter/api_compatibility_test.go
    - internal/exporter/integration_test.go

decisions:
  - name: jobPageLimit constant
    choice: Separate constant for jobs (100) from storage pagination
    rationale: Jobs and storage may have different optimal page sizes
  - name: Loop over all jobs in batch
    choice: Use range loop instead of single-item access
    rationale: Process all jobs in API response efficiently
  - name: Remove PageNumber attribute
    choice: Remove AttrNetBackupPageNumber from span attributes
    rationale: Meaningless with batch pagination; offset suffices

metrics:
  duration: 10 minutes
  completed: 2026-01-23
---

# Phase 5 Plan 1: Batched Job Pagination Summary

**One-liner:** Job fetching now uses page size 100 instead of 1, reducing API calls by ~100x for large job sets.

## What Was Built

Implemented batch job pagination to dramatically reduce API calls during Prometheus scrapes:

1. **Batch Pagination Constants**
   - Added `jobPageLimit = "100"` constant for jobs endpoint
   - Separate from existing `pageLimit` for storage (allows independent tuning)

2. **FetchJobDetails Batch Processing**
   - Changed `QueryParamLimit` from `"1"` to `jobPageLimit`
   - Added range loop to process all jobs in response: `for _, job := range jobs.Data {}`
   - Updated span attributes to use `AttrNetBackupJobsInPage` (actual count)
   - Removed `AttrNetBackupPageNumber` (meaningless with batches)

3. **Comprehensive Batch Tests**
   - `TestFetchJobDetails_BatchProcessing`: Verifies all jobs in batch are processed
   - `TestFetchJobDetails_BatchPagination`: Tests 150 jobs across 2 pages (100 + 50)
   - `TestFetchJobDetails_EmptyBatch`: Verifies empty response returns -1
   - `TestFetchJobDetails_MixedJobTypes`: Verifies different job types counted separately

4. **Updated Test Helpers**
   - Fixed `isPaginatedJobsRequest` to handle any page limit
   - Updated `createPaginatedJobsResponse` for batch responses
   - Added `setBatchPaginationMetadata` for correct pagination metadata

## Performance Impact

| Metric | Before | After | Improvement |
|--------|--------|-------|-------------|
| API calls for 1000 jobs | 1000 | ~10 | 100x reduction |
| API calls for 100 jobs | 100 | 1 | 100x reduction |
| API calls for 50 jobs | 50 | 1 | 50x reduction |

## Key Code Changes

### netbackup.go

```go
const (
    jobPageLimit = "100"  // Maximum allowed by NetBackup API for jobs
)

// In FetchJobDetails:
QueryParamLimit:  jobPageLimit,  // Was "1", now "100"

// Process ALL jobs in the batch
for _, job := range jobs.Data {
    jobKey := JobMetricKey{...}
    jobsCount[jobKey]++
    // ...
}
```

## Commits

| Hash | Type | Description |
|------|------|-------------|
| d0ca24d | feat | Implement batch job pagination (100 jobs per page) |
| a4d96da | test | Add batch processing tests for FetchJobDetails |

## Verification Results

- [x] QueryParamLimit uses "100" for job fetching (grep confirmed)
- [x] FetchJobDetails loops over jobs.Data (not just Data[0])
- [x] Existing tests pass (no regressions)
- [x] New batch tests pass (4 tests)
- [x] Full test suite passes with race detector
- [x] Build succeeds

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Updated test helpers for batch pagination**

- **Found during:** Task 1 verification
- **Issue:** Test mock server checked for `page[limit]=1` specifically
- **Fix:** Updated `isPaginatedJobsRequest` and pagination helpers to handle batch responses
- **Files modified:** api_compatibility_test.go, integration_test.go

## Files Changed

| File | Changes |
|------|---------|
| internal/exporter/netbackup.go | Added jobPageLimit constant, batch processing loop, updated comments |
| internal/exporter/netbackup_test.go | Created with 4 batch processing tests |
| internal/exporter/api_compatibility_test.go | Fixed pagination helpers for batch responses |
| internal/exporter/integration_test.go | Updated TestPaginationHandling for batch behavior |

## Next Phase Readiness

Plan 05-01 complete. Ready for:

- Plan 05-02: Connection pooling optimization (if exists)
- Plan 05-03: Concurrent fetching (if exists)

No blockers or concerns for subsequent plans.
