# Plan 02-02 Summary: Rate Limiting & Retry with Backoff

## Status: Complete

## Changes Made

### Task 1: Configure resty retry with exponential backoff
- Added retry configuration constants: `retryCount=3`, `retryWaitTime=5s`, `retryMaxWaitTime=60s`
- Configured resty client with `SetRetryCount()`, `SetRetryWaitTime()`, `SetRetryMaxWaitTime()`
- Added retry condition for network errors, HTTP 429 (rate limiting), and 5xx (server errors)
- Enabled `AddRetryAfterErrorCondition()` for automatic Retry-After header handling

### Task 2: Configure connection pool for performance
- Added connection pool constants: `maxIdleConns=100`, `maxIdleConnsPerHost=20`, `idleConnTimeout=90s`
- Configured custom `http.Transport` with connection pool settings
- Moved TLS configuration from `SetTLSClientConfig()` to Transport for unified config
- Added `TLSHandshakeTimeout=10s` for explicit handshake timeout

### Task 3: Add tests for retry behavior
- `TestNbuClient_RetryConfiguration`: Verifies retry count (3) and wait times (5s/60s)
- `TestNbuClient_ConnectionPoolConfiguration`: Verifies connection pool settings (100/20/90s)
- `TestNbuClient_TLSInTransport`: Verifies TLS 1.2 minimum in transport for secure/insecure modes

## Files Modified
- `internal/exporter/client.go` - Added retry and connection pool configuration
- `internal/exporter/client_test.go` - Added 3 new tests

## Commits
1. `refactor(02-01): simplify TLS insecure mode to config-only` - Removed over-engineered NBU_INSECURE_MODE env var (user feedback)
2. `feat(02-02): add retry backoff and connection pool tuning` - Implemented retry and connection pool

## Verification
- Build: PASS (`go build ./...`)
- New tests: PASS (3 tests for retry, connection pool, TLS)
- Race detector: Tests pass (linker warning is macOS issue, not code)

## Requirements Addressed
- **SEC-03**: Rate limiting and backoff - HTTP client now handles 429 responses with exponential backoff

## Deviations
- Plan 02-01 simplification: Removed `NBU_INSECURE_MODE` env var requirement per user feedback. The config file option `insecureSkipVerify` is sufficient explicit opt-in.

## Duration
- 10 minutes (including user feedback handling)
