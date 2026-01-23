# GSD State

## Project Reference

See: .planning/PROJECT.md (updated 2026-01-23)

**Core value:** Improve code reliability and maintainability by fixing identified concerns
**Current focus:** Milestone v1.1 — Codebase Improvements

## Current Position

Phase: 3 of 6 (Architecture Improvements)
Plan: 2 of 5 complete
Status: In progress
Last activity: 2026-01-23 — Completed plan 03-02 (TracerProvider Injection)

## Progress

- [x] Codebase mapped (7 documents in .planning/codebase/)
- [x] PROJECT.md created with requirements from CONCERNS.md
- [x] Milestone v1.1 defined
- [x] REQUIREMENTS.md created with 27 requirements
- [x] Roadmap created with 6 phases (100% requirement coverage)
- [x] Phase 1 planning complete (4 plans)
- [x] Phase 1 execution (4 of 4 plans complete: 01-01, 01-02, 01-03, 01-04)
- [x] Phase 1 verification passed (15/16 must-haves, 93.75%)
- [x] Phase 2 research complete (02-RESEARCH.md)
- [x] Phase 2 planning complete (2 plans: 02-01, 02-02)
- [x] Phase 2 execution complete (2 of 2 plans: 02-01, 02-02)
- [x] Phase 3 research complete (03-RESEARCH.md)
- [x] Phase 3 planning complete (5 plans: 03-01, 03-02, 03-03, 03-04, 03-05)
- [ ] Phase 3 execution (2 of 5 plans complete: 03-01, 03-02)

## Accumulated Context

**Decisions:**

- Address concerns in priority order: Bugs -> Security -> Tech Debt -> Performance -> Features
- Maintain backwards compatibility for Prometheus metrics
- Require test coverage for all changes
- Phase structure: Critical Fixes -> Security -> Architecture -> Tests -> Performance -> Features
- Phase 1 plans are independent (Wave 1) - can execute in parallel
- (01-01) APIVersionDetector stores only immutable values (baseURL, apiKey) instead of config reference
- (01-01) Version detection builds headers inline with test version instead of relying on client.getHeaders()
- (01-01) Config mutation happens only in performVersionDetectionIfNeeded after successful detection
- (01-01) Detectors return results without side effects - caller applies changes
- (01-02) Keep BuildURL() signature unchanged for backward compatibility
- (01-02) Validate URL during Config.Validate() instead of at BuildURL() time
- (01-02) Document BuildURL() assumption of validated config
- (01-03) Use atomic int32 for activeReqs counter to minimize lock contention
- (01-03) Store local channel reference before releasing lock to prevent data race
- (01-03) Default 30-second timeout for Close() balances graceful shutdown vs hang prevention
- (01-03) CloseWithContext() provides custom timeout control for advanced use cases
- (01-04) Use buffered error channel (capacity 1) for async server errors to prevent goroutine leak
- (01-04) Server errors trigger graceful shutdown via Shutdown() rather than abrupt exit
- (01-04) Error channel pattern: goroutine errors communicated via buffered channel instead of log.Fatalf
- (02-01) TLS insecure mode requires NBU_INSECURE_MODE=true environment variable for explicit opt-in
- (02-01) TLS 1.2 is minimum supported version (industry standard, blocks older protocols)
- (02-01) Security warnings log at Error level (not Warn) for better visibility
- (02-01) API key protection verified via comprehensive audit - no code changes needed
- (03-01) Use noop.NewTracerProvider() as default instead of nil tracer for zero-overhead disabled tracing
- (03-01) TracerWrapper guarantees valid span return (never nil) to eliminate nil-checks
- (03-01) Keep deprecated createSpan for backward compatibility during migration
- (03-02) Options pattern for TracerProvider injection eliminates global otel.GetTracerProvider() calls
- (03-02) TracerProvider flows explicitly: telemetry.Manager → main.go → NbuCollector → NbuClient
- (03-02) Components work correctly without TracerProvider (noop default via TracerWrapper)
- (03-02) Separate SDK trace import (sdktrace) from API trace import in telemetry.Manager
- (03-04) ImmutableConfig extracts values after validation and version detection complete
- (03-04) All ImmutableConfig fields private with accessor methods returning copies/values
- (03-04) Incremental adoption allows gradual migration of existing components
- (03-04) Full component migration (NbuClient, NbuCollector) deferred to future phase

**Phase 1 Plans:**

| Plan  | Focus                          | Requirements    | Files Modified                                           |
| ----- | ------------------------------ | --------------- | -------------------------------------------------------- |
| 01-01 | Version Detection Immutability | BUG-01, FRAG-01 | version_detector.go, client.go, version_detector_test.go |
| 01-02 | URL Validation                 | FRAG-03         | Config.go, Config_test.go                                |
| 01-03 | Resource Cleanup               | TD-05           | client.go, client_test.go, interface.go                  |
| 01-04 | Error Channel Pattern          | TD-06           | main.go                                                  |

**Phase 2 Plans:**

| Plan  | Focus                              | Requirements   | Files Modified                                       |
| ----- | ---------------------------------- | -------------- | ---------------------------------------------------- |
| 02-01 | TLS Enforcement & API Key Security | SEC-01, SEC-02 | Config.go, Config_test.go, client.go, client_test.go |
| 02-02 | Rate Limiting & Retry with Backoff | SEC-03         | client.go, client_test.go                            |

**Phase 3 Plans:**

| Plan  | Focus                            | Requirements | Files Modified                                             |
| ----- | -------------------------------- | ------------ | ---------------------------------------------------------- |
| 03-01 | TracerWrapper with Noop Default  | TD-02        | tracing.go, tracing_test.go                                |
| 03-02 | TracerProvider Injection         | TD-02        | client.go, prometheus.go, netbackup.go, manager.go, main.go |
| 03-03 | Collector Responsibility Split   | TD-03        | prometheus.go, collector.go                                |
| 03-04 | ImmutableConfig Type             | TD-01        | immutable.go, immutable_test.go                            |
| 03-05 | Migrate to ImmutableConfig       | TD-01        | client.go, prometheus.go, main.go                          |

**Blockers:** None

## Session Notes

**2026-01-23 (Plan 03-02 Execution):** Completed plan 03-02 (TracerProvider Injection). Implemented options pattern for TracerProvider injection in NbuClient (WithTracerProvider) and NbuCollector (WithCollectorTracerProvider). Eliminated all global otel.GetTracerProvider() and otel.Tracer() calls from component constructors. TracerProvider flows explicitly: telemetry.Manager.TracerProvider() → main.go → NbuCollector → NbuClient. Updated telemetry.Manager to separate SDK/API trace imports and expose TracerProvider() method. All components work correctly without TracerProvider (noop default via TracerWrapper). Updated all tests to use new options pattern and TracerWrapper behavior. Four atomic commits: (1) NbuClient options, (2) NbuCollector options, (3) main.go injection, (4) test updates. All tests pass with race detector. Fixes TD-02 (eliminate global OpenTelemetry state). No deviations from plan. Duration: 10 minutes.

**2026-01-23 (Plan 03-04 Execution):** Completed plan 03-04 (ImmutableConfig Type). Created ImmutableConfig type with private fields and accessor methods, enabling thread-safe runtime configuration through snapshot pattern. NewImmutableConfig constructor extracts values from validated Config after validation and version detection complete. All fields private with accessor methods returning copies/values (never references). Added MaskedAPIKey() for safe logging. Comprehensive tests verify immutability guarantees and snapshot behavior (Config changes don't affect ImmutableConfig). Two atomic commits: (1) ImmutableConfig type creation, (2) comprehensive tests. All tests pass with race detector. Fixes TD-01 (configuration objects immutable after initialization). Incremental adoption documented; component migration deferred to future phase. No deviations from plan. Duration: 3 minutes.

**2026-01-23 (Plan 03-01 Execution):** Completed plan 03-01 (TracerWrapper with Noop Default). Created TracerWrapper type that uses noop.NewTracerProvider() as default, ensuring all span operations are always safe without nil-checks. NewTracerWrapper constructor guarantees valid tracer regardless of input. Added comprehensive tests verifying nil-safety with noop tracer. Deprecated existing createSpan function for backward compatibility during migration. Two atomic commits: (1) TracerWrapper implementation, (2) comprehensive tests. All tests pass with race detector. Fixes FRAG-04 (tracer nil-checks scattered). No deviations from plan. Duration: 3 minutes.

**2026-01-23 (Plan 02-02 Execution):** Completed plan 02-02 (Rate Limiting & Retry with Backoff). Implemented exponential backoff retry logic in HTTP client with configurable parameters: 3 max retries, 5s initial delay, 60s max delay, 2.0 backoff factor. Connection pool tuned with MaxIdleConns=100, MaxIdleConnsPerHost=20, IdleConnTimeout=90s. Fixed test suite performance issue by disabling retries in unit tests with SetRetryCount(0). Tests reduced from 247s to ~35s. Fixes SEC-03 (rate limiting and backoff).

**2026-01-23 (Plan 02-01 Execution):** Completed plan 02-01 (TLS Enforcement & API Key Security). Implemented TLS enforcement requiring NBU_INSECURE_MODE=true environment variable for insecure mode. TLS 1.2 enforced as minimum version in HTTP client. Comprehensive API key protection audit completed - verified API key never appears in error messages, logs, or span attributes. Security warnings upgraded to Error level for better visibility. Three atomic commits: (1) TLS enforcement validation, (2) TLS 1.2 minimum and security logging, (3) API key audit verification. All tests pass with race detector. Fixes SEC-01 (secure API key handling) and SEC-02 (TLS enforcement). No deviations from plan. Duration: 5 minutes.

**2026-01-23 (Plan 01-03 Execution):** Completed plan 01-03 (Resource Cleanup). Implemented graceful HTTP client shutdown with connection draining. NbuClient now tracks active requests with atomic counter and mutex-protected state. Close() waits up to 30 seconds for active requests to complete before forcing cleanup. CloseWithContext() allows custom timeout control. Two auto-fixes: (1) Fixed blocking compilation issue from plan 01-01 (NewAPIVersionDetector signature), (2) Fixed data race in Close() by storing local channel reference before releasing lock. All tests pass with race detector. Fixes TD-05 (resource cleanup on shutdown). Duration: 10 minutes.

**2026-01-23 (Plan 01-02 Execution):** Completed plan 01-02 (URL Validation). Added validateNBUBaseURL() method to validate NBU server URL format during config initialization. Invalid URLs now caught at startup with clear error messages instead of silently failing in BuildURL(). BuildURL() signature unchanged for backward compatibility. Added comprehensive tests for URL validation scenarios. One auto-fix: corrected broken test YAML using literal strings instead of actual values. Fixes FRAG-03 (URL parsing errors silently ignored). Duration: 10 minutes.

**2026-01-23 (Plan 01-01 Execution):** Completed plan 01-01 (Version Detection Immutability). Refactored APIVersionDetector to be immutable - removed config reference, added immutable baseURL and apiKey fields. Eliminated setTemporaryVersion() and restoreOriginalVersion() methods that caused BUG-01. Config mutation now happens only in performVersionDetectionIfNeeded() after successful detection. Added comprehensive tests verifying config immutability during detection success, failure, and context cancellation. Fixes BUG-01 (state restoration on context cancellation) and FRAG-01 (shared config reference). No deviations from plan. Duration: 9 minutes.

**2026-01-23 (Plan 01-04 Execution):** Completed plan 01-04 (Error Channel Pattern). Replaced log.Fatalf in HTTP server goroutine with buffered error channel. Main function now uses select to handle both shutdown signals and server errors gracefully. No deviations from plan. Duration: 4 minutes.

**2026-01-23 (Phase 1 Planning):** Created 4 plans for Phase 1:

- 01-01: Refactor version detection to return detected version without mutating config
- 01-02: Add URL validation during config initialization
- 01-03: Implement proper connection drain in NbuClient.Close()
- 01-04: Replace log.Fatalf in goroutine with error channel pattern

All 4 plans are Wave 1 (independent, can run in parallel). Each plan includes:

- Specific file changes with code examples
- Verification criteria
- Test requirements
- must_haves derived from phase success criteria

**2026-01-23 (Roadmap Creation):** Roadmap created with 6 phases derived from 27 requirements. All requirements mapped to phases with 100% coverage. Phase 1 (Critical Fixes & Stability) ready for planning with 5 requirements: BUG-01, FRAG-01, FRAG-03, TD-05, TD-06.

**2026-01-23 (Initial):** Project initialized from codebase concerns analysis. 27 active requirements identified across 7 categories. Starting milestone v1.1 for codebase improvements.

---

**2026-01-23 (Phase 1 Verification):** Phase 1 verified by gsd-verifier. Score: 15/16 must-haves (93.75%). All 4 success criteria from ROADMAP.md met. All 5 requirements satisfied (BUG-01, FRAG-01, FRAG-03, TD-05, TD-06). One design decision noted: BuildURL() keeps original signature for backward compatibility, URL validation happens at startup instead. Report: .planning/phases/01-critical-fixes-stability/01-VERIFICATION.md

---

_Last updated: 2026-01-23 after completing plan 03-02_
