# Requirements: NBU Exporter v1.1

**Defined:** 2026-01-23
**Core Value:** Improve code reliability and maintainability by fixing identified concerns

## v1.1 Requirements

Requirements for codebase improvements milestone. Each maps to roadmap phases.

### Known Bugs

- [ ] **BUG-01**: Fix version detection state restoration on context cancellation

### Security

- [ ] **SEC-01**: Implement secure API key handling in memory
- [ ] **SEC-02**: Enforce TLS verification by default with explicit opt-out
- [ ] **SEC-03**: Add rate limiting and backoff for metric collection

### Tech Debt

- [ ] **TD-01**: Fix configuration mutation during version detection (immutable config pattern)
- [ ] **TD-02**: Eliminate global OpenTelemetry state (dependency injection)
- [ ] **TD-03**: Replace pipe-delimited metric keys with structured format
- [ ] **TD-04**: Add test coverage for main.go entry point
- [ ] **TD-05**: Implement proper resource cleanup in NbuClient.Close()
- [ ] **TD-06**: Replace fatal log in async goroutine with error channel

### Fragile Areas

- [ ] **FRAG-01**: Remove shared config reference from APIVersionDetector
- [ ] **FRAG-02**: Add connection pool lifecycle management
- [ ] **FRAG-03**: Handle URL parsing errors in BuildURL
- [ ] **FRAG-04**: Centralize tracer nil-checks into wrapper method

### Test Coverage

- [ ] **TEST-01**: Add integration tests for main.go (0% → target 60%+)
- [ ] **TEST-02**: Increase testutil coverage (51.9% → 80%+)
- [ ] **TEST-03**: Increase telemetry coverage (78.3% → 90%+)
- [ ] **TEST-04**: Add concurrent collector access tests
- [ ] **TEST-05**: Add client error handling edge case tests

### Performance

- [ ] **PERF-01**: Increase pagination page size for job fetching
- [ ] **PERF-02**: Parallelize storage and job metric collection
- [ ] **PERF-03**: Implement streaming metrics to reduce memory accumulation
- [ ] **PERF-04**: Cache split results for pipe-delimited keys

### Missing Features

- [ ] **FEAT-01**: Add caching for storage metrics
- [ ] **FEAT-02**: Implement health check with NetBackup connectivity verification
- [ ] **FEAT-03**: Add metric staleness tracking
- [ ] **FEAT-04**: Implement dynamic configuration reload

## Future Requirements

Deferred to future milestones. Not in current roadmap.

### Scaling

- **SCALE-01**: Multi-collector framework for multiple NBU servers
- **SCALE-02**: Time-based job sharding for large environments

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| Certificate pinning | TLS verification improvements sufficient for now |
| Mobile/web dashboard | Not core exporter functionality |
| Real-time streaming metrics | Would require architectural changes |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| BUG-01 | — | Pending |
| SEC-01 | — | Pending |
| SEC-02 | — | Pending |
| SEC-03 | — | Pending |
| TD-01 | — | Pending |
| TD-02 | — | Pending |
| TD-03 | — | Pending |
| TD-04 | — | Pending |
| TD-05 | — | Pending |
| TD-06 | — | Pending |
| FRAG-01 | — | Pending |
| FRAG-02 | — | Pending |
| FRAG-03 | — | Pending |
| FRAG-04 | — | Pending |
| TEST-01 | — | Pending |
| TEST-02 | — | Pending |
| TEST-03 | — | Pending |
| TEST-04 | — | Pending |
| TEST-05 | — | Pending |
| PERF-01 | — | Pending |
| PERF-02 | — | Pending |
| PERF-03 | — | Pending |
| PERF-04 | — | Pending |
| FEAT-01 | — | Pending |
| FEAT-02 | — | Pending |
| FEAT-03 | — | Pending |
| FEAT-04 | — | Pending |

**Coverage:**
- v1.1 requirements: 27 total
- Mapped to phases: 0
- Unmapped: 27 (pending roadmap creation)

---
*Requirements defined: 2026-01-23*
*Last updated: 2026-01-23 after initial definition*
