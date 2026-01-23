# NBU Exporter Codebase Improvement

## Current Milestone: v1.1 Codebase Improvements

**Goal:** Address technical debt, bugs, security concerns, and test coverage gaps identified in codebase analysis.

**Target features:**
- Fix known bugs and fragile code patterns
- Improve security posture (API key handling, TLS enforcement, rate limiting)
- Reduce technical debt (config mutation, global state, resource cleanup)
- Increase test coverage (main.go from 0% to 60%+)
- Add missing operational features (health checks, caching, config reload)

## What This Is

A project to address technical debt, bugs, security concerns, and test coverage gaps in the NetBackup Prometheus exporter codebase. This is a brownfield improvement initiative focused on code quality, reliability, and maintainability.

## Core Value

Improve code reliability and maintainability by fixing identified concerns without breaking existing functionality.

## Requirements

### Validated

<!-- Shipped and confirmed valuable. -->

- ✓ Prometheus metrics exposition for NetBackup storage and jobs — v1.0
- ✓ Auto-detection of NetBackup API version — v1.0
- ✓ OpenTelemetry distributed tracing support — v1.0
- ✓ YAML-based configuration — v1.0

### Active

<!-- Current scope. Building toward these. -->

**Tech Debt:**
- [ ] **TD-01**: Fix configuration mutation during version detection (immutable config pattern)
- [ ] **TD-02**: Eliminate global OpenTelemetry state (dependency injection)
- [ ] **TD-03**: Replace pipe-delimited metric keys with structured format
- [ ] **TD-04**: Add test coverage for main.go entry point
- [ ] **TD-05**: Implement proper resource cleanup in NbuClient.Close()
- [ ] **TD-06**: Replace fatal log in async goroutine with error channel

**Known Bugs:**
- [ ] **BUG-01**: Fix version detection state restoration on context cancellation

**Security:**
- [ ] **SEC-01**: Implement secure API key handling in memory
- [ ] **SEC-02**: Enforce TLS verification by default with explicit opt-out
- [ ] **SEC-03**: Add rate limiting and backoff for metric collection

**Performance:**
- [ ] **PERF-01**: Increase pagination page size for job fetching
- [ ] **PERF-02**: Parallelize storage and job metric collection
- [ ] **PERF-03**: Implement streaming metrics to reduce memory accumulation
- [ ] **PERF-04**: Cache split results for pipe-delimited keys

**Fragile Areas:**
- [ ] **FRAG-01**: Remove shared config reference from APIVersionDetector
- [ ] **FRAG-02**: Add connection pool lifecycle management
- [ ] **FRAG-03**: Handle URL parsing errors in BuildURL
- [ ] **FRAG-04**: Centralize tracer nil-checks into wrapper method

**Test Coverage:**
- [ ] **TEST-01**: Add integration tests for main.go (0% → target 60%+)
- [ ] **TEST-02**: Increase testutil coverage (51.9% → 80%+)
- [ ] **TEST-03**: Increase telemetry coverage (78.3% → 90%+)
- [ ] **TEST-04**: Add concurrent collector access tests
- [ ] **TEST-05**: Add client error handling edge case tests

**Missing Features:**
- [ ] **FEAT-01**: Add caching for storage metrics
- [ ] **FEAT-02**: Implement health check with NetBackup connectivity verification
- [ ] **FEAT-03**: Add metric staleness tracking
- [ ] **FEAT-04**: Implement dynamic configuration reload

### Out of Scope

<!-- Explicit boundaries. Includes reasoning to prevent re-adding. -->

- Multi-collector framework for multiple NBU servers — High complexity, requires architectural changes beyond this improvement cycle
- Time-based job sharding — Performance optimization for very large environments, defer to future
- Certificate pinning — Alternative security measure, TLS verification improvements sufficient for now
- Mobile/web dashboard — Not core exporter functionality

## Context

This is a brownfield improvement project on an existing, functional Prometheus exporter. The codebase is well-structured with clear separation of concerns but has accumulated technical debt and has gaps in test coverage identified during codebase analysis.

**Current state:**
- Go 1.25 with modern dependency versions
- 97.1% coverage in exporter package, but 0% in main.go
- Functional but with reliability concerns around resource cleanup and error handling
- Security warnings present but mitigated by logging

**Codebase map available at:** `.planning/codebase/`

## Constraints

- **Backwards Compatibility**: Must not break existing Prometheus metrics format or configuration schema
- **Zero Downtime**: Changes should not require extended maintenance windows
- **Test First**: All fixes should include corresponding tests
- **Incremental**: Changes should be atomic and independently deployable

## Key Decisions

<!-- Decisions that constrain future work. Add throughout project lifecycle. -->

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Address concerns in priority order: Bugs → Security → Tech Debt → Performance → Features | Fix breaking issues first, then improve quality | — Pending |
| Maintain backwards compatibility for Prometheus metrics | Avoid breaking existing dashboards and alerts | — Pending |
| Require test coverage for all changes | Prevent regression, improve confidence | — Pending |

---
*Last updated: 2026-01-23 after initial project definition*
