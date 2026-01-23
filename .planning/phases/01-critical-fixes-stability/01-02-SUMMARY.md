---
phase: 01-critical-fixes-stability
plan: 02
subsystem: config
tags: [url-validation, config, error-handling, go-stdlib]

# Dependency graph
requires: []
provides:
  - URL validation during config initialization
  - validateNBUBaseURL() method for NBU server URL validation
  - Clear error messages for malformed URLs at startup
affects: [config, error-handling, startup-validation]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "URL validation during config initialization before use"
    - "Defensive programming: validate input early, fail fast with clear errors"

key-files:
  created: []
  modified:
    - internal/models/Config.go
    - internal/models/Config_test.go

key-decisions:
  - "Keep BuildURL() signature unchanged for backward compatibility"
  - "Validate URL during Config.Validate() instead of at BuildURL() time"
  - "Document BuildURL() assumption of validated config"

patterns-established:
  - "Config validation: validate composed URLs, not just individual components"
  - "Test pattern: separate URL validation tests from BuildURL functionality tests"

# Metrics
duration: 10min
completed: 2026-01-23
---

# Phase 01 Plan 02: URL Validation Summary

**Config URL validation catches malformed URLs at startup with clear error messages instead of silently failing in BuildURL**

## Performance

- **Duration:** 10 min
- **Started:** 2026-01-23T03:20:37Z
- **Completed:** 2026-01-23T03:30:16Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- Invalid NBU server URLs now caught during config validation with descriptive errors
- BuildURL() documented to assume validated config (maintains backward compatibility)
- Comprehensive test coverage for URL validation scenarios

## Task Commits

Each task was committed atomically:

1. **Task 1: Add URL validation to Config.Validate()** - `8fd84ed` (feat)
2. **Task 2: Add comprehensive URL validation tests** - `faf2b50` (test)

## Files Created/Modified
- `internal/models/Config.go` - Added validateNBUBaseURL() method, updated Validate() and BuildURL() documentation
- `internal/models/Config_test.go` - Added TestConfigValidateInvalidURL and TestConfigBuildURLAfterValidation, fixed broken test YAML

## Decisions Made
- **Keep BuildURL() signature unchanged:** Maintains backward compatibility. Since Config.Validate() is called at startup before BuildURL() is used, URL validation happens early enough to catch errors.
- **Validate during Config.Validate() not BuildURL():** Fail fast at startup rather than during API calls. Provides better error messages and clearer failure point.
- **Document BuildURL() assumption:** Clear documentation that BuildURL() assumes validated config helps future developers understand the contract.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed broken test YAML with literal string constants**
- **Found during:** Task 1 (running tests after adding URL validation)
- **Issue:** TestConfigBackwardCompatibility used literal strings like "testServerNBUMaster" instead of actual values ("nbu-master"), creating invalid URLs like "https://testServerNBUMaster:1556testPathNetBackup"
- **Fix:** Replaced literal constant names with actual values in YAML strings
- **Files modified:** internal/models/Config_test.go
- **Verification:** TestConfigBackwardCompatibility now passes
- **Committed in:** 8fd84ed (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** Bug fix was necessary for tests to pass with new URL validation. No scope creep.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- URL validation complete and tested
- Config validation now catches malformed URLs early
- Ready for additional config improvements or security hardening

**Blockers:** None

**Concerns:** None

---
*Phase: 01-critical-fixes-stability*
*Completed: 2026-01-23*
