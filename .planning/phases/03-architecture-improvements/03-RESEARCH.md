# Phase 3: Architecture Improvements - Research

**Researched:** 2026-01-23
**Domain:** Go architecture patterns (immutability, dependency injection, connection management)
**Confidence:** HIGH (patterns verified against official docs and codebase analysis)

## Summary

This research investigates five architectural improvements for the NBU exporter: config immutability (TD-01), OpenTelemetry dependency injection (TD-02), structured metric keys (TD-03), connection pool lifecycle (FRAG-02), and tracer nil-check centralization (FRAG-04).

Key findings:

1. **Phase 1 already implemented significant portions** of TD-01, TD-02, and FRAG-02. The remaining work is refinement and extension of existing patterns.
2. **Structured metric keys already exist** in `metrics.go` - the issue is the pipe-delimited `String()` method used for map keys, not the types themselves.
3. **OpenTelemetry's noop package** provides the idiomatic pattern for tracer nil-safety, eliminating manual nil-checks entirely.
4. **TracerProvider injection** is the recommended pattern over global state access.

**Primary recommendation:** Leverage OpenTelemetry's noop.TracerProvider as the central pattern - inject TracerProvider into components rather than tracer instances, use noop as default, and eliminate all manual nil-checks.

## Current State Analysis

### TD-01: Config Immutability (Partially Complete)

**What Phase 1 did:**

- `APIVersionDetector` no longer holds `*models.Config` reference
- Stores only immutable values: `baseURL string`, `apiKey string`
- Version detection returns result; caller applies changes in single location

**What remains:**

- `NbuClient` still holds full `models.Config` (mutable struct)
- `NbuCollector` holds `cfg models.Config` (copy, but still mutable)
- Config validation mutates config via `SetDefaults()`

**Current pattern (client.go:62-72):**

```go
type NbuClient struct {
    client *resty.Client
    cfg    models.Config  // Holds copy, but config is mutable struct
    tracer trace.Tracer
    // ... connection tracking fields
}
```

### TD-02: OpenTelemetry Global State (Partially Addressed)

**Current pattern (version_detector.go:67-72):**

```go
// Initialize tracer from global provider if available
var tracer trace.Tracer
tracerProvider := otel.GetTracerProvider()  // GLOBAL STATE ACCESS
if tracerProvider != nil {
    tracer = tracerProvider.Tracer("nbu-exporter/version-detector")
}
```

**Same pattern in:**

- `client.go:127-131` - `NewNbuClient()`
- `prometheus.go:84` - `NewNbuCollector()`
- `main.go:143-148` - W3C propagator setup

**What exists:**

- `telemetry.Manager` handles TracerProvider lifecycle
- `telemetry.Manager.Initialize()` registers global provider
- Components fetch tracer from global after initialization

### TD-03: Metric Key Format

**Current implementation (metrics.go):**

```go
type StorageMetricKey struct {
    Name string
    Type string
    Size string
}

func (k StorageMetricKey) String() string {
    return k.Name + "|" + k.Type + "|" + k.Size  // Pipe delimiter
}
```

**Problem:** If label values contain `|` character, the `strings.Split(key, "|")` in prometheus.go will break:

```go
// prometheus.go:311-318
for key, value := range storageMetrics {
    labels := strings.Split(key, "|")  // BREAKS if value contains "|"
    ch <- prometheus.MustNewConstMetric(
        c.nbuDiskSize, prometheus.GaugeValue, value,
        labels[0], labels[1], labels[2],
    )
}
```

**Note:** The `Labels() []string` method already exists and is NOT used in the current code path.

### FRAG-02: Connection Pool Lifecycle (Mostly Complete)

**Phase 1 implemented:**

- `NbuClient.Close()` with 30-second timeout
- `NbuClient.CloseWithContext()` for custom timeouts
- Active request tracking with atomic counter
- `FetchData()` rejects requests after Close()

**What remains:**

- Server.Shutdown() doesn't call client.Close()
- No integration between NbuCollector and client lifecycle
- No documented cleanup sequence

### FRAG-04: Tracer Nil-Checks

**Current pattern scattered across codebase:**

```go
// tracing.go:28-32 - helper exists
func createSpan(ctx context.Context, tracer trace.Tracer, operation string, kind trace.SpanKind) (context.Context, trace.Span) {
    if tracer == nil {
        return ctx, nil
    }
    return tracer.Start(ctx, operation, trace.WithSpanKind(kind))
}

// But then every call site still needs nil-check on span:
ctx, span := createSpan(ctx, client.tracer, "netbackup.fetch_storage", trace.SpanKindClient)
if span != nil {  // STILL NEEDED
    defer span.End()
}
```

**Nil-checks scattered in:**

- `version_detector.go`: 14 instances of `if span != nil`
- `client.go`: 6 instances
- `prometheus.go`: 8 instances
- `netbackup.go`: 12 instances

## Research Findings

### Pattern 1: Immutable Config in Go

**Sources:** [Functional Options Pattern](https://dev.to/kittipat1413/understanding-the-options-pattern-in-go-390c), [Immutable Data Sharing](https://goperf.dev/01-common-patterns/immutable-data/)

**Recommended approach: Separate read-only config from mutable runtime state**

```go
// ImmutableConfig holds configuration that never changes after startup
type ImmutableConfig struct {
    baseURL     string
    apiKey      string
    apiVersion  string  // Set once during initialization
    // ... other immutable fields
}

// NewImmutableConfig creates config from validated models.Config
func NewImmutableConfig(cfg *models.Config) ImmutableConfig {
    return ImmutableConfig{
        baseURL:    cfg.GetNBUBaseURL(),
        apiKey:     cfg.NbuServer.APIKey,
        apiVersion: cfg.NbuServer.APIVersion,
    }
}
```

**Key insight:** Don't try to make `models.Config` itself immutable - it's a data transfer object from YAML. Create a separate type that extracts needed values after validation.

**Confidence:** HIGH - This is standard Go practice for separating config parsing from runtime use.

### Pattern 2: TracerProvider Dependency Injection

**Sources:** [OpenTelemetry Best Practices](https://github.com/open-telemetry/opentelemetry-go/discussions/4532), [OpenTelemetry Libraries](https://opentelemetry.io/docs/concepts/instrumentation/libraries/)

**Recommended approach: Inject TracerProvider, use noop as default**

```go
// Accept TracerProvider in constructor
func NewNbuClient(cfg models.Config, opts ...ClientOption) *NbuClient {
    options := defaultOptions()
    for _, opt := range opts {
        opt(&options)
    }

    tracer := options.tracerProvider.Tracer("nbu-exporter/http-client")
    // ...
}

// Option pattern for optional TracerProvider
type ClientOption func(*clientOptions)

type clientOptions struct {
    tracerProvider trace.TracerProvider
}

func defaultOptions() clientOptions {
    return clientOptions{
        tracerProvider: noop.NewTracerProvider(),  // Default to noop
    }
}

func WithTracerProvider(tp trace.TracerProvider) ClientOption {
    return func(o *clientOptions) {
        if tp != nil {
            o.tracerProvider = tp
        }
    }
}
```

**Key insight:** With noop as default, you never need nil-checks. The tracer always exists and always produces valid spans.

**Confidence:** HIGH - This is the official OpenTelemetry recommendation.

### Pattern 3: Noop Tracer for Nil-Safety

**Sources:** [noop package](https://pkg.go.dev/go.opentelemetry.io/otel/trace/noop), [trace package](https://pkg.go.dev/go.opentelemetry.io/otel/trace)

**Key discovery: SpanFromContext never returns nil**

From official docs:
> If no Span is currently set in ctx, an implementation of a Span that performs no operations is returned.

**This means:**

1. `trace.SpanFromContext(ctx)` is always safe to call
2. The returned span is always safe to call methods on
3. Manual nil-checks on spans are unnecessary

**Recommended pattern using noop:**

```go
import "go.opentelemetry.io/otel/trace/noop"

// TracerWrapper provides nil-safe tracing
type TracerWrapper struct {
    tracer trace.Tracer
}

func NewTracerWrapper(tp trace.TracerProvider) *TracerWrapper {
    if tp == nil {
        tp = noop.NewTracerProvider()
    }
    return &TracerWrapper{
        tracer: tp.Tracer("nbu-exporter"),
    }
}

// StartSpan always returns a valid span (noop if tracing disabled)
func (w *TracerWrapper) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
    return w.tracer.Start(ctx, name, opts...)
}
```

**Usage without nil-checks:**

```go
ctx, span := wrapper.StartSpan(ctx, "operation", trace.WithSpanKind(trace.SpanKindClient))
defer span.End()  // Always safe, no nil-check needed
span.SetAttributes(...)  // Always safe
span.RecordError(err)  // Always safe
```

**Confidence:** HIGH - Official noop package behavior.

### Pattern 4: Structured Map Keys Without Delimiters

**Sources:** [Prometheus labels.go](https://github.com/prometheus/client_golang/blob/main/prometheus/labels.go), [Prometheus escaping schemes](https://prometheus.io/docs/instrumenting/escaping_schemes/)

**Problem:** Pipe delimiter breaks if label values contain `|`

**Solution options:**

1. **Use the existing Labels() method directly** (Recommended)

```go
// Instead of: map[string]float64 with String() keys
// Use: map[StorageMetricKey]float64 directly

// metrics.go already has this:
func (k StorageMetricKey) Labels() []string {
    return []string{k.Name, k.Type, k.Size}
}

// Change prometheus.go to:
type storageMetric struct {
    key   StorageMetricKey
    value float64
}
storageMetrics := []storageMetric{}
// ...
for _, m := range storageMetrics {
    ch <- prometheus.MustNewConstMetric(
        c.nbuDiskSize, prometheus.GaugeValue, m.value,
        m.key.Labels()...,  // Use Labels() directly
    )
}
```

1. **URL-style encoding for String() method**

```go
func (k StorageMetricKey) String() string {
    // URL encode each field to handle special chars
    return url.QueryEscape(k.Name) + "|" +
           url.QueryEscape(k.Type) + "|" +
           url.QueryEscape(k.Size)
}
```

1. **JSON encoding for String() method**

```go
func (k StorageMetricKey) String() string {
    b, _ := json.Marshal(k)
    return string(b)
}
```

**Recommendation:** Option 1 - use `Labels()` directly with struct slices instead of string-keyed maps.

**Confidence:** HIGH - The Labels() method already exists for exactly this purpose.

### Pattern 5: Connection Pool Lifecycle

**Sources:** [Go transport.go](https://go.dev/src/net/http/transport.go), [HTTP Connection Pooling in Go](https://davidbacisin.com/writing/golang-http-connection-pools-1)

**Current state is mostly complete.** Phase 1 implemented:

- Close() with request draining
- CloseWithContext() for custom timeouts
- FetchData rejection after close

**Remaining work:**

1. **Integrate with Server.Shutdown()**

```go
func (s *Server) Shutdown() error {
    // 1. Stop accepting new scrapes
    // 2. Wait for current scrape to complete
    // 3. Shutdown telemetry
    // 4. Close client connections  <- ADD THIS
    // 5. Stop HTTP server
}
```

1. **Document cleanup sequence**

```
Shutdown Order:
1. Stop HTTP server (no new scrapes)
2. Wait for active Collect() calls
3. Close NbuClient (drains API connections)
4. Shutdown OpenTelemetry (flush traces)
```

**Confidence:** HIGH - Phase 1 laid the groundwork, this is integration work.

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `go.opentelemetry.io/otel/trace/noop` | 1.39.0+ | Nil-safe tracing | Official OTel package for disabled tracing |
| `go.opentelemetry.io/otel/trace` | 1.39.0+ | Tracing interfaces | Standard OTel Go API |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| (existing) | - | No new dependencies needed | - |

**Installation:** No new dependencies required - `noop` is part of existing `go.opentelemetry.io/otel/trace` import.

## Architecture Patterns

### Recommended Project Structure

```
internal/
├── exporter/
│   ├── client.go           # NbuClient with injected TracerProvider
│   ├── collector.go        # NbuCollector with injected TracerProvider
│   ├── metrics.go          # Metric key types (existing)
│   ├── netbackup.go        # API fetching
│   ├── tracing.go          # TracerWrapper centralized helper
│   └── version_detector.go # Immutable detector (from Phase 1)
├── telemetry/
│   ├── manager.go          # TracerProvider lifecycle
│   └── wrapper.go          # NEW: TracerWrapper for nil-safe spans
└── models/
    ├── Config.go           # YAML config (DTO)
    └── immutable.go        # NEW: ImmutableConfig for runtime
```

### Pattern: TracerProvider Injection Flow

```
main.go
   |
   v
telemetry.NewManager() --> telemetry.Manager
   |                            |
   v                            v
Initialize() <----- Gets TracerProvider from manager
   |
   v
exporter.NewNbuCollector(cfg, WithTracerProvider(tp))
   |
   v
exporter.NewNbuClient(cfg, WithTracerProvider(tp))
```

### Anti-Patterns to Avoid

1. **Accessing `otel.GetTracerProvider()` in constructors** - Use injection instead
2. **Manual nil-checks on spans** - Use noop.TracerProvider as default
3. **Mutating config after initialization** - Extract immutable values at startup
4. **String concatenation for metric keys when Labels() exists** - Use struct types with Labels()

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Nil-safe tracing | Manual `if span != nil` checks | `noop.TracerProvider` | Official, tested, zero overhead |
| Tracer lifecycle | Custom nil-aware tracer wrapper | Inject `trace.TracerProvider` | OTel standard pattern |
| Map key escaping | Custom escape functions | Struct slices with `Labels()` | Already exists in codebase |
| Config immutability | Freeze/clone methods | Separate immutable type | Simpler, clearer intent |

**Key insight:** The codebase already has the right abstractions (`Labels()`, `telemetry.Manager`). The issue is inconsistent usage, not missing features.

## Common Pitfalls

### Pitfall 1: Partial Noop Adoption

**What goes wrong:** Mixing noop TracerProvider with manual nil-checks creates confusion
**Why it happens:** Incremental refactoring leaves some code paths with old patterns
**How to avoid:** Adopt noop comprehensively - remove ALL nil-checks when switching
**Warning signs:** Code has both `if span != nil` and noop.TracerProvider usage

### Pitfall 2: Config Mutation After Initialization

**What goes wrong:** Version detection updates config, but other components already copied it
**Why it happens:** Config passed by value creates copies that diverge
**How to avoid:**

- Complete version detection BEFORE creating components
- Use pointer to shared immutable config after finalization
**Warning signs:** Different components report different API versions

### Pitfall 3: Metric Key Collision

**What goes wrong:** Label values containing delimiter character cause incorrect parsing
**Why it happens:** Using `strings.Split()` on user-controlled data
**How to avoid:** Use `Labels()` method directly, never reconstruct from String()
**Warning signs:** Metrics with wrong label values, panics in MustNewConstMetric

### Pitfall 4: Incomplete Shutdown Sequence

**What goes wrong:** Traces lost during shutdown, connections leaked
**Why it happens:** Shutdown order matters - flush traces before closing connections
**How to avoid:** Document and enforce shutdown order:

  1. Stop accepting requests
  2. Wait for active requests
  3. Flush telemetry
  4. Close connections
**Warning signs:** Missing traces at end of runs, resource leaks in tests

## Code Examples

### Example 1: TracerProvider Injection (Recommended Pattern)

```go
// Source: OpenTelemetry Go Contrib instrumentation pattern

// Option pattern for TracerProvider injection
type Option func(*options)

type options struct {
    tracerProvider trace.TracerProvider
}

func defaultOptions() options {
    return options{
        tracerProvider: noop.NewTracerProvider(),
    }
}

func WithTracerProvider(tp trace.TracerProvider) Option {
    return func(o *options) {
        if tp != nil {
            o.tracerProvider = tp
        }
    }
}

// Constructor with injection
func NewNbuClient(cfg ImmutableConfig, opts ...Option) *NbuClient {
    o := defaultOptions()
    for _, opt := range opts {
        opt(&o)
    }

    return &NbuClient{
        cfg:    cfg,
        tracer: o.tracerProvider.Tracer("nbu-exporter/client"),
        // ...
    }
}
```

### Example 2: Noop-Based Tracing (No Nil-Checks)

```go
// Source: go.opentelemetry.io/otel/trace/noop pattern

func (c *NbuClient) FetchData(ctx context.Context, url string, target interface{}) error {
    // No nil-check needed - tracer is always valid (noop if disabled)
    ctx, span := c.tracer.Start(ctx, "http.request",
        trace.WithSpanKind(trace.SpanKindClient),
    )
    defer span.End()  // Always safe

    // All span methods are safe to call
    span.SetAttributes(
        attribute.String("http.url", url),
    )

    resp, err := c.client.R().SetContext(ctx).Get(url)
    if err != nil {
        span.RecordError(err)  // Always safe
        span.SetStatus(codes.Error, err.Error())
        return err
    }

    span.SetAttributes(
        attribute.Int("http.status_code", resp.StatusCode()),
    )
    span.SetStatus(codes.Ok, "")

    return nil
}
```

### Example 3: Metrics Without String Parsing

```go
// Source: Existing metrics.go pattern, properly applied

// Storage metric collection using slice instead of map
type storageMetricValue struct {
    key   StorageMetricKey
    value float64
}

func (c *NbuCollector) collectStorageMetrics() []storageMetricValue {
    var metrics []storageMetricValue
    // ... fetch and populate
    return metrics
}

// Exposition using Labels() directly
func (c *NbuCollector) exposeStorageMetrics(ch chan<- prometheus.Metric, metrics []storageMetricValue) {
    for _, m := range metrics {
        ch <- prometheus.MustNewConstMetric(
            c.nbuDiskSize,
            prometheus.GaugeValue,
            m.value,
            m.key.Labels()...,  // Directly use Labels(), no parsing needed
        )
    }
}
```

### Example 4: Immutable Config Extraction

```go
// Source: Go immutable data patterns

// ImmutableConfig holds values that never change after startup
type ImmutableConfig struct {
    baseURL            string
    apiKey             string
    apiVersion         string
    scrapingInterval   time.Duration
    insecureSkipVerify bool
}

// NewImmutableConfig creates from validated Config
// Called ONCE after validation and version detection complete
func NewImmutableConfig(cfg *models.Config) (ImmutableConfig, error) {
    duration, err := cfg.GetScrapingDuration()
    if err != nil {
        return ImmutableConfig{}, err
    }

    return ImmutableConfig{
        baseURL:            cfg.GetNBUBaseURL(),
        apiKey:             cfg.NbuServer.APIKey,
        apiVersion:         cfg.NbuServer.APIVersion,
        scrapingInterval:   duration,
        insecureSkipVerify: cfg.NbuServer.InsecureSkipVerify,
    }, nil
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `if span != nil` checks | noop.TracerProvider | Always available | Cleaner code, no nil panics |
| Global `otel.GetTracerProvider()` | Inject TracerProvider | OTel best practice | Testable, explicit dependencies |
| Manual nil span handling | SpanFromContext safety | Always true | Never returns nil span |

**Deprecated/outdated:**

- `trace.NewNoopTracerProvider()` - Use `noop.NewTracerProvider()` instead (official deprecation)

## Dependencies & Wave Structure

### Wave Analysis

```
Wave 1 (Independent, can be parallel):
  - TD-03: Structured metric keys (isolated to metrics.go, netbackup.go, prometheus.go)
  - FRAG-04: Tracer nil-checks (can begin with tracing.go wrapper)

Wave 2 (Depends on Wave 1):
  - TD-02: TracerProvider injection (uses wrapper from FRAG-04)

Wave 3 (Depends on Wave 2):
  - TD-01: Immutable config (affects components modified in TD-02)
  - FRAG-02: Connection lifecycle integration (cleanup after other changes)
```

### Recommended Implementation Order

1. **FRAG-04 first** - Create TracerWrapper using noop, establishes pattern
2. **TD-02 second** - Inject TracerProvider using new wrapper
3. **TD-03 third** - Refactor metrics to use Labels() (isolated change)
4. **TD-01 fourth** - Extract ImmutableConfig (after component interfaces stable)
5. **FRAG-02 last** - Integrate shutdown sequence (requires all components)

### Dependency Graph

```
FRAG-04 (Tracer Wrapper)
    |
    v
TD-02 (DI for OTel) --+
    |                  |
    v                  |
TD-01 (Immutable) -----+
    |                  |
    v                  v
FRAG-02 (Lifecycle) <--+

TD-03 (Metric Keys) -- Independent, can be done anytime
```

## Risks & Mitigations

### Risk 1: Breaking Changes to Component Constructors

**Description:** Adding Option parameters changes function signatures
**Probability:** HIGH (intentional change)
**Impact:** LOW (all call sites in same repo)
**Mitigation:** Update all call sites in same commit; ensure tests pass

### Risk 2: Noop Overhead in Hot Path

**Description:** Noop span methods still have function call overhead
**Probability:** LOW (noop is highly optimized)
**Impact:** LOW (tracing path is not performance-critical)
**Mitigation:** Benchmark if concerned; noop is designed for minimal overhead

### Risk 3: Incomplete Migration Leaves Mixed Patterns

**Description:** Half-migrated code is harder to understand than before
**Probability:** MEDIUM (large refactoring)
**Impact:** MEDIUM (confusion, maintenance burden)
**Mitigation:** Complete each TD/FRAG fully before moving to next; no partial commits

### Risk 4: Metric Key Refactor Breaks Dashboards

**Description:** If metric label values change format, existing dashboards break
**Probability:** LOW (format stays same, only implementation changes)
**Impact:** HIGH (user-facing)
**Mitigation:** TD-03 changes ONLY internal map keys, not exposed Prometheus labels

## Open Questions

### Question 1: Collector Lifecycle

**What we know:** NbuCollector creates NbuClient internally
**What's unclear:** Should collector expose client for shutdown?
**Recommendation:** Add `Close()` method to NbuCollector that closes internal client

### Question 2: Config Reload Support

**What we know:** Current design is immutable after startup
**What's unclear:** Future requirement for config reload without restart?
**Recommendation:** Document that config reload requires restart; defer atomic config swap to future phase if needed

## Sources

### Primary (HIGH confidence)

- `go.opentelemetry.io/otel/trace/noop` - [Official noop package](https://pkg.go.dev/go.opentelemetry.io/otel/trace/noop)
- `go.opentelemetry.io/otel/trace` - [Official trace package](https://pkg.go.dev/go.opentelemetry.io/otel/trace)
- OpenTelemetry Go Discussions - [Best practices for instrumenting](https://github.com/open-telemetry/opentelemetry-go/discussions/4532)
- Prometheus client_golang - [labels.go](https://github.com/prometheus/client_golang/blob/main/prometheus/labels.go)

### Secondary (MEDIUM confidence)

- Go performance patterns - [Immutable Data Sharing](https://goperf.dev/01-common-patterns/immutable-data/)
- Go HTTP transport - [transport.go](https://go.dev/src/net/http/transport.go)
- Functional Options Pattern - [DEV Community guide](https://dev.to/kittipat1413/understanding-the-options-pattern-in-go-390c)

### Codebase Analysis (HIGH confidence)

- Phase 1 plans: `.planning/phases/01-critical-fixes-stability/01-01-PLAN.md`
- Phase 1 resource cleanup: `.planning/phases/01-critical-fixes-stability/01-03-PLAN.md`
- Current metrics implementation: `internal/exporter/metrics.go`
- Current tracing helper: `internal/exporter/tracing.go`

## Metadata

**Confidence breakdown:**

- Standard stack: HIGH - Using existing OpenTelemetry packages
- Architecture patterns: HIGH - Verified against official docs and existing code
- Pitfalls: HIGH - Identified from codebase analysis
- Wave structure: MEDIUM - Dependencies clear, but implementation may reveal issues

**Research date:** 2026-01-23
**Valid until:** 2026-02-23 (30 days - stable patterns, no fast-moving dependencies)
