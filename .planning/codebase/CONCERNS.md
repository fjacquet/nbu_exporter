# Codebase Concerns

**Analysis Date:** 2026-01-22

## Tech Debt

**Configuration Mutation During Version Detection:**

- Issue: `performVersionDetectionIfNeeded()` in `internal/exporter/client.go` (lines 153-154) modifies the configuration object in place by setting `cfg.NbuServer.APIVersion` and `client.cfg.NbuServer.APIVersion`. Similarly, `setTemporaryVersion()` in `internal/exporter/version_detector.go` (lines 194-197) modifies and restores the version. This pattern is fragile if errors occur during detection or if multiple collectors share config instances.
- Files: `internal/exporter/client.go`, `internal/exporter/version_detector.go`, `internal/exporter/prometheus.go` (lines 76-81)
- Impact: Configuration state becomes unpredictable if version detection fails partially. Multiple concurrent operations could see inconsistent API versions.
- Fix approach: Pass detected version back without mutating the original config, or clone config before modification. Consider immutable config pattern.

**Global State via OpenTelemetry:**

- Issue: Code relies on `otel.SetTextMapPropagator()` and `otel.GetTracerProvider()` (main.go line 117-119, client.go line 84, version_detector.go line 64) which are global singletons. Tests may interfere with each other if OTel is not properly isolated.
- Files: `main.go`, `internal/exporter/client.go`, `internal/exporter/version_detector.go`, `internal/exporter/prometheus.go`
- Impact: Test pollution when OpenTelemetry is initialized; concurrent instances share global propagator state.
- Fix approach: Properly reset global OTel state between test cases. Consider dependency injection for tracer instead of relying on global provider.

**Metrics with Pipe-Delimited Keys:**

- Issue: Storage and job metrics use pipe-delimited compound keys (e.g., "pool1|AdvancedDisk|free") that are split at exposition time using `strings.Split(key, "|")` in `internal/exporter/prometheus.go` (lines 312, 325, 338, 351). If a storage name or policy type contains a pipe character, metric labels will be incorrect.
- Files: `internal/exporter/prometheus.go` (lines 310-359), `internal/exporter/netbackup.go` (lines 96-97, 189-198)
- Impact: Metrics silently fail to expose correctly if source data contains pipe characters. No validation or escaping.
- Fix approach: Use structured metric keys instead of string concatenation, or implement proper escaping/unescaping with validation.

**Main Package Has Zero Test Coverage:**

- Issue: `main.go` has 0% coverage (confirmed by test run output). Critical entry point logic including server startup, graceful shutdown, and signal handling is untested.
- Files: `main.go` (all 342 lines)
- Impact: Regressions in startup sequence, graceful shutdown, or signal handling could go unnoticed. Changes to the HTTP server configuration or middleware are not validated.
- Fix approach: Create integration tests that start the server, verify endpoints respond, and test graceful shutdown with signals.

**Incomplete Resource Cleanup in NbuClient:**

- Issue: `Close()` method in `internal/exporter/client.go` (lines 469-479) calls `CloseIdleConnections()` but doesn't properly drain pending requests or wait for active connections to complete. The client is set to nil immediately, making follow-up operations unsafe.
- Files: `internal/exporter/client.go` (lines 464-479), `internal/exporter/interface.go` (lines 45-49)
- Impact: Potential for connection leaks if the client has pending requests when closed. No guarantee all connections are actually released.
- Fix approach: Implement proper drain logic with context deadline for pending requests before closing idle connections.

**Fatal Log in Async Goroutine:**

- Issue: `main.go` line 159 calls `log.Fatalf("HTTP server error: %v", err)` inside a goroutine launched in `Start()`. This can exit the process unexpectedly without graceful shutdown, losing telemetry data.
- Files: `main.go` (line 159)
- Impact: Sudden process termination without flushing OpenTelemetry spans or gracefully closing connections.
- Fix approach: Send errors back through a channel or error callback instead of fatally exiting from goroutines. Let the main function handle shutdown.

---

## Known Bugs

**Version Detection State Not Restored on Context Cancellation:**

- Symptoms: If the context is cancelled during version detection (e.g., timeout), the temporary version set in `setTemporaryVersion()` should be restored, but if a panic occurs, restoration via `defer` might fail in some edge cases.
- Files: `internal/exporter/version_detector.go` (lines 183-191, 194-204)
- Trigger: Calling `tryVersion()` with a context that cancels mid-request or with a network panic
- Workaround: The code uses `defer` for restoration, which should handle most cases, but recovery from panic is not guaranteed.

---

## Security Considerations

**API Key Stored in Plain Memory:**

- Risk: The API key is stored in `models.Config.NbuServer.APIKey` as a plain string in memory. While there's masking for log output (line 316: `MaskAPIKey()`), the key itself is never encrypted or protected in memory.
- Files: `internal/models/Config.go` (lines 47, 310-315), `main.go` (line 316)
- Current mitigation: Masking in logs; no logging of raw key except in debug mode (line 316: `if debug`)
- Recommendations: Consider using Go's `go-secure-env` or similar for sensitive config values. Ensure API key comes from environment variables or secure vaults, never from version control. Document that config files with API keys must have restricted permissions.

**TLS Certificate Verification Disabled in Production:**

- Risk: `InsecureSkipVerify` in `models.Config` (line 50) allows disabling certificate verification. While there's a warning log in `client.go` line 73, this setting makes the connection vulnerable to MITM attacks.
- Files: `internal/models/Config.go` (line 50), `internal/exporter/client.go` (lines 72-74)
- Current mitigation: Warning logged when set to true
- Recommendations: Default to `false` and require explicit opt-in. Add documentation warning about production risks. Consider adding certificate pinning as an alternative for self-signed certs.

**No Rate Limiting or Backoff Beyond Version Detection:**

- Risk: The collector fetches storage and job data on every Prometheus scrape without checking for rate limits. If NetBackup API rate limits are hit, the exporter will fail with HTTP 429 but has no retry logic or exponential backoff (except during initial version detection).
- Files: `internal/exporter/netbackup.go` (lines 64-115, 316-389)
- Current mitigation: Version detection has retry logic with exponential backoff (DefaultRetryConfig in `version_detector.go`)
- Recommendations: Implement rate limit handling for metric collection; cache storage metrics if possible; add 429 response handling with backoff.

---

## Performance Bottlenecks

**Single Pagination Pass with Hardcoded Page Size:**

- Problem: `FetchAllJobs()` in `internal/exporter/netbackup.go` paginates through jobs one at a time (limit=1, line 157) to maintain job-level granularity. For large job counts, this requires many sequential API calls.
- Files: `internal/exporter/netbackup.go` (lines 316-389, specifically line 157: `"1"`)
- Cause: Design choice to process one job per request for span tracking; no pagination optimization
- Improvement path: Increase page size to 100 for initial fetch, then aggregate metrics. Add job filtering server-side if available. Consider async pagination if NetBackup API supports it.

**Two Separate API Calls Per Scrape:**

- Problem: `collectAllMetrics()` in `internal/exporter/prometheus.go` (lines 213-229) makes separate calls to fetch storage metrics and job metrics sequentially, both with 2-minute timeout.
- Files: `internal/exporter/prometheus.go` (lines 223-226)
- Cause: Logical separation of concerns, but no parallelization
- Improvement path: Fetch storage and jobs in parallel using goroutines with shared context to reduce total scrape time. Current total time = storage_time + jobs_time; could reduce to max(storage_time, jobs_time).

**Memory Accumulation in Maps During Pagination:**

- Problem: `jobsSize`, `jobsCount`, `jobsStatusCount` maps accumulate all jobs in memory across pagination without bounds checking. For large NetBackup environments with millions of jobs, this could exhaust memory.
- Files: `internal/exporter/netbackup.go` (lines 316-389), `internal/exporter/prometheus.go` (lines 213-229)
- Cause: No streaming or aggregation limits; all metrics held until scrape completes
- Improvement path: Implement metric streaming directly to Prometheus channel instead of accumulating in maps. Add memory limits with warning if exceeded.

**String Splitting on Every Metric Exposition:**

- Problem: `exposeStorageMetrics()`, `exposeJobSizeMetrics()`, etc. in `internal/exporter/prometheus.go` split the pipe-delimited key on every metric exposition (lines 312, 325, 338, 351).
- Files: `internal/exporter/prometheus.go` (lines 310-359)
- Cause: Metrics are stored with compound keys then split on every read instead of being pre-structured
- Improvement path: Store metrics in structured format (e.g., structs with labeled fields) instead of strings, or cache the split results.

---

## Fragile Areas

**APIVersionDetector Relies on Shared Config Reference:**

- Files: `internal/exporter/version_detector.go` (lines 45-51, 194-204)
- Why fragile: The detector holds a pointer to `*models.Config` and modifies it directly. If multiple goroutines call `DetectVersion()` concurrently with the same config, race conditions occur.
- Safe modification: Pass version detection results back without mutating input config. Use read-only config views for detection.
- Test coverage: Extensive testing in `version_detector_test.go`, but no concurrent access tests.

**HTTP Client Connection Pool Not Explicitly Managed:**

- Files: `internal/exporter/client.go` (lines 56-94, 469-479)
- Why fragile: The resty client maintains an internal connection pool. Calling `Close()` only closes idle connections, not active ones. If many collectors are created and destroyed, connection leaks may occur.
- Safe modification: Implement proper connection drain with deadline. Document the cleanup requirements.
- Test coverage: `client_test.go` tests individual methods but not lifecycle stress tests.

**BuildURL Ignores URL Parsing Errors:**

- Files: `internal/models/Config.go` (lines 331-341)
- Why fragile: Line 332 calls `url.Parse()` and ignores the error: `u, _ := url.Parse(...)`. If an invalid base URL somehow gets constructed (e.g., invalid characters), the error is silently lost.
- Safe modification: Validate `GetNBUBaseURL()` during config validation, not at call time.
- Test coverage: No test for invalid URL handling.

**Tracer Nil-Checks Scattered Throughout:**

- Files: Multiple files including `client.go`, `prometheus.go`, `netbackup.go` with patterns like `if c.tracer == nil { return ctx, nil }`
- Why fragile: The nil-safe tracer pattern is repeated ~15 times. If one is missed, a nil dereference panic occurs.
- Safe modification: Create a wrapper method `createSpanIfEnabled()` that handles nil checks centrally.
- Test coverage: Tests exist but nil tracer scenarios are not exhaustively tested.

---

## Scaling Limits

**Single Collector Instance Per Registry:**

- Current capacity: One `NbuCollector` registered per `prometheus.Registry`. Multiple collectors would duplicate metrics.
- Limit: Cannot scale to multiple NetBackup servers or shards without creating separate exporter instances.
- Scaling path: Design exporter as a multi-collector framework where each collector targets a different NetBackup instance. Implement dynamic collector registration/deregistration.

**Pagination Offset Limited by Sequential Processing:**

- Current capacity: Sequential pagination through all jobs within time window. For very large job counts (>100k in 5-minute window), scrape times could exceed Prometheus default timeout (10s).
- Limit: Scrape timeout if job volume exceeds ~10k items.
- Scaling path: Implement time-based sharding (e.g., fetch jobs from different time buckets in parallel). Add server-side filtering if NetBackup API supports it.

**Memory Growth Linear with Scrape Window:**

- Current capacity: All metrics accumulated in memory maps. With 100k jobs, memory usage could exceed available heap.
- Limit: Likely OOM error if job count is very large and scraping interval is wide.
- Scaling path: Stream metrics directly to channel instead of buffering. Implement incremental aggregation.

---

## Test Coverage Gaps

**Main Package Entry Point (0% coverage):**

- What's not tested: Server startup sequence, CLI flag parsing, graceful shutdown with signals, HTTP endpoint registration, OpenTelemetry initialization from main.
- Files: `main.go` (all 342 lines)
- Risk: Regressions in core startup/shutdown logic go unnoticed. Changes to HTTP handler registration or flag definitions are not validated.
- Priority: **High** - This is the entry point; it should have integration tests.

**TestUtil Package (51.9% coverage):**

- What's not tested: Some helper functions in `internal/testutil/helpers.go` are likely not fully exercised (exact gaps unknown without detailed coverage report).
- Files: `internal/testutil/helpers.go`
- Risk: Test helpers themselves may have bugs that affect test reliability.
- Priority: **Medium** - While helpers are important, functional tests depend on them more than they depend on helper coverage.

**Telemetry Package (78.3% coverage):**

- What's not tested: Likely gaps in OpenTelemetry lifecycle edge cases, error paths, and concurrent initialization.
- Files: `internal/telemetry/manager.go`, `internal/telemetry/manager_test.go`
- Risk: OTel failures might not be caught; graceful degradation not fully validated.
- Priority: **Medium** - Telemetry is optional; failures here don't break core functionality.

**Client Error Handling Edge Cases:**

- What's not tested: Some HTTP error scenarios, network timeout behaviors, and trace context injection failures.
- Files: `internal/exporter/client.go`
- Risk: Unusual HTTP responses or network conditions might not be handled gracefully.
- Priority: **Medium** - Main paths are tested; edge cases are rarer but could affect reliability.

**Concurrent Collector Access:**

- What's not tested: Multiple goroutines concurrently calling `Collect()` or concurrent metrics collection during shutdown.
- Files: `internal/exporter/prometheus.go`
- Risk: Race conditions or deadlocks during high-concurrency scenarios or shutdown.
- Priority: **Low to Medium** - Prometheus typically serializes scrapes, but this should be verified.

---

## Dependencies at Risk

**OpenTelemetry SDK (go.opentelemetry.io/otel):**

- Risk: Core OTel functionality is essential for tracing. If OTel SDK has breaking changes, compatibility issues could arise.
- Current version: 1.39.0
- Impact: Spans may not be created or exported correctly if SDK versions mismatch between client and exporter.
- Migration plan: Pin specific OTel version in go.mod (currently done). Monitor release notes for breaking changes. Have a fallback telemetry strategy if OTel fails.

**Prometheus Client Library (github.com/prometheus/client_golang):**

- Risk: Metrics exposition format or registry behavior could change; Prometheus library is core to exporter functionality.
- Current version: 1.23.2
- Impact: Metrics might not be scraped correctly or format compatibility could break.
- Migration plan: Use strict version pinning. Test against multiple Prometheus versions (9.x, 10.x).

**Resty HTTP Client (github.com/go-resty/resty/v2):**

- Risk: Resty is used for all NetBackup API calls. A bug or deprecation could impact reliability.
- Current version: 2.17.1
- Impact: HTTP timeouts, connection pooling, or TLS handling could break.
- Migration plan: Consider switching to stdlib `net/http` with custom wrappers if resty becomes unmaintained. Resty is well-maintained but adds a dependency.

---

## Missing Critical Features

**No Caching of Expensive Metrics:**

- Problem: Storage metrics are fetched on every Prometheus scrape even though they rarely change. This wastes NetBackup API quota.
- Blocks: Efficient metric collection for high-scrape-frequency setups.
- Impact: High API load on NetBackup server; inefficient resource usage.

**No Health Check for NetBackup Connectivity:**

- Problem: The `/health` endpoint (main.go line 146) just returns "OK" without checking if NetBackup is actually reachable.
- Blocks: Operators can't determine if metrics are stale or collection is failing.
- Impact: Hidden failures; metrics continue to be exposed as current even if NetBackup is unavailable.

**No Metric Staleness Tracking:**

- Problem: If a job or storage unit is deleted in NetBackup, the exporter still exposes the old metric without indicating it's stale.
- Blocks: Prometheus can't distinguish between "no activity" and "source deleted."
- Impact: Misleading metrics; deleted storage units still show in dashboards.

**No Dynamic Configuration Reload:**

- Problem: Configuration changes require restarting the exporter.
- Blocks: Updating NetBackup server address or credentials without downtime.
- Impact: Operational inconvenience; potential for unplanned downtime.

---

_Concerns audit: 2026-01-22_
