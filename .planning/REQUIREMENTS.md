# Requirements: NBU Exporter v1.1

**Defined:** 2026-01-23
**Core Value:** Improve code reliability and maintainability by fixing identified concerns

## v1.1 Requirements

Requirements for codebase improvements milestone. Each maps to roadmap phases.

### Known Bugs

- [x] **BUG-01**: Fix version detection state restoration on context cancellation ✓

### Security

- [x] **SEC-01**: Implement secure API key handling in memory ✓
- [x] **SEC-02**: Enforce TLS verification by default with explicit opt-out ✓
- [x] **SEC-03**: Add rate limiting and backoff for metric collection ✓

### Tech Debt

- [x] **TD-01**: Fix configuration mutation during version detection (immutable config pattern) ✓
- [x] **TD-02**: Eliminate global OpenTelemetry state (dependency injection) ✓
- [x] **TD-03**: Replace pipe-delimited metric keys with structured format ✓
- [x] **TD-04**: Add test coverage for main.go entry point ✓
- [x] **TD-05**: Implement proper resource cleanup in NbuClient.Close() ✓
- [x] **TD-06**: Replace fatal log in async goroutine with error channel ✓

### Fragile Areas

- [x] **FRAG-01**: Remove shared config reference from APIVersionDetector ✓
- [x] **FRAG-02**: Add connection pool lifecycle management ✓
- [x] **FRAG-03**: Handle URL parsing errors in BuildURL ✓
- [x] **FRAG-04**: Centralize tracer nil-checks into wrapper method ✓

### Test Coverage

- [x] **TEST-01**: Add integration tests for main.go (0% → 60%+) ✓
- [x] **TEST-02**: Increase testutil coverage (51.9% → 97.5%) ✓
- [x] **TEST-03**: Increase telemetry coverage (76.6% → 83.7%) ✓
- [x] **TEST-04**: Add concurrent collector access tests ✓
- [x] **TEST-05**: Add client error handling edge case tests ✓

### Performance

- [x] **PERF-01**: Increase pagination page size for job fetching ✓
- [x] **PERF-02**: Parallelize storage and job metric collection ✓
- [x] **PERF-03**: Pre-allocate maps/slices for memory efficiency ✓
- [x] **PERF-04**: Structured keys eliminate split operations (completed in TD-03) ✓

### Missing Features

- [x] **FEAT-01**: Add caching for storage metrics ✓
- [x] **FEAT-02**: Implement health check with NetBackup connectivity verification ✓
- [x] **FEAT-03**: Add metric staleness tracking ✓
- [x] **FEAT-04**: Implement dynamic configuration reload ✓

## Future Requirements

Deferred to future milestones. Not in current roadmap.

### Scaling

- **SCALE-01**: Multi-collector framework for multiple NBU servers
- **SCALE-02**: Time-based job sharding for large environments

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature                     | Reason                                           |
| --------------------------- | ------------------------------------------------ |
| Certificate pinning         | TLS verification improvements sufficient for now |
| Mobile/web dashboard        | Not core exporter functionality                  |
| Real-time streaming metrics | Would require architectural changes              |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase   | Status   |
| ----------- | ------- | -------- |
| BUG-01      | Phase 1 | Complete |
| SEC-01      | Phase 2 | Complete |
| SEC-02      | Phase 2 | Complete |
| SEC-03      | Phase 2 | Complete |
| TD-01       | Phase 3 | Complete |
| TD-02       | Phase 3 | Complete |
| TD-03       | Phase 3 | Complete |
| TD-04       | Phase 4 | Complete |
| TD-05       | Phase 1 | Complete |
| TD-06       | Phase 1 | Complete |
| FRAG-01     | Phase 1 | Complete |
| FRAG-02     | Phase 3 | Complete |
| FRAG-03     | Phase 1 | Complete |
| FRAG-04     | Phase 3 | Complete |
| TEST-01     | Phase 4 | Complete |
| TEST-02     | Phase 4 | Complete |
| TEST-03     | Phase 4 | Complete |
| TEST-04     | Phase 4 | Complete |
| TEST-05     | Phase 4 | Complete |
| PERF-01     | Phase 5 | Complete |
| PERF-02     | Phase 5 | Complete |
| PERF-03     | Phase 5 | Complete |
| PERF-04     | Phase 3 | Complete |
| FEAT-01     | Phase 6 | Complete |
| FEAT-02     | Phase 6 | Complete |
| FEAT-03     | Phase 6 | Complete |
| FEAT-04     | Phase 6 | Complete |

**Coverage:**

- v1.1 requirements: 27 total
- Mapped to phases: 27
- Unmapped: 0 (100% coverage)

**Note:** TD-04 and TEST-01 both address main.go test coverage (0% → 60%+) and will be implemented together in Phase 4.

---

_Requirements defined: 2026-01-23_
_Last updated: 2026-01-23 after Phase 5 completion_
