---
phase: 03-architecture-improvements
plan: 01
subsystem: tracing
tags: [opentelemetry, tracing, noop, nil-safety]

# Dependency graph
requires:
  - phase: 02-security-hardening
    provides: Secure HTTP client foundation
provides:
  - TracerWrapper type with guaranteed nil-safe operations
  - Noop tracer as default when tracing disabled
  - Centralized tracer initialization pattern
affects: [03-02, 03-03, future-tracing-work]

# Tech tracking
tech-stack:
  added: ["go.opentelemetry.io/otel/trace/noop"]
  patterns: ["Wrapper pattern for nil-safety", "Noop provider as default"]

key-files:
  created:
    - internal/exporter/tracing_test.go
  modified:
    - internal/exporter/tracing.go

key-decisions:
  - "Use noop.NewTracerProvider() as default instead of nil tracer"
  - "TracerWrapper guarantees valid span return (never nil)"
  - "Keep deprecated createSpan for backward compatibility during migration"

patterns-established:
  - "Wrapper pattern: wrap external dependency to provide nil-safety guarantees"
  - "Noop provider: use OpenTelemetry's noop implementation for zero-overhead disabled tracing"

# Metrics
duration: 3min
completed: 2026-01-23
---

# Phase 03 Plan 01: TracerWrapper with Noop Default Summary

**TracerWrapper type centralizes tracer nil-safety using OpenTelemetry's noop.NewTracerProvider() as default, eliminating scattered nil-checks across the codebase**

## Performance

- **Duration:** 3 min
- **Started:** 2026-01-23T16:52:40Z
- **Completed:** 2026-01-23T16:55:13Z
- **Tasks:** 3
- **Files modified:** 2

## Accomplishments

- Created TracerWrapper type that guarantees nil-safe span operations
- Implemented NewTracerWrapper constructor with noop.NewTracerProvider() default
- Added comprehensive tests verifying all span methods work without nil-checks
- Marked existing createSpan function as deprecated for backward compatibility

## Task Commits

Each task was committed atomically:

1. **Task 1: Create TracerWrapper with noop default** - `0191a20` (feat)
   - Added TracerWrapper type with NewTracerWrapper constructor
   - Implemented StartSpan method that always returns valid span
   - Added Tracer accessor for advanced use cases
   - Deprecated createSpan for future migration

2. **Task 2: Add comprehensive tests for TracerWrapper** - `4e29f51` (test)
   - Test nil provider defaults to noop
   - Test non-nil provider usage
   - Test StartSpan nil-safety guarantee
   - Test all span methods safe without nil-checks
   - Test tracer accessor
   - Test context propagation with parent/child spans

3. **Task 3: Verify integration and commit** - (verification only, no commit)
   - Ran full test suite with race detector
   - Verified go vet passes
   - Confirmed noop import usage

## Files Created/Modified

- `internal/exporter/tracing.go` - Added TracerWrapper type with noop default, deprecated createSpan
- `internal/exporter/tracing_test.go` - Comprehensive tests for TracerWrapper nil-safety (95 lines, 6 test cases)

## Decisions Made

- **Use noop.NewTracerProvider() as default:** Instead of returning nil tracer, use OpenTelemetry's built-in noop implementation. This provides zero-overhead disabled tracing with safe no-op span operations.

- **TracerWrapper guarantees valid span return:** All span operations are guaranteed safe to call without nil-checks. This centralizes nil-safety in one place rather than scattering checks throughout the codebase.

- **Keep deprecated createSpan for backward compatibility:** Marked existing createSpan as deprecated but kept it functional to avoid breaking existing call sites. Migration to TracerWrapper can happen incrementally.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

TracerWrapper ready for use in migration tasks:

- Plan 03-02 can refactor FetchStorage and FetchAllJobs to use TracerWrapper
- Plan 03-03 can update NbuCollector.Collect to use TracerWrapper
- All scattered nil-checks can be removed as migration progresses

No blockers. FRAG-04 (tracer nil-checks scattered) centralized and ready for cleanup.

---

_Phase: 03-architecture-improvements_
_Completed: 2026-01-23_
