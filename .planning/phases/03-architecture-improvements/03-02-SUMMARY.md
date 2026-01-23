---
phase: 03-architecture-improvements
plan: 02
subsystem: architecture
tags: [opentelemetry, dependency-injection, tracing, options-pattern]

# Dependency graph
requires:
  - phase: 03-01
    provides: TracerWrapper type for nil-safe tracing
provides:
  - Options pattern (WithTracerProvider, WithCollectorTracerProvider) for explicit TracerProvider injection
  - Elimination of global otel.GetTracerProvider() calls in component constructors
  - Explicit dependency flow: telemetry.Manager → NbuCollector → NbuClient
affects: [any-future-components-needing-opentelemetry, testing-utilities]

# Tech tracking
tech-stack:
  added: []
  patterns: [options-pattern, dependency-injection, tracer-provider-injection]

key-files:
  created: []
  modified:
    - internal/exporter/client.go
    - internal/exporter/prometheus.go
    - internal/exporter/netbackup.go
    - internal/telemetry/manager.go
    - main.go

key-decisions: []

patterns-established:
  - "Options pattern: Functions accept variadic ...Option parameters for optional configuration"
  - "TracerProvider injection: Components receive trace.TracerProvider via options instead of accessing global state"
  - "Dependency flow: telemetry.Manager.TracerProvider() → WithCollectorTracerProvider → WithTracerProvider"

# Metrics
duration: 10min
completed: 2026-01-23
---

# Phase 03 Plan 02: TracerProvider Injection Summary

**Options pattern eliminates global OpenTelemetry state access via explicit TracerProvider injection through component constructors**

## Performance

- **Duration:** 10 min
- **Started:** 2026-01-23T04:46:32Z
- **Completed:** 2026-01-23T04:56:23Z
- **Tasks:** 4
- **Files modified:** 9 (5 source files + 4 test files)

## Accomplishments
- NbuClient and NbuCollector accept TracerProvider via options pattern
- Removed all global otel.GetTracerProvider() and otel.Tracer() calls from constructors
- TracerProvider flows explicitly: telemetry.Manager → main.go → NbuCollector → NbuClient
- All components work correctly without TracerProvider (noop default via TracerWrapper)
- All tests pass with race detector

## Task Commits

Each task was committed atomically:

1. **Task 1: Add options pattern to NbuClient** - `98d60f3` (feat)
2. **Task 2: Add options pattern to NbuCollector** - `6cf884a` (feat)
3. **Task 3: Inject TracerProvider from telemetry manager** - `956d7a6` (feat)
4. **Task 4: Update tests and verify full suite** - `15d52a6` (test)

## Files Created/Modified

**Source files:**
- `internal/exporter/client.go` - Added ClientOption with WithTracerProvider, updated to use TracerWrapper
- `internal/exporter/prometheus.go` - Added CollectorOption with WithCollectorTracerProvider, updated to use TracerWrapper
- `internal/exporter/netbackup.go` - Updated to use client.tracing.StartSpan()
- `internal/telemetry/manager.go` - Added TracerProvider() method, separated SDK/API trace imports
- `main.go` - Updated to inject TracerProvider from telemetry manager to collector

**Test files:**
- `internal/exporter/client_test.go` - Updated for TracerWrapper behavior
- `internal/exporter/prometheus_test.go` - Updated collector test instantiation
- `internal/exporter/otel_benchmark_test.go` - Updated benchmark tests
- `internal/exporter/otel_integration_test.go` - Updated integration tests

## Decisions Made

None - followed plan exactly as specified.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

**Issue: SDK vs API trace import conflict in telemetry.Manager**
- **Problem:** Manager's tracerProvider field was `*sdktrace.TracerProvider` (SDK concrete type) but new method needed to return `trace.TracerProvider` (API interface)
- **Solution:** Separated imports: `sdktrace "go.opentelemetry.io/otel/sdk/trace"` for SDK types, `"go.opentelemetry.io/otel/trace"` for API types
- **Resolution:** Updated all SDK references to use `sdktrace.` prefix, API method returns interface type

**Issue: Test expectations for nil-safety**
- **Problem:** Old tests expected nil spans when tracer was disabled, but TracerWrapper always returns valid spans
- **Solution:** Updated test expectations - TracerWrapper guarantees non-nil spans (noop if tracing disabled)
- **Resolution:** Tests now verify spans are always valid and don't panic

## Next Phase Readiness

**Ready for TD-03 (Immutable Config):** TracerProvider injection complete, global state eliminated from tracing path.

**Blockers:** None

**Concerns:** None

**Notes:**
- All existing tests pass without modification to tracing setup (as required)
- Components gracefully degrade when TracerProvider is not provided
- Options pattern established for future optional dependencies

---
*Phase: 03-architecture-improvements*
*Completed: 2026-01-23*
