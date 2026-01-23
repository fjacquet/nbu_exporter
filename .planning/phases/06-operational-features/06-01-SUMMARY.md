---
phase: 06
plan: 01
subsystem: exporter
tags: [caching, performance, go-cache, prometheus]

dependency-graph:
  requires:
    - 05 (Performance optimizations foundation)
  provides:
    - StorageCache type for TTL-based caching
    - CacheTTL configuration option
    - Cache-aware storage metric collection
  affects:
    - 06-02 (graceful shutdown improvements)
    - Future operational features

tech-stack:
  added:
    - "github.com/patrickmn/go-cache v2.1.0"
  patterns:
    - TTL-based caching for expensive API calls
    - Cache hit/miss span events for observability
    - Configurable cache duration via YAML

key-files:
  created:
    - internal/exporter/cache.go
    - internal/exporter/cache_test.go
  modified:
    - internal/models/Config.go
    - internal/models/Config_test.go
    - internal/exporter/prometheus.go
    - internal/exporter/prometheus_test.go
    - internal/exporter/otel_integration_test.go
    - go.mod
    - go.sum

decisions:
  - id: cache-library
    decision: "Use patrickmn/go-cache for TTL caching"
    rationale: "Well-tested, zero-dependency, thread-safe cache library"
  - id: cache-default-ttl
    decision: "Default cache TTL is 5 minutes"
    rationale: "Balance between API load reduction and data freshness"
  - id: help-string-documentation
    decision: "Include cache TTL in metric HELP string"
    rationale: "Users see caching behavior when querying metric metadata"

metrics:
  duration: 10m
  completed: 2026-01-23
---

# Phase 6 Plan 1: Storage Metrics Caching Summary

**One-liner:** TTL-based storage metrics caching using go-cache with configurable duration

## What Was Built

### StorageCache Type (cache.go)
- Wraps `patrickmn/go-cache` for TTL-based caching
- `Get()` returns cached metrics if available (cache hit)
- `Set()` stores metrics with configured TTL
- `Flush()` clears cache for config reload scenarios
- `TTL()` returns configured cache duration
- `GetLastCollectionTime()` tracks staleness

### CacheTTL Configuration (Config.go)
- Added `CacheTTL` field to Server struct with yaml tag
- Default value: `5m` (5 minutes)
- Validation ensures valid Go duration format
- `GetCacheTTL()` helper returns parsed duration

### NbuCollector Integration (prometheus.go)
- `storageCache` field added to NbuCollector struct
- Cache initialized in `NewNbuCollector()` with configured TTL
- `collectStorageMetrics()` checks cache before API call
- Cache hit returns instantly without NBU API call
- Cache miss triggers API call and stores result
- OpenTelemetry span events record cache hit/miss
- `GetStorageCache()` method for external cache management

### Metric Documentation
- `nbu_disk_bytes` HELP string now includes: `(cached: {TTL} TTL)`
- Example: `"The quantity of storage bytes (cached: 5m0s TTL)"`

## Test Coverage

- `cache_test.go`: 7 tests covering Get/Set, TTL, expiration, flush
- `prometheus_test.go`: 2 new integration tests for cache
- All existing tests updated with CacheTTL field
- Race detector passes

## Commits

| Hash | Type | Description |
|------|------|-------------|
| 700e5b7 | feat | Add TTL-based StorageCache with go-cache |
| c4447cc | feat | Add CacheTTL configuration field |
| 0c62a1e | feat | Integrate StorageCache with NbuCollector |

## Deviations from Plan

None - plan executed exactly as written.

## Configuration Example

```yaml
server:
  port: "2112"
  host: "0.0.0.0"
  uri: "/metrics"
  scrapingInterval: "5m"
  cacheTTL: "5m"  # Storage metrics cache TTL (default: 5m)
```

## Benefits

1. **Reduced API Load**: Storage metrics fetched once per TTL instead of every scrape
2. **Faster Scrapes**: Cache hits return instantly
3. **Configurable**: Users can tune TTL based on their environment
4. **Observable**: Cache hit/miss events visible in traces
5. **Self-Documenting**: HELP string shows caching behavior

## Next Phase Readiness

Ready for 06-02 (Graceful Shutdown Improvements):
- Cache can be flushed during shutdown
- No blockers identified
