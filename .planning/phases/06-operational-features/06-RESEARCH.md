# Phase 6: Operational Features - Research

**Researched:** 2026-01-23
**Domain:** Prometheus Exporter Operational Patterns (Go)
**Confidence:** HIGH

## Summary

Operational features for Prometheus exporters follow well-established patterns in the Go ecosystem. The research focused on four key areas: TTL-based caching for expensive metrics, health checks with connectivity verification, metric staleness tracking, and dynamic configuration reloading.

The standard approach uses patrickmn/go-cache for TTL caching, the SafeConfig pattern with sync.RWMutex for thread-safe configuration reloading, fsnotify for file watching, and an exporter-specific "up" metric for health tracking. Prometheus provides clear guidance that expensive metrics (>1 minute to collect) should be cached and documented in HELP strings. Configuration reload follows the Prometheus blackbox_exporter pattern: validate first, then atomically swap under write lock.

Metric staleness is handled automatically by Prometheus (5-minute default), but exporters should track collection timestamps internally and expose an "up" metric to signal data freshness. The graceful degradation pattern (expose partial metrics + up=0) is preferred over returning HTTP 5xx errors when possible.

**Primary recommendation:** Use patrickmn/go-cache for storage metric caching, implement SafeConfig pattern with sync.RWMutex for config reload triggered by fsnotify + SIGHUP, add nbu_up metric for health tracking, and track internal collection timestamps for staleness detection.

## Standard Stack

The established libraries/tools for this domain:

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| patrickmn/go-cache | stable (pre-generics) | In-memory TTL cache | Battle-tested, thread-safe, used widely in single-machine exporters |
| fsnotify/fsnotify | v1.9.0 | Cross-platform file watching | Official Go community standard for filesystem notifications |
| os/signal | stdlib | Signal handling (SIGHUP) | Standard library, zero dependencies, canonical for Unix signals |
| sync | stdlib | RWMutex for config protection | Standard library, SafeConfig pattern used by official Prometheus exporters |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| jellydator/ttlcache | v3.x | Generics-based TTL cache | If type safety is critical; newer but less battle-tested |
| sync/atomic | stdlib | Atomic value swapping | Read-heavy config access (alternative to RWMutex) |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| patrickmn/go-cache | jellydator/ttlcache | Generics support, but less mature; benchmarks show similar performance |
| sync.RWMutex | atomic.Value | Better for read-heavy (no locks), but can only swap entire config pointer |
| fsnotify | File polling | Works on all filesystems (NFS/FUSE), but wastes CPU cycles |

**Installation:**
```bash
go get github.com/patrickmn/go-cache
go get github.com/fsnotify/fsnotify
# sync, os/signal are stdlib - no installation needed
```

## Architecture Patterns

### Recommended Project Structure
```
internal/
├── exporter/
│   ├── prometheus.go          # NbuCollector (existing)
│   ├── cache.go               # StorageCache wrapper around go-cache
│   └── health.go              # Health check with NBU connectivity test
├── models/
│   ├── Config.go              # Existing config (add SafeConfig wrapper)
│   └── safe_config.go         # SafeConfig with RWMutex
└── config/
    ├── reload.go              # Config reload logic
    └── watcher.go             # fsnotify file watcher + SIGHUP handler
```

### Pattern 1: TTL Cache for Storage Metrics
**What:** Wrap expensive API calls (FetchStorage) with TTL-based cache to reduce NetBackup API load
**When to use:** When metric collection takes >1 minute OR when upstream API has rate limits
**Example:**
```go
// Source: https://github.com/patrickmn/go-cache + Prometheus best practices
import (
    "github.com/patrickmn/go-cache"
    "time"
)

type StorageCache struct {
    cache *cache.Cache
    cfg   models.Config
}

func NewStorageCache(ttl time.Duration) *StorageCache {
    return &StorageCache{
        cache: cache.New(ttl, ttl*2), // TTL, cleanup interval
    }
}

func (sc *StorageCache) FetchStorageWithCache(ctx context.Context, client *NbuClient) ([]StorageMetricValue, error) {
    // Try cache first
    if cached, found := sc.cache.Get("storage_metrics"); found {
        return cached.([]StorageMetricValue), nil
    }

    // Cache miss - fetch from API
    metrics, err := FetchStorage(ctx, client)
    if err != nil {
        return nil, err
    }

    // Store in cache
    sc.cache.Set("storage_metrics", metrics, cache.DefaultExpiration)
    return metrics, nil
}
```

**Important:** Document in metric HELP string:
```go
nbuDiskSize: prometheus.NewDesc(
    "nbu_disk_bytes",
    "The quantity of storage bytes (cached: 5m TTL)",  // Document caching
    []string{"name", "type", "size"}, nil,
)
```

### Pattern 2: SafeConfig with RWMutex
**What:** Thread-safe configuration wrapper that allows atomic config swapping
**When to use:** For any configuration that can be reloaded at runtime
**Example:**
```go
// Source: https://github.com/prometheus/blackbox_exporter/config
type SafeConfig struct {
    sync.RWMutex
    C *models.Config
}

func (sc *SafeConfig) ReloadConfig(configFile string) error {
    // Validate BEFORE acquiring write lock
    newCfg, err := validateConfig(configFile)
    if err != nil {
        return fmt.Errorf("config validation failed: %w", err)
    }

    // Acquire write lock only for pointer swap
    sc.Lock()
    sc.C = newCfg
    sc.Unlock()

    log.Info("Configuration reloaded successfully")
    return nil
}

func (sc *SafeConfig) Get() *models.Config {
    sc.RLock()
    defer sc.RUnlock()
    return sc.C
}
```

### Pattern 3: File Watcher + SIGHUP Handler
**What:** Monitor config file changes with fsnotify and respond to SIGHUP signal
**When to use:** For production deployments where config changes need zero downtime
**Example:**
```go
// Source: https://github.com/fsnotify/fsnotify + os/signal best practices
import (
    "github.com/fsnotify/fsnotify"
    "os"
    "os/signal"
    "syscall"
)

func WatchConfig(configFile string, reloadFn func(string) error) error {
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        return err
    }
    defer watcher.Close()

    // Watch config file directory (not file itself - editors create temp files)
    configDir := filepath.Dir(configFile)
    if err := watcher.Add(configDir); err != nil {
        return err
    }

    // Setup SIGHUP handler (buffered channel to avoid missing signals)
    sighup := make(chan os.Signal, 1)
    signal.Notify(sighup, syscall.SIGHUP)

    for {
        select {
        case event := <-watcher.Events:
            if event.Has(fsnotify.Write) && event.Name == configFile {
                log.Info("Config file changed, reloading...")
                if err := reloadFn(configFile); err != nil {
                    log.Errorf("Config reload failed: %v", err)
                }
            }
        case <-sighup:
            log.Info("SIGHUP received, reloading config...")
            if err := reloadFn(configFile); err != nil {
                log.Errorf("Config reload failed: %v", err)
            }
        case err := <-watcher.Errors:
            log.Errorf("File watcher error: %v", err)
        }
    }
}
```

### Pattern 4: Exporter Health with Up Metric
**What:** Expose nbu_up metric to signal NetBackup connectivity + health endpoint
**When to use:** Always - this is Prometheus best practice for exporters
**Example:**
```go
// Source: https://prometheus.io/docs/instrumenting/writing_exporters/
type NbuCollector struct {
    // ... existing fields
    nbuUp *prometheus.Desc
}

func NewNbuCollector(cfg models.Config, opts ...CollectorOption) (*NbuCollector, error) {
    // ... existing code
    return &NbuCollector{
        // ... existing fields
        nbuUp: prometheus.NewDesc(
            "nbu_up",
            "1 if NetBackup API is reachable, 0 otherwise",
            nil, nil,
        ),
    }
}

func (c *NbuCollector) Collect(ch chan<- prometheus.Metric) {
    // Try to collect metrics
    storageMetrics, jobsSize, jobsCount, jobsStatusCount, storageErr, jobsErr := c.collectAllMetrics(ctx, span)

    // Determine up status (1 if any collection succeeded, 0 if all failed)
    upValue := 0.0
    if storageErr == nil || jobsErr == nil {
        upValue = 1.0
    }

    // Expose up metric first
    ch <- prometheus.MustNewConstMetric(c.nbuUp, prometheus.GaugeValue, upValue)

    // Continue to expose partial metrics even if some failed (graceful degradation)
    c.exposeMetrics(ch, storageMetrics, jobsSize, jobsCount, jobsStatusCount)
}

// Health endpoint verifies connectivity
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
    // Test NetBackup connectivity
    ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
    defer cancel()

    if err := s.collector.TestConnectivity(ctx); err != nil {
        w.WriteHeader(http.StatusServiceUnavailable)
        fmt.Fprintf(w, "UNHEALTHY: %v\n", err)
        return
    }

    w.WriteHeader(http.StatusOK)
    fmt.Fprintf(w, "OK\n")
}
```

### Pattern 5: Staleness Tracking with Last Collection Time
**What:** Track internal timestamps for last successful collection, expose as metric
**When to use:** When cache TTL is long and consumers need to know data freshness
**Example:**
```go
// Source: Prometheus staleness best practices
type StorageCache struct {
    cache           *cache.Cache
    lastCollectionTime time.Time
    lastCollectionMutex sync.RWMutex
}

// Add metric descriptor
nbuLastScrapeTime: prometheus.NewDesc(
    "nbu_last_scrape_timestamp_seconds",
    "Unix timestamp of last successful scrape",
    []string{"source"}, nil,
)

func (sc *StorageCache) FetchStorageWithCache(ctx context.Context, client *NbuClient) ([]StorageMetricValue, error) {
    // ... cache logic

    // On successful fetch, update timestamp
    sc.lastCollectionMutex.Lock()
    sc.lastCollectionTime = time.Now()
    sc.lastCollectionMutex.Unlock()

    return metrics, nil
}

func (sc *StorageCache) GetLastCollectionTime() time.Time {
    sc.lastCollectionMutex.RLock()
    defer sc.lastCollectionMutex.RUnlock()
    return sc.lastCollectionTime
}

// In Collect()
lastScrape := c.storageCache.GetLastCollectionTime()
if !lastScrape.IsZero() {
    ch <- prometheus.MustNewConstMetric(
        c.nbuLastScrapeTime,
        prometheus.GaugeValue,
        float64(lastScrape.Unix()),
        "storage",
    )
}
```

### Anti-Patterns to Avoid
- **Cache without TTL:** Leads to stale data that never refreshes. Always use expiration.
- **Watch individual files:** Editors use atomic writes (temp file → rename), breaking file watchers. Watch directory instead.
- **Mutexes without defer:** Panics can leave locks held forever. Always use `defer mu.Unlock()`.
- **Explicit metric timestamps:** Disables Prometheus staleness handling. Let Prometheus manage timestamps.
- **HTTP 5xx on partial failure:** Lose all metrics when one source fails. Use up=0 + partial metrics instead.
- **Config reload without validation:** Broken config can crash exporter. Validate before applying.

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| TTL-based cache | Custom map + goroutine cleanup | patrickmn/go-cache | Thread-safety, proven expiration logic, edge case handling (concurrent access during cleanup) |
| File watching | Polling with time.Ticker | fsnotify | Cross-platform, efficient (OS events vs polling), handles edge cases (atomic writes) |
| Thread-safe config | Custom mutex wrapper | SafeConfig pattern (blackbox_exporter) | Proven pattern from official exporters, handles validation-before-swap correctly |
| Signal handling | Custom os.Signal channels | os/signal.Notify with buffered channels | Handles signal coalescing, prevents missed signals, stdlib guarantees |
| Metric staleness | Manual timestamp tracking | Prometheus automatic staleness (5m) | Prometheus handles this natively, explicit timestamps cause problems |

**Key insight:** Prometheus exporters have well-established patterns. Deviating from these patterns causes operational surprises (missed signals, stale locks, incorrect staleness handling). Use proven libraries and patterns from official Prometheus exporters.

## Common Pitfalls

### Pitfall 1: File Watcher on Config File Directly
**What goes wrong:** Config file watcher stops working after first edit. `vim`, `nano`, and most editors use atomic writes: write to temp file, rename to target. Watcher on original file loses the inode reference.
**Why it happens:** Misunderstanding of how editors save files vs how inotify works.
**How to avoid:** Watch the directory containing the config file, filter events by filename.
**Warning signs:** Config reload works once after startup, then never again. No errors in logs.

### Pitfall 2: RWMutex Read Lock During Config Reload
**What goes wrong:** Deadlock when config reload needs write lock while many collectors hold read locks during long scrapes.
**Why it happens:** Write lock waits for all read locks to release. If scrape takes 2 minutes and reload happens mid-scrape, reload blocks.
**How to avoid:** Keep read lock duration minimal. Use pointer swapping - read lock only to get pointer, not during entire scrape.
**Warning signs:** Config reload hangs indefinitely. `go tool trace` shows goroutines blocked on mutex.

### Pitfall 3: Cache TTL Longer Than Prometheus Staleness Marker (5m)
**What goes wrong:** Prometheus marks metrics as stale (after 5 minutes) while cache still has "fresh" data. Next scrape returns cached data, but Prometheus already dropped the series.
**Why it happens:** Prometheus staleness (5m default) is independent of exporter cache TTL.
**How to avoid:** Either keep cache TTL < 5 minutes, OR expose `nbu_last_scrape_timestamp_seconds` so users can detect staleness.
**Warning signs:** Metrics disappear from Prometheus, then reappear. Gaps in time series graphs.

### Pitfall 4: Unbuffered Signal Channel
**What goes wrong:** SIGHUP sent while handler is busy (during config reload) gets dropped. Signal lost, config never reloads.
**Why it happens:** `signal.Notify()` with unbuffered channel can't queue signals. If receiver isn't ready, signal is lost.
**How to avoid:** Use buffered channel: `make(chan os.Signal, 1)`. This queues one signal.
**Warning signs:** `kill -HUP <pid>` sometimes works, sometimes doesn't. No error message.

### Pitfall 5: Validation After Write Lock
**What goes wrong:** Invalid config acquired write lock, then validation fails. But old config is already overwritten or partially modified. Exporter left in broken state.
**Why it happens:** Optimistic locking - trying to reduce lock duration by validating under lock.
**How to avoid:** Always validate config BEFORE acquiring write lock. Only use write lock for pointer swap (nanoseconds).
**Warning signs:** Bad config reload leaves exporter broken (won't scrape). Requires restart to recover.

## Code Examples

Verified patterns from official sources:

### Cache Expensive Metrics (Prometheus Official Guidance)
```go
// Source: https://prometheus.io/docs/instrumenting/writing_exporters/
// "If a metric is particularly expensive to retrieve (i.e. takes more than a minute),
//  it is acceptable to cache it. This should be noted in the HELP string."

type StorageCache struct {
    cache    *cache.Cache
    client   *NbuClient
    cacheTTL time.Duration
}

func NewStorageCache(ttl time.Duration, client *NbuClient) *StorageCache {
    return &StorageCache{
        cache:    cache.New(ttl, ttl*2),
        client:   client,
        cacheTTL: ttl,
    }
}

func (sc *StorageCache) Get(ctx context.Context) ([]StorageMetricValue, error) {
    // Check cache first
    if cached, found := sc.cache.Get("metrics"); found {
        log.Debug("Cache hit for storage metrics")
        return cached.([]StorageMetricValue), nil
    }

    // Cache miss - fetch from NetBackup API
    log.Debug("Cache miss, fetching from NetBackup API")
    metrics, err := FetchStorage(ctx, sc.client)
    if err != nil {
        return nil, err
    }

    // Update cache
    sc.cache.Set("metrics", metrics, cache.DefaultExpiration)
    return metrics, nil
}

// Document caching in metric descriptor
prometheus.NewDesc(
    "nbu_disk_bytes",
    fmt.Sprintf("The quantity of storage bytes (cached: %v TTL)", cacheTTL),
    []string{"name", "type", "size"}, nil,
)
```

### SafeConfig Pattern (Prometheus Blackbox Exporter)
```go
// Source: https://pkg.go.dev/github.com/prometheus/blackbox_exporter/config
type SafeConfig struct {
    sync.RWMutex
    C *models.Config
}

func NewSafeConfig() *SafeConfig {
    return &SafeConfig{
        C: &models.Config{},
    }
}

func (sc *SafeConfig) ReloadConfig(configFile string) error {
    // Step 1: Validate config WITHOUT holding any locks
    newCfg, err := validateConfig(configFile)
    if err != nil {
        return fmt.Errorf("config validation failed: %w", err)
    }

    // Step 2: Acquire write lock ONLY for pointer swap
    sc.Lock()
    sc.C = newCfg
    sc.Unlock()

    log.Info("Configuration reloaded successfully")
    return nil
}

func (sc *SafeConfig) Get() *models.Config {
    sc.RLock()
    defer sc.RUnlock()
    return sc.C
}
```

### fsnotify File Watcher (Official Pattern)
```go
// Source: https://github.com/fsnotify/fsnotify v1.9.0
import (
    "github.com/fsnotify/fsnotify"
    "path/filepath"
)

func watchConfigFile(configFile string, reloadFn func(string) error) error {
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        return err
    }
    defer watcher.Close()

    // IMPORTANT: Watch directory, not file (atomic write compatibility)
    configDir := filepath.Dir(configFile)
    if err := watcher.Add(configDir); err != nil {
        return err
    }

    log.Infof("Watching config directory: %s", configDir)

    for {
        select {
        case event := <-watcher.Events:
            // Filter for our specific config file and Write events
            if event.Name == configFile && event.Has(fsnotify.Write) {
                log.Info("Config file changed, reloading...")
                if err := reloadFn(configFile); err != nil {
                    log.Errorf("Config reload failed: %v", err)
                }
            }
        case err := <-watcher.Errors:
            log.Errorf("File watcher error: %v", err)
        }
    }
}
```

### SIGHUP Signal Handler (Go Best Practice)
```go
// Source: https://pkg.go.dev/os/signal + https://rossedman.io/blog/computers/hot-reload-sighup-with-go/
import (
    "os"
    "os/signal"
    "syscall"
)

func setupSignalHandler(reloadFn func() error) {
    // Buffered channel prevents signal loss
    sighup := make(chan os.Signal, 1)
    signal.Notify(sighup, syscall.SIGHUP)

    go func() {
        for {
            <-sighup
            log.Info("SIGHUP received, reloading configuration...")
            if err := reloadFn(); err != nil {
                log.Errorf("Configuration reload failed: %v", err)
            } else {
                log.Info("Configuration reloaded successfully")
            }
        }
    }()
}

// Usage in main()
setupSignalHandler(func() error {
    return safeConfig.ReloadConfig(configFile)
})
```

### Up Metric for Health Tracking (Prometheus Pattern)
```go
// Source: https://prometheus.io/docs/instrumenting/writing_exporters/
// Pattern: "myexporter_up variable (e.g. haproxy_up) with a value of 0 or 1 depending on
//           whether the scrape worked. The latter is better where there's still some useful
//           metrics you can get even with a failed scrape."

func (c *NbuCollector) Collect(ch chan<- prometheus.Metric) {
    storageMetrics, jobsMetrics, storageErr, jobsErr := c.collectAllMetrics(ctx)

    // Determine up status: 1 if ANY collection succeeded, 0 if ALL failed
    upValue := 0.0
    if storageErr == nil || jobsErr == nil {
        upValue = 1.0
    }

    // Expose up metric FIRST (convention)
    ch <- prometheus.MustNewConstMetric(
        c.nbuUp,
        prometheus.GaugeValue,
        upValue,
    )

    // Graceful degradation: expose whatever metrics we successfully collected
    if storageErr == nil {
        c.exposeStorageMetrics(ch, storageMetrics)
    }
    if jobsErr == nil {
        c.exposeJobMetrics(ch, jobsMetrics)
    }
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Polling config file every N seconds | fsnotify + SIGHUP signal | fsnotify v1.0 (2015) | Instant reload, no CPU waste, works with all editors |
| sync.Mutex for config | sync.RWMutex (SafeConfig pattern) | Prometheus 2.x era (2017+) | Parallel reads during scrapes, write lock only for reload |
| Custom TTL cache | patrickmn/go-cache or generics-based ttlcache | go-cache: 2012, ttlcache v3: 2021 | Thread-safe, battle-tested cleanup, generics type safety |
| Return 5xx on failed scrape | Return 200 + up=0 + partial metrics | Prometheus best practices (2018+) | Better observability during degraded state |
| Explicit metric timestamps | Let Prometheus manage staleness | Prometheus 2.0 (2017) | Avoids staleness handling conflicts, simpler code |

**Deprecated/outdated:**
- **Polling for config changes:** Wastes CPU, introduces latency (up to polling interval). Use fsnotify.
- **Global config variable without mutex:** Race conditions in production. Use SafeConfig pattern.
- **Metric-level TTL via explicit timestamps:** Breaks Prometheus staleness. Use cache layer instead.
- **howeyc/fsnotify:** Deprecated. Use fsnotify/fsnotify (actively maintained fork).

## Open Questions

Things that couldn't be fully resolved:

1. **Optimal cache TTL for storage metrics**
   - What we know: Prometheus docs say cache if >1 minute, must be <5 minutes (staleness marker)
   - What's unclear: NetBackup storage metrics change frequency varies by environment (SAN vs local disk)
   - Recommendation: Make TTL configurable in config.yaml (default: 5m, range: 1m-4m). Let operators tune based on their NetBackup infrastructure.

2. **Health check timeout for connectivity test**
   - What we know: Should test NetBackup API reachability quickly to avoid blocking health endpoint
   - What's unclear: Acceptable timeout for health checks in production (Kubernetes probes have short timeouts)
   - Recommendation: Use 5-second timeout for health endpoint connectivity check (separate from 2-minute scrape timeout). Document in code.

3. **Config reload partial failure handling**
   - What we know: SafeConfig validates before swapping, but what if collector reconstruction fails?
   - What's unclear: Should we keep old collector instance or fail the reload?
   - Recommendation: Keep old collector on reload failure (don't disrupt running scrapes). Log error, return failure, retry on next SIGHUP.

4. **Cache invalidation on config change**
   - What we know: If NBU server address changes in config, cached metrics are from wrong server
   - What's unclear: Should config reload automatically flush cache?
   - Recommendation: YES - flush cache on config reload if NbuServer.Host or NbuServer.Port changes. Compare old vs new config.

## Sources

### Primary (HIGH confidence)
- [Prometheus: Writing Exporters](https://prometheus.io/docs/instrumenting/writing_exporters/) - Official guidance on caching, up metrics, graceful degradation
- [Prometheus: Jobs and Instances](https://prometheus.io/docs/concepts/jobs_instances/) - Built-in "up" metric behavior
- [pkg.go.dev/github.com/prometheus/blackbox_exporter/config](https://pkg.go.dev/github.com/prometheus/blackbox_exporter/config) - SafeConfig pattern
- [GitHub: fsnotify/fsnotify](https://github.com/fsnotify/fsnotify) - v1.9.0, cross-platform file watching
- [GitHub: patrickmn/go-cache](https://github.com/patrickmn/go-cache) - In-memory TTL cache
- [pkg.go.dev/os/signal](https://pkg.go.dev/os/signal) - Signal handling stdlib
- [pkg.go.dev/sync/atomic](https://pkg.go.dev/sync/atomic) - Atomic operations stdlib

### Secondary (MEDIUM confidence)
- [rossedman.io: Hot Reload with SIGHUP](https://rossedman.io/blog/computers/hot-reload-sighup-with-go/) - SIGHUP pattern examples
- [GitHub: Xeoncross/go-cache-benchmark](https://github.com/Xeoncross/go-cache-benchmark) - Performance comparisons of go-cache
- [Medium: Golang Prometheus Exporter for Timestamped Metrics](https://jboothomas.medium.com/golang-prometheus-exporter-for-timestamped-metrics-b1104c9deacf) - Staleness handling caveats
- [Leapcell: Understanding Atomic Operations in Go](https://leapcell.io/blog/understanding-atomic-operations-in-go-with-sync-atomic) - atomic.Value patterns

### Tertiary (LOW confidence)
- Various blog posts and Stack Overflow discussions on cache libraries - used for ecosystem awareness, not specific recommendations

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - patrickmn/go-cache and fsnotify are industry standards, widely used in production exporters
- Architecture: HIGH - SafeConfig pattern is from official Prometheus exporters, battle-tested at scale
- Pitfalls: HIGH - Based on documented issues in GitHub repos and official documentation warnings

**Research date:** 2026-01-23
**Valid until:** 2026-02-23 (30 days - stable ecosystem, slow-moving patterns)
