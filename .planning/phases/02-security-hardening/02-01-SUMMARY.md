---
phase: 02-security-hardening
plan: 01
subsystem: security
tags: [tls, api-key, authentication, security-logging, environment-variables]

# Dependency graph
requires:
  - phase: 01-critical-fixes-stability
    provides: Stable codebase with immutable version detection and resource cleanup
provides:
  - TLS enforcement with explicit opt-in for insecure mode via NBU_INSECURE_MODE env var
  - TLS 1.2 minimum version enforced in HTTP client
  - API key protection verified - no exposure in error messages or logs
  - Enhanced security logging with Error-level warnings for insecure TLS
affects: [deployment, configuration, security-audits]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Environment variable gates for security-sensitive configuration
    - Error-level logging for security warnings (not Warn)
    - Security documentation in code comments

key-files:
  created: []
  modified:
    - internal/models/Config.go
    - internal/models/Config_test.go
    - internal/exporter/client.go
    - internal/exporter/client_test.go

key-decisions:
  - "TLS insecure mode requires NBU_INSECURE_MODE=true environment variable for explicit opt-in"
  - "TLS 1.2 is minimum supported version (industry standard, blocks older protocols)"
  - "Security warnings log at Error level (not Warn) for better visibility"
  - "API key protection verified via comprehensive audit - no code changes needed"

patterns-established:
  - "Security-sensitive config settings require environment variable confirmation"
  - "Document security responsibilities in code comments (getHeaders SECURITY note)"
  - "Use MaskAPIKey() for any debugging context that might reference the API key"

# Metrics
duration: 5min
completed: 2026-01-23
---

# Phase 2 Plan 1: TLS Enforcement & API Key Protection Summary

**TLS 1.2 minimum enforced, insecure mode requires NBU_INSECURE_MODE=true environment variable, API key protection audit verified zero leaks**

## Performance

- **Duration:** 5 min
- **Started:** 2026-01-23T05:02:48Z
- **Completed:** 2026-01-23T05:08:07Z
- **Tasks:** 3
- **Files modified:** 4

## Accomplishments

- TLS verification disabled only with explicit NBU_INSECURE_MODE=true environment variable - prevents accidental production deployment with insecure settings
- TLS 1.2 enforced as minimum version in HTTP client - blocks deprecated TLS 1.0/1.1 protocols
- Comprehensive API key protection audit completed - verified API key never appears in error messages, logs, or span attributes
- Security warnings upgraded to Error level for better visibility in production logs

## Task Commits

Each task was committed atomically:

1. **Task 1: Add TLS enforcement validation to Config** - `6c318ae` (feat)
2. **Task 2: Configure TLS 1.2 minimum and enhance security logging** - `350ff1f` (feat)
3. **Task 3: Audit and fix API key exposure in error messages** - `0548992` (audit)

## Files Created/Modified

- `internal/models/Config.go` - Added validateTLSConfig() method requiring NBU_INSECURE_MODE env var for InsecureSkipVerify=true
- `internal/models/Config_test.go` - Added tests for TLS enforcement: RequiresEnvVar, WithEnvVar, SecureByDefault
- `internal/exporter/client.go` - Set TLS MinVersion to 1.2, upgraded security warning to Error level, added SECURITY comment to getHeaders()
- `internal/exporter/client_test.go` - Added TestNewNbuClient_TLSConfig verifying secure and insecure configurations

## Decisions Made

**1. TLS enforcement via environment variable gate**

- Rationale: Requires explicit action to enable insecure mode, preventing accidental production deployment with TLS verification disabled
- Implementation: Config.Validate() checks for NBU_INSECURE_MODE=true when InsecureSkipVerify=true
- Impact: Deployment documentation must mention this requirement for development/testing environments

**2. TLS 1.2 minimum version**

- Rationale: Industry standard, TLS 1.0 and 1.1 deprecated (RFC 8996), most NetBackup deployments use 1.2+
- Implementation: Set MinVersion in resty TLS config
- Impact: Very old NetBackup servers (pre-2014) may not connect (acceptable trade-off for security)

**3. Error-level security warnings**

- Rationale: Warn-level logs often filtered in production; Error level ensures visibility in log aggregation systems
- Implementation: Changed log.Warn to log.Error for InsecureSkipVerify warning
- Impact: Better visibility for security audits and compliance checks

**4. API key protection audit approach**

- Rationale: Comprehensive grep audit more reliable than manual review for detecting leaks
- Implementation: Verified API key only appears in Authorization headers (required for auth) and parameter passing
- Impact: Zero API key exposure confirmed in error messages, logs, and telemetry

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None - all tasks completed as planned with tests passing.

## User Setup Required

**For development/testing environments only:**

If you need to disable TLS verification (not recommended for production):

```bash
export NBU_INSECURE_MODE=true
```

Without this environment variable, the exporter will refuse to start if `insecureSkipVerify: true` is set in config.yaml.

**Production environments:**

- Use valid TLS certificates
- Keep `insecureSkipVerify: false` (default)
- No environment variable needed

## Next Phase Readiness

**Ready for Phase 2 Plan 2 (Authentication Improvements):**

- TLS enforcement provides foundation for secure authentication
- API key protection audit establishes baseline for future security enhancements
- Environment variable gate pattern can be reused for other security-sensitive settings

**No blockers identified.**

---

_Phase: 02-security-hardening_
_Completed: 2026-01-23_
