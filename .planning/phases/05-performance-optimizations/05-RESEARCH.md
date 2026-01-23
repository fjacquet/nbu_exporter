# Phase 5: Performance Optimizations - Research

**Researched:** 2026-01-23
**Domain:** Go concurrency patterns, Prometheus exporter optimization, memory management
**Confidence:** HIGH

## Summary

This research investigates four performance optimizations for the NBU exporter: PERF-01 (pagination page size), PERF-02 (parallel collection), PERF-03 (streaming metrics), and PERF-04 (string split caching).

Key findings:

1. **PERF-04 is already complete** - Phase 3 Plan 03-03 eliminated all `strings.Split()` operations by using `Labels()` method directly. No further work needed.
2. **Job fetching is severely inefficient** - Current code fetches ONE job per API call (`page[limit]=1`), while NetBackup API supports up to 100 items per page. This is a 100x efficiency gain opportunity.
3. **Parallel collection is straightforward** - Storage and job metrics can be fetched concurrently using `errgroup.WithContext()`, reducing scrape time to max(storage_time, jobs_time).
4. **Memory optimization has diminishing returns** - With typed slices already in place, sync.Pool or streaming adds complexity. Pre-allocation is the simpler win.

**Primary recommendation:** Focus on PERF-01 (100x reduction in job API calls) and PERF-02 (parallel fetching), as these provide the highest impact with lowest complexity.

## Current State Analysis

### Job Fetching: Critical Performance Issue

**Current implementation (netbackup.go:124-127):**

```go
queryParams := map[string]string{
    QueryParamLimit:  "1",  // FETCHES ONE JOB PER API CALL!
    QueryParamOffset: strconv.Itoa(offset),
    // ...
}
```

**Impact analysis:**

- If 1000 jobs exist in time window: 1000 sequential API calls
- Each call has network latency (50-200ms)
- Total time: 50-200 seconds just for job metrics

**NetBackup API limits (verified from API docs):**

- Minimum page limit: 1
- Maximum page limit: 100
- Default page limit: 10

### Storage and Job Collection: Sequential

**Current implementation (prometheus.go:231-237):**

```go
func (c *NbuCollector) collectAllMetrics(ctx context.Context, span trace.Span) (...) {
    // Collect storage metrics FIRST
    storageMetrics, storageErr = c.collectStorageMetrics(ctx, span)

    // Collect job metrics AFTER storage completes
    jobsSize, jobsCount, jobsStatusCount, jobsErr = c.collectJobMetrics(ctx, span)

    return
}
```

**Impact:** Total scrape time = storage_time + jobs_time

### Metric Accumulation: Already Optimized

**Phase 3 changes (now in prometheus.go):**

```go
// Uses typed slices instead of string-keyed maps
func (c *NbuCollector) exposeStorageMetrics(ch chan<- prometheus.Metric, metrics []StorageMetricValue) {
    for _, m := range metrics {
        ch <- prometheus.MustNewConstMetric(
            c.nbuDiskSize,
            prometheus.GaugeValue,
            m.Value,
            m.Key.Labels()...,  // Direct use, no strings.Split()
        )
    }
}
```

**PERF-04 status:** COMPLETE - TD-03 in Phase 3 Plan 03-03 eliminated string parsing entirely.

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `golang.org/x/sync/errgroup` | v0.11.0+ | Parallel goroutine management | Official Go sub-repo, handles errors and context cancellation |
| `sync` (stdlib) | Go 1.25 | WaitGroup, Pool primitives | Standard library, no external dependency |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `sync.Pool` | stdlib | Slice reuse for memory pressure | Only if profiling shows GC pressure |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| errgroup | sync.WaitGroup + channels | errgroup handles errors and context cancellation automatically |
| sync.Pool | Pre-allocation | sync.Pool adds complexity, pre-allocation is simpler and often sufficient |
| Channel streaming | Slice accumulation | Streaming adds complexity, slice accumulation with pre-allocation is usually sufficient |

**No new dependencies required** - errgroup is likely already in go.mod via other dependencies.

## Architecture Patterns

### Pattern 1: Batched Pagination (PERF-01)

**What:** Fetch multiple items per API call instead of one
**When to use:** When API supports batch fetching (NetBackup max = 100)

**Current pattern (inefficient):**

```go
queryParams := map[string]string{
    QueryParamLimit:  "1",  // One item per request
    QueryParamOffset: strconv.Itoa(offset),
}
```

**Recommended pattern:**

```go
const jobPageLimit = "100"  // Maximum allowed by NetBackup API

queryParams := map[string]string{
    QueryParamLimit:  jobPageLimit,  // 100 items per request
    QueryParamOffset: strconv.Itoa(offset),
}
```

**Impact:** For 1000 jobs:

- Before: 1000 API calls
- After: 10 API calls
- Improvement: 100x fewer network round trips

### Pattern 2: Parallel Collection with errgroup (PERF-02)

**What:** Fetch storage and job metrics concurrently
**When to use:** Independent data sources that can be queried in parallel

**Source:** [errgroup package - Go Packages](https://pkg.go.dev/golang.org/x/sync/errgroup)

```go
import "golang.org/x/sync/errgroup"

func (c *NbuCollector) collectAllMetrics(ctx context.Context, span trace.Span) (
    storageMetrics []StorageMetricValue,
    jobsSize, jobsCount []JobMetricValue,
    jobsStatusCount []JobStatusMetricValue,
    storageErr, jobsErr error,
) {
    // Create errgroup with context for cancellation
    g, ctx := errgroup.WithContext(ctx)

    // Collect storage metrics in parallel
    g.Go(func() error {
        var err error
        storageMetrics, err = c.collectStorageMetrics(ctx, span)
        if err != nil {
            storageErr = err
        }
        return nil  // Don't cancel jobs if storage fails
    })

    // Collect job metrics in parallel
    g.Go(func() error {
        var err error
        jobsSize, jobsCount, jobsStatusCount, err = c.collectJobMetrics(ctx, span)
        if err != nil {
            jobsErr = err
        }
        return nil  // Don't cancel storage if jobs fail
    })

    // Wait for both to complete
    _ = g.Wait()

    return
}
```

**Impact:**

- Before: scrape_time = storage_time + jobs_time
- After: scrape_time = max(storage_time, jobs_time)
- Typical improvement: 30-50% faster scrapes

### Pattern 3: Pre-allocation for Memory Efficiency (PERF-03)

**What:** Pre-allocate slices based on expected size
**When to use:** When typical sizes are known

**Source:** [Optimizing Performance When Using Slices in Go](https://www.slingacademy.com/article/optimizing-performance-when-using-slices-in-go/)

```go
func FetchAllJobs(ctx context.Context, client *NbuClient, scrapingInterval string) (...) {
    // Pre-allocate based on expected job count
    // Typical: 50-500 unique metric keys
    sizeMap := make(map[JobMetricKey]float64, 100)
    countMap := make(map[JobMetricKey]float64, 100)
    statusMap := make(map[JobStatusKey]float64, 50)

    // ... pagination logic ...

    // Pre-allocate result slices
    jobsSize = make([]JobMetricValue, 0, len(sizeMap))
    jobsCount = make([]JobMetricValue, 0, len(countMap))
    statusCount = make([]JobStatusMetricValue, 0, len(statusMap))

    // ... populate ...
}
```

**Impact:** Reduces allocations during slice growth

### Pattern 4: Batch Processing in FetchJobDetails

**What:** Process multiple jobs per page instead of one
**When to use:** After increasing page limit

```go
func FetchJobDetails(
    ctx context.Context,
    client *NbuClient,
    jobsSize, jobsCount map[JobMetricKey]float64,
    jobsStatusCount map[JobStatusKey]float64,
    offset int,
    startTime time.Time,
) (int, error) {
    var jobs models.Jobs

    queryParams := map[string]string{
        QueryParamLimit:  "100",  // Fetch 100 jobs per page
        QueryParamOffset: strconv.Itoa(offset),
        QueryParamSort:   "jobId",
        QueryParamFilter: fmt.Sprintf("endTime%%20gt%%20%s", utils.ConvertTimeToNBUDate(startTime)),
    }

    // ... fetch data ...

    // Process ALL jobs in the response, not just one
    for _, job := range jobs.Data {
        jobKey := JobMetricKey{
            Action:     job.Attributes.JobType,
            PolicyType: job.Attributes.PolicyType,
            Status:     strconv.Itoa(job.Attributes.Status),
        }

        statusKey := JobStatusKey{
            Action: job.Attributes.JobType,
            Status: strconv.Itoa(job.Attributes.Status),
        }

        jobsCount[jobKey]++
        jobsStatusCount[statusKey]++
        jobsSize[jobKey] += float64(job.Attributes.KilobytesTransferred * bytesPerKilobyte)
    }

    // Return next offset
    if jobs.Meta.Pagination.Offset == jobs.Meta.Pagination.Last {
        return -1, nil
    }
    return jobs.Meta.Pagination.Next, nil
}
```

### Anti-Patterns to Avoid

1. **One-item pagination** - Fetching `page[limit]=1` when API supports 100
2. **Sequential independent fetches** - Waiting for storage before starting jobs
3. **Unbounded memory growth** - Not pre-allocating when sizes are predictable
4. **Over-engineering streaming** - Using channels when slices suffice

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Parallel error handling | Manual goroutine + channel | `errgroup.WithContext()` | Handles cancellation, errors, synchronization |
| Slice reuse pools | Custom pool with size tracking | `sync.Pool` or pre-allocation | Standard solution, well-tested |
| Metric key parsing | `strings.Split()` + reconstruction | `Labels()` method (already done) | Phase 3 solved this |

**Key insight:** The biggest wins are from reducing API calls (100x) and parallelizing fetches, not from complex memory optimizations.

## Common Pitfalls

### Pitfall 1: Error Handling in Parallel Collection

**What goes wrong:** Using errgroup's error propagation incorrectly causes one failure to cancel the other
**Why it happens:** `errgroup.WithContext()` cancels context on first error
**How to avoid:** Return `nil` from goroutines, track errors separately for graceful degradation
**Warning signs:** One API failure causes both metrics to be empty

```go
// WRONG: Cancels jobs if storage fails
g.Go(func() error {
    return FetchStorage(ctx, client)  // Error cancels context
})

// RIGHT: Continue with partial metrics
g.Go(func() error {
    _, storageErr = FetchStorage(ctx, client)
    return nil  // Don't cancel
})
```

### Pitfall 2: Offset Calculation with Larger Pages

**What goes wrong:** Offset increments by 1 instead of page size
**Why it happens:** Current code assumes limit=1, so offset=job_number
**How to avoid:** Use pagination metadata from API response

```go
// WRONG: Assumes offset increments by 1
offset += 1

// RIGHT: Use API's next offset
return jobs.Meta.Pagination.Next, nil
```

### Pitfall 3: Span Attributes Across Goroutines

**What goes wrong:** Concurrent writes to same span cause race conditions
**Why it happens:** OpenTelemetry spans are not thread-safe for attribute writes
**How to avoid:** Create separate child spans per goroutine, or use atomic counters

```go
// Create separate spans for each parallel operation
g.Go(func() error {
    ctx, storageSpan := tracer.Start(ctx, "fetch_storage")
    defer storageSpan.End()
    // ...
})
```

### Pitfall 4: Pre-allocation Size Mismatch

**What goes wrong:** Pre-allocated capacity too small causes reallocations; too large wastes memory
**Why it happens:** Hard-coded sizes don't match actual workload
**How to avoid:** Use reasonable defaults (100), document assumptions
**Warning signs:** Memory profiler shows frequent slice growth

## Code Examples

### Example 1: Parallel Collection with Graceful Degradation

```go
// Source: errgroup pattern + Prometheus node_exporter pattern

import "golang.org/x/sync/errgroup"

func (c *NbuCollector) collectAllMetrics(ctx context.Context, parentSpan trace.Span) (
    storageMetrics []StorageMetricValue,
    jobsSize, jobsCount []JobMetricValue,
    jobsStatusCount []JobStatusMetricValue,
    storageErr, jobsErr error,
) {
    g, gCtx := errgroup.WithContext(ctx)

    // Storage collection goroutine
    g.Go(func() error {
        storageMetrics, storageErr = c.collectStorageMetrics(gCtx, parentSpan)
        // Return nil to continue even if storage fails
        // Graceful degradation: jobs can still be collected
        return nil
    })

    // Jobs collection goroutine
    g.Go(func() error {
        jobsSize, jobsCount, jobsStatusCount, jobsErr = c.collectJobMetrics(gCtx, parentSpan)
        // Return nil to continue even if jobs fail
        return nil
    })

    // Wait for both to complete (or context cancellation)
    _ = g.Wait()

    return
}
```

### Example 2: Batched Job Fetching

```go
// Source: NetBackup API pagination pattern

const (
    jobPageLimit = "100"  // Maximum allowed by NetBackup API
)

func FetchJobDetails(
    ctx context.Context,
    client *NbuClient,
    jobsSize, jobsCount map[JobMetricKey]float64,
    jobsStatusCount map[JobStatusKey]float64,
    offset int,
    startTime time.Time,
) (int, error) {
    ctx, span := client.tracing.StartSpan(ctx, "netbackup.fetch_job_page", trace.SpanKindClient)
    defer span.End()

    var jobs models.Jobs

    queryParams := map[string]string{
        QueryParamLimit:  jobPageLimit,  // 100 items per page
        QueryParamOffset: strconv.Itoa(offset),
        QueryParamSort:   "jobId",
        QueryParamFilter: fmt.Sprintf("endTime%%20gt%%20%s", utils.ConvertTimeToNBUDate(startTime)),
    }

    url := client.cfg.BuildURL(jobsPath, queryParams)

    if err := client.FetchData(ctx, url, &jobs); err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return -1, fmt.Errorf("failed to fetch job details at offset %d: %w", offset, err)
    }

    // Process ALL jobs in the page
    for _, job := range jobs.Data {
        jobKey := JobMetricKey{
            Action:     job.Attributes.JobType,
            PolicyType: job.Attributes.PolicyType,
            Status:     strconv.Itoa(job.Attributes.Status),
        }

        statusKey := JobStatusKey{
            Action: job.Attributes.JobType,
            Status: strconv.Itoa(job.Attributes.Status),
        }

        jobsCount[jobKey]++
        jobsStatusCount[statusKey]++
        jobsSize[jobKey] += float64(job.Attributes.KilobytesTransferred * bytesPerKilobyte)
    }

    // Record batch size in span
    span.SetAttributes(
        attribute.Int("jobs.batch_size", len(jobs.Data)),
        attribute.Int("jobs.offset", offset),
    )

    // Check for end of pagination
    if len(jobs.Data) == 0 || jobs.Meta.Pagination.Offset == jobs.Meta.Pagination.Last {
        return -1, nil
    }

    return jobs.Meta.Pagination.Next, nil
}
```

### Example 3: Pre-allocated Map and Slice Creation

```go
// Source: Go slice optimization patterns

func FetchAllJobs(
    ctx context.Context,
    client *NbuClient,
    scrapingInterval string,
) (jobsSize []JobMetricValue, jobsCount []JobMetricValue, statusCount []JobStatusMetricValue, err error) {
    // Pre-allocate maps with reasonable capacity
    // Typical environments: 20-100 unique job type/policy/status combinations
    const expectedUniqueKeys = 100
    const expectedStatusKeys = 50

    sizeMap := make(map[JobMetricKey]float64, expectedUniqueKeys)
    countMap := make(map[JobMetricKey]float64, expectedUniqueKeys)
    statusMap := make(map[JobStatusKey]float64, expectedStatusKeys)

    // ... pagination logic ...

    // Pre-allocate result slices based on actual map sizes
    jobsSize = make([]JobMetricValue, 0, len(sizeMap))
    jobsCount = make([]JobMetricValue, 0, len(countMap))
    statusCount = make([]JobStatusMetricValue, 0, len(statusMap))

    // Convert maps to slices (no reallocation needed)
    for key, value := range sizeMap {
        jobsSize = append(jobsSize, JobMetricValue{Key: key, Value: value})
    }
    // ... same for other slices

    return
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| String-keyed maps + Split | Typed structs + Labels() | Phase 3 (03-03) | PERF-04 complete |
| Sequential collection | Parallel with errgroup | Common since Go 1.7 | 30-50% faster scrapes |
| One-item pagination | Batch pagination | API supports 100 | 100x fewer API calls |

**Deprecated/outdated:**

- `page[limit]=1` approach: Was likely a simplification that became a bottleneck

## Dependencies & Wave Structure

### Wave Analysis

```
Wave 1 (Independent, highest impact):
  - PERF-01: Increase job page size to 100 (100x improvement)
  - PERF-02: Parallel storage/job collection (30-50% improvement)

Wave 2 (After Wave 1):
  - PERF-03: Pre-allocation optimizations (minor improvement)

Wave 3 (Already complete):
  - PERF-04: String split elimination (done in Phase 3)
```

### Recommended Implementation Order

1. **PERF-01 first** - Change `page[limit]` from 1 to 100, update loop to process batch
2. **PERF-02 second** - Add errgroup for parallel collection
3. **PERF-03 third** - Add pre-allocation (optional, based on profiling)
4. **PERF-04** - Already done, no work needed

### Dependency Graph

```
PERF-01 (Batch Pagination)
    |
    v
PERF-02 (Parallel Collection) -- independent, can be done in parallel with PERF-01
    |
    v
PERF-03 (Pre-allocation) -- refinement, optional

PERF-04 (String Split) -- DONE in Phase 3
```

## Risks & Mitigations

### Risk 1: Batch Processing Changes Offset Logic

**Description:** Current offset logic assumes limit=1, changing to 100 requires different offset handling
**Probability:** HIGH (certain to need changes)
**Impact:** MEDIUM (incorrect offset = missed or duplicate jobs)
**Mitigation:** Use API's `Meta.Pagination.Next` directly, add integration tests

### Risk 2: Parallel Collection Race Conditions

**Description:** Span attributes or logging could have race conditions
**Probability:** LOW (well-understood pattern)
**Impact:** MEDIUM (sporadic failures in tests)
**Mitigation:** Create separate spans per goroutine, use atomic counters

### Risk 3: Large Page Size Increases Memory Per Request

**Description:** Fetching 100 jobs vs 1 job uses more memory per request
**Probability:** LOW (jobs are small JSON objects)
**Impact:** LOW (100 jobs ~50KB, well within reason)
**Mitigation:** Monitor memory during testing, reduce page size if needed

### Risk 4: API Behavior Differences with Larger Pages

**Description:** NetBackup API might behave differently with page[limit]=100
**Probability:** LOW (documented API behavior)
**Impact:** MEDIUM (would require reverting)
**Mitigation:** Test against real NetBackup server before release

## Open Questions

### Question 1: Optimal Page Size

**What we know:** NetBackup max is 100, current is 1
**What's unclear:** Is 100 always optimal, or should it be configurable?
**Recommendation:** Use 100 (maximum), no configuration needed initially

### Question 2: Timeout Adjustment for Larger Batches

**What we know:** Current 2-minute timeout was set for many small requests
**What's unclear:** Should timeout increase with batch processing?
**Recommendation:** Keep 2-minute timeout initially, monitor in production

### Question 3: Memory Profiling Baseline

**What we know:** Pre-allocation and sync.Pool can help with memory pressure
**What's unclear:** Is there actual memory pressure in production?
**Recommendation:** Profile before implementing PERF-03, may not be needed

## Sources

### Primary (HIGH confidence)

- [errgroup package - Go Packages](https://pkg.go.dev/golang.org/x/sync/errgroup) - Official documentation for parallel execution
- [Writing exporters | Prometheus](https://prometheus.io/docs/instrumenting/writing_exporters/) - Exporter best practices
- [prometheus/node_exporter](https://github.com/prometheus/node_exporter) - Reference implementation for parallel collection
- NetBackup API page[limit] validation: `testdata/api-10.5/error-responses.json` shows "Value must be between 1 and 100"

### Secondary (MEDIUM confidence)

- [How to Use errgroup for Parallel Operations in Go](https://oneuptime.com/blog/post/2026-01-07-go-errgroup/view) - errgroup patterns
- [Efficient Buffering - Go Optimization Guide](https://goperf.dev/01-common-patterns/buffered-io/) - Memory optimization patterns
- [Go sync.Pool and the Mechanics Behind It](https://victoriametrics.com/blog/go-sync-pool/) - sync.Pool usage patterns

### Codebase Analysis (HIGH confidence)

- Phase 3 Plan 03-03: `.planning/phases/03-architecture-improvements/03-03-PLAN.md` - TD-03 implementation
- Current netbackup.go line 126: `QueryParamLimit: "1"` - Confirmed single-item pagination
- Current prometheus.go lines 231-237: Sequential collection confirmed

## Metadata

**Confidence breakdown:**

- PERF-01 (Batch pagination): HIGH - API limits verified, pattern straightforward
- PERF-02 (Parallel collection): HIGH - Standard Go pattern, well-documented
- PERF-03 (Memory optimization): MEDIUM - May not be needed, profile first
- PERF-04 (String split): HIGH - Verified complete in Phase 3

**Research date:** 2026-01-23
**Valid until:** 2026-02-23 (30 days - stable patterns, no fast-moving dependencies)
