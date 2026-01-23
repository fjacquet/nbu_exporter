# NBU Exporter Codebase Improvement

## Shipped: v1.1 Codebase Improvements (2026-01-23)

**Delivered:** Comprehensive codebase improvements addressing technical debt, bugs, security concerns, and test coverage gaps.

**Accomplishments:**

- Fixed known bugs and fragile code patterns (5 requirements)
- Improved security posture with TLS enforcement, API key protection, rate limiting (3 requirements)
- Reduced technical debt: immutable config, eliminated global OTel state, structured metrics (6 requirements)
- Increased test coverage: main.go 0%→60%, testutil 51.9%→97.5%, telemetry 76.6%→83.7% (6 requirements)
- Added operational features: storage caching, health checks with connectivity verification, dynamic config reload (4 requirements)
- Performance optimizations: batch pagination (~100x fewer API calls), parallel collection, pre-allocation (4 requirements)

## What This Is

A project to address technical debt, bugs, security concerns, and test coverage gaps in the NetBackup Prometheus exporter codebase. This is a brownfield improvement initiative focused on code quality, reliability, and maintainability.

## Core Value

Improve code reliability and maintainability by fixing identified concerns without breaking existing functionality.

## Requirements

### Validated

<!-- Shipped and confirmed valuable. -->

**v1.0 Base:**

- ✓ Prometheus metrics exposition for NetBackup storage and jobs — v1.0
- ✓ Auto-detection of NetBackup API version — v1.0
- ✓ OpenTelemetry distributed tracing support — v1.0
- ✓ YAML-based configuration — v1.0

**v1.1 Codebase Improvements (all 27 requirements):**

- ✓ **BUG-01**: Fix version detection state restoration on context cancellation — v1.1
- ✓ **SEC-01**: Implement secure API key handling in memory — v1.1
- ✓ **SEC-02**: Enforce TLS verification by default with explicit opt-out — v1.1
- ✓ **SEC-03**: Add rate limiting and backoff for metric collection — v1.1
- ✓ **TD-01**: Fix configuration mutation during version detection (immutable config pattern) — v1.1
- ✓ **TD-02**: Eliminate global OpenTelemetry state (dependency injection) — v1.1
- ✓ **TD-03**: Replace pipe-delimited metric keys with structured format — v1.1
- ✓ **TD-04**: Add test coverage for main.go entry point — v1.1 (60%+ achieved)
- ✓ **TD-05**: Implement proper resource cleanup in NbuClient.Close() — v1.1
- ✓ **TD-06**: Replace fatal log in async goroutine with error channel — v1.1
- ✓ **FRAG-01**: Remove shared config reference from APIVersionDetector — v1.1
- ✓ **FRAG-02**: Add connection pool lifecycle management — v1.1
- ✓ **FRAG-03**: Handle URL parsing errors in BuildURL — v1.1
- ✓ **FRAG-04**: Centralize tracer nil-checks into wrapper method — v1.1
- ✓ **TEST-01**: Add integration tests for main.go (0% → 60%+) — v1.1
- ✓ **TEST-02**: Increase testutil coverage (51.9% → 97.5%) — v1.1
- ✓ **TEST-03**: Increase telemetry coverage (76.6% → 83.7%) — v1.1
- ✓ **TEST-04**: Add concurrent collector access tests — v1.1
- ✓ **TEST-05**: Add client error handling edge case tests — v1.1
- ✓ **PERF-01**: Increase pagination page size for job fetching (~100x reduction) — v1.1
- ✓ **PERF-02**: Parallelize storage and job metric collection — v1.1
- ✓ **PERF-03**: Pre-allocate maps/slices for memory efficiency — v1.1
- ✓ **PERF-04**: Structured keys eliminate split operations (completed in TD-03) — v1.1
- ✓ **FEAT-01**: Add caching for storage metrics (TTL-based) — v1.1
- ✓ **FEAT-02**: Implement health check with NetBackup connectivity verification — v1.1
- ✓ **FEAT-03**: Add metric staleness tracking (nbu_up, nbu_last_scrape_timestamp_seconds) — v1.1
- ✓ **FEAT-04**: Implement dynamic configuration reload (SafeConfig + SIGHUP + fsnotify) — v1.1

### Active

<!-- Next scope. Planning for future milestones. -->

(None currently — ready for v1.2 planning)

### Out of Scope

<!-- Explicit boundaries. Includes reasoning to prevent re-adding. -->

- Multi-collector framework for multiple NBU servers — High complexity, requires architectural changes beyond this improvement cycle
- Time-based job sharding — Performance optimization for very large environments, defer to future
- Certificate pinning — Alternative security measure, TLS verification improvements sufficient for now
- Mobile/web dashboard — Not core exporter functionality

## Context

This is a brownfield improvement project on an existing, functional Prometheus exporter. Started with 27 identified requirements across reliability, security, test coverage, and performance. v1.1 milestone successfully shipped all 27 requirements in 6 phases.

**Current state (post-v1.1):**

- Go 1.25 with modern dependency versions
- **Test coverage increased:** main 0%→60%, testutil 51.9%→97.5%, telemetry 76.6%→83.7%
- **Reliability improved:** Immutable config, proper resource cleanup, graceful degradation
- **Security hardened:** TLS 1.2 default, API key protection verified, rate limiting with exponential backoff
- **Performance optimized:** Batch pagination (~100x fewer API calls), parallel collection, pre-allocation hints
- **Operational visibility:** Storage caching (TTL), health endpoint with connectivity check, metrics staleness tracking, dynamic config reload

**Architecture improvements:**
- Eliminated 50+ tracer nil-checks through noop wrapper
- Removed global OpenTelemetry state via dependency injection
- Structured metric keys replace pipe-delimited strings (handles special characters safely)
- SafeConfig pattern with RWMutex for thread-safe concurrent access

**Code quality:**
- 99 files modified in v1.1 (5,013 insertions, 635 deletions)
- All changes maintain backwards compatibility
- Incremental adoption patterns allow gradual migration

**Codebase map available at:** `.planning/codebase/`
**Milestone archive at:** `.planning/milestones/v1.1-ROADMAP.md`

## Constraints

- **Backwards Compatibility**: Must not break existing Prometheus metrics format or configuration schema
- **Zero Downtime**: Changes should not require extended maintenance windows
- **Test First**: All fixes should include corresponding tests
- **Incremental**: Changes should be atomic and independently deployable

## Key Decisions

<!-- Decisions that constrain future work. Add throughout project lifecycle. -->

| Decision                                                                                 | Rationale                                       | Outcome           |
| ---------------------------------------------------------------------------------------- | ----------------------------------------------- | ----------------- |
| Address concerns in priority order: Bugs → Security → Tech Debt → Performance → Features | Fix breaking issues first, then improve quality | ✅ Executed (v1.1) |
| Maintain backwards compatibility for Prometheus metrics                                  | Avoid breaking existing dashboards and alerts   | ✅ Verified (v1.1) |
| Require test coverage for all changes                                                    | Prevent regression, improve confidence          | ✅ Achieved (v1.1) |
| Use ImmutableConfig snapshot pattern for runtime config thread-safety                    | Enable incremental adoption without breaking API | ✅ Implemented     |
| Options pattern for TracerProvider injection eliminates global state                     | Explicit dependency flow vs implicit global      | ✅ Implemented     |
| SafeConfig with RWMutex for concurrent reads, serialized writes                         | Balance performance with thread-safety          | ✅ Implemented     |
| Phase numbering continues across milestones (never restart at 01)                        | Preserve full project history, maintain git refs | ✅ Established     |

---

_Last updated: 2026-01-23 after v1.1 milestone completion_
_Next: Ready for v1.2 planning_
