---
phase: 01-critical-fixes-stability
plan: 01
subsystem: api
tags: [go, netbackup, api-client, version-detection, immutability]

# Dependency graph
requires:
  - phase: baseline
    provides: Existing version detection implementation with config mutation
provides:
  - Immutable version detection that returns detected version without modifying config
  - Single point of config mutation in performVersionDetectionIfNeeded after successful detection
  - Config immutability tests verifying no mutation during detection or on context cancellation
affects: [01-03-resource-cleanup]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Immutable detectors that return results instead of mutating input state"
    - "Config mutation only after successful operation completion"
    - "Version detection uses inline header construction instead of client state"

key-files:
  created: []
  modified:
    - internal/exporter/version_detector.go
    - internal/exporter/client.go
    - internal/exporter/version_detector_test.go
    - internal/exporter/version_detection_integration_test.go

key-decisions:
  - "APIVersionDetector stores only immutable values (baseURL, apiKey) instead of config reference"
  - "Version detection builds headers inline with test version instead of relying on client.getHeaders()"
  - "Config mutation happens only in performVersionDetectionIfNeeded after successful detection"

patterns-established:
  - "Detectors return results without side effects - caller applies changes"
  - "Context cancellation cannot leave config in inconsistent state"

# Metrics
duration: 9min
completed: 2026-01-23
---

# Phase 1 Plan 1: Version Detection Immutability Summary

**APIVersionDetector refactored to be immutable - returns detected version without mutating config, eliminating race conditions and state restoration bugs**

## Performance

- **Duration:** 9 min
- **Started:** 2026-01-23T03:21:00Z
- **Completed:** 2026-01-23T03:29:41Z
- **Tasks:** 3
- **Files modified:** 4

## Accomplishments

- Removed mutable config reference from APIVersionDetector - now stores only immutable values (baseURL, apiKey)
- Eliminated setTemporaryVersion() and restoreOriginalVersion() methods that caused BUG-01
- Config mutation happens only in performVersionDetectionIfNeeded() after successful detection
- Added comprehensive tests verifying config immutability during detection success, failure, and context cancellation

## Task Commits

Each task was committed atomically:

1. **Tasks 1-2: Refactor APIVersionDetector to be immutable + Update client.go** - `4955592` (refactor)
   - Removed cfg field, added baseURL and apiKey fields
   - Updated constructor signature: NewAPIVersionDetector(client, baseURL, apiKey)
   - Removed setTemporaryVersion() and restoreOriginalVersion() methods
   - Updated performVersionDetectionIfNeeded() to create detector with immutable values
   - Updated all test files to use new constructor signature

2. **Task 3: Add config immutability tests** - `643737d` (test)
   - TestAPIVersionDetectorConfigImmutability verifies config unchanged during successful and failed detection
   - TestAPIVersionDetectorContextCancellationNoConfigMutation verifies config unchanged on context cancellation
   - All tests pass including race detection

## Files Created/Modified

- `internal/exporter/version_detector.go` - APIVersionDetector struct refactored to be immutable, stores baseURL and apiKey instead of config reference
- `internal/exporter/client.go` - performVersionDetectionIfNeeded() creates detector with immutable values, config mutation only after successful detection
- `internal/exporter/version_detector_test.go` - Added config immutability tests, updated test helper to use new constructor
- `internal/exporter/version_detection_integration_test.go` - Updated integration tests to use new constructor signature

## Decisions Made

**Immutable detector pattern:** Version detection is a query operation - it should not have side effects on the config. The detector returns the detected version, and the caller (performVersionDetectionIfNeeded) is responsible for applying it to the config after successful detection.

**Inline header construction:** Instead of relying on client.getHeaders() which reads from mutable client.cfg state, makeVersionTestRequest() builds headers inline with the test version. This ensures the request is truly immutable and doesn't depend on client configuration state.

**Single mutation point:** Config mutation happens in exactly one place - performVersionDetectionIfNeeded() after DetectVersion() succeeds. If detection fails or context is cancelled, config remains unchanged. This eliminates the BUG-01 state restoration problem.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None - refactoring proceeded smoothly. All existing tests passed after updating constructor signatures.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Version detection is now immutable and safe for concurrent use
- Context cancellation cannot leave config in inconsistent state (BUG-01 fixed)
- Shared config reference issue eliminated (FRAG-01 fixed)
- Ready for resource cleanup implementation (Plan 01-03)
- All existing tests pass with race detection

---

_Phase: 01-critical-fixes-stability_
_Completed: 2026-01-23_
