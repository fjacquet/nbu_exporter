# Roadmap: NBU Exporter v1.1 Codebase Improvements

## Overview

This milestone addresses technical debt, bugs, security concerns, and test coverage gaps in the NetBackup Prometheus exporter. The journey progresses from critical stability fixes through security hardening and architectural improvements, then expands test coverage, optimizes performance, and adds operational features. Each phase delivers measurable improvements while maintaining backwards compatibility.

## Phases

**Phase Numbering:**

- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [x] **Phase 1: Critical Fixes & Stability** - Eliminate crashes and resource leaks
- [x] **Phase 2: Security Hardening** - Protect sensitive data and enforce secure defaults
- [x] **Phase 3: Architecture Improvements** - Reduce technical debt and improve maintainability
- [x] **Phase 4: Test Coverage** - Increase confidence and prevent regressions
- [x] **Phase 5: Performance Optimizations** - Improve scrape performance and reduce resource usage
- [ ] **Phase 6: Operational Features** - Add missing operational capabilities

## Phase Details

### Phase 1: Critical Fixes & Stability

**Goal**: Exporter runs reliably without crashes or resource leaks
**Depends on**: Nothing (first phase)
**Requirements**: BUG-01, FRAG-01, FRAG-03, TD-05, TD-06
**Success Criteria** (what must be TRUE):

1. Exporter handles context cancellation during version detection without leaving config in inconsistent state
2. Exporter can be stopped and restarted multiple times without connection leaks
3. Exporter continues running even if metric collection errors occur (no fatal exits from goroutines)
4. Invalid URLs in configuration are caught during startup with clear error messages

**Plans:** 4 plans (all Wave 1 - parallel)

Plans:

- [x] 01-01-PLAN.md - Version Detection Immutability (BUG-01 + FRAG-01)
- [x] 01-02-PLAN.md - URL Validation (FRAG-03)
- [x] 01-03-PLAN.md - Resource Cleanup (TD-05)
- [x] 01-04-PLAN.md - Error Channel Pattern (TD-06)

### Phase 2: Security Hardening

**Goal**: Sensitive data is protected and connections are secure by default
**Depends on**: Nothing (independent of Phase 1)
**Requirements**: SEC-01, SEC-02, SEC-03
**Success Criteria** (what must be TRUE):

1. API key is not visible in memory dumps or logs (except when debug mode is explicitly enabled)
2. TLS verification is enabled by default; insecure mode requires explicit opt-in flag
3. Exporter handles API rate limits gracefully with automatic backoff and retry

**Plans:** 2 plans (Wave 1 + Wave 2 sequential)

Plans:

- [x] 02-01-PLAN.md - TLS Enforcement & API Key Security (SEC-01 + SEC-02)
- [x] 02-02-PLAN.md - Rate Limiting & Retry with Backoff (SEC-03)

### Phase 3: Architecture Improvements

**Goal**: Code is maintainable with clear ownership and minimal shared state
**Depends on**: Phase 1 (config mutation relates to BUG-01/FRAG-01)
**Requirements**: TD-01, TD-02, TD-03, FRAG-02, FRAG-04
**Success Criteria** (what must be TRUE):

1. Configuration objects are immutable after initialization; version detection returns results without mutating config
2. OpenTelemetry dependencies are explicitly injected rather than accessed via global state
3. Metric keys use structured format that handles special characters safely
4. Connection pool lifecycle is explicitly managed with documented cleanup requirements
5. Tracer nil-checks are centralized in a single wrapper method (DRY principle)

**Plans:** 5 plans (4 waves)

Plans:

- [x] 03-01-PLAN.md - Tracer Wrapper with noop default (FRAG-04) - Wave 1
- [x] 03-02-PLAN.md - TracerProvider Injection (TD-02) - Wave 2
- [x] 03-03-PLAN.md - Structured Metric Keys (TD-03) - Wave 3
- [x] 03-04-PLAN.md - Immutable Config (TD-01) - Wave 3
- [x] 03-05-PLAN.md - Connection Lifecycle Integration (FRAG-02) - Wave 4

### Phase 4: Test Coverage

**Goal**: Critical code paths have automated test coverage to prevent regressions
**Depends on**: Phases 1-3 complete (stable codebase makes testing easier)
**Requirements**: TD-04, TEST-01, TEST-02, TEST-03, TEST-04, TEST-05
**Success Criteria** (what must be TRUE):

1. Main package test coverage increases from 0% to 60%+ with integration tests covering server startup, graceful shutdown, and signal handling
2. Testutil package coverage increases from 51.9% to 80%+
3. Telemetry package coverage increases from 76.6% to 90%+
4. Tests verify concurrent access to collector without race conditions
5. Tests cover client error handling edge cases (network timeouts, unusual HTTP responses)

**Plans:** 4 plans (all Wave 1 - parallel)

Plans:

- [x] 04-01-PLAN.md - Main Package Integration Tests (TEST-01 + TD-04) - Wave 1
- [x] 04-02-PLAN.md - Testutil Coverage Expansion (TEST-02) - Wave 1
- [x] 04-03-PLAN.md - Telemetry Coverage Expansion (TEST-03) - Wave 1
- [x] 04-04-PLAN.md - Concurrent & Edge Case Tests (TEST-04 + TEST-05) - Wave 1

**Note**: TD-04 and TEST-01 both address main.go test coverage and will be implemented together in plan 04-01.

### Phase 5: Performance Optimizations

**Goal**: Scrape times are reduced and memory usage is predictable
**Depends on**: Phase 3 (PERF-04 benefits from TD-03 structured keys)
**Requirements**: PERF-01, PERF-02, PERF-03, PERF-04
**Success Criteria** (what must be TRUE):

1. Job fetching uses larger page sizes (100+ items), reducing number of sequential API calls
2. Storage and job metrics are collected in parallel, reducing total scrape time to max(storage_time, jobs_time) instead of sum
3. Metrics are streamed rather than accumulated in memory, bounding memory usage even for large job counts
4. String split operations for metric keys are minimized through caching or structured storage

**Plans:** 3 plans (2 waves)

**Note:** PERF-04 (string split caching) was already completed in Phase 3 Plan 03-03 (TD-03 structured metric keys). The Labels() method now provides direct label access without string parsing.

Plans:

- [x] 05-01-PLAN.md - Batch Pagination (PERF-01) - Wave 1
- [x] 05-02-PLAN.md - Parallel Collection (PERF-02) - Wave 1
- [x] 05-03-PLAN.md - Pre-allocation Optimization (PERF-03) - Wave 2

### Phase 6: Operational Features

**Goal**: Operators have visibility into exporter health and metrics freshness
**Depends on**: Nothing (features independent of previous phases)
**Requirements**: FEAT-01, FEAT-02, FEAT-03, FEAT-04
**Success Criteria** (what must be TRUE):

1. Storage metrics are cached and only refreshed at configured intervals (configurable TTL)
2. Health endpoint verifies NetBackup connectivity and API version before returning success status
3. Stale metrics are identified and either refreshed or marked as unavailable in exposition
4. Configuration changes (credentials, server address) can be applied without restarting the exporter
   **Plans**: TBD

Plans:

- [ ] 06-01: [TBD during phase planning]

## Progress

**Execution Order:**
Phases execute in numeric order: 1 -> 2 -> 3 -> 4 -> 5 -> 6

| Phase                         | Plans Complete | Status      | Completed  |
| ----------------------------- | -------------- | ----------- | ---------- |
| 1. Critical Fixes & Stability | 4/4            | Complete    | 2026-01-23 |
| 2. Security Hardening         | 2/2            | Complete    | 2026-01-23 |
| 3. Architecture Improvements  | 5/5            | Complete    | 2026-01-23 |
| 4. Test Coverage              | 4/4            | Complete    | 2026-01-23 |
| 5. Performance Optimizations  | 3/3            | Complete    | 2026-01-23 |
| 6. Operational Features       | 0/TBD          | Not started | -          |

---

_Roadmap created: 2026-01-23_
_Last updated: 2026-01-23 after Phase 5 planning_
