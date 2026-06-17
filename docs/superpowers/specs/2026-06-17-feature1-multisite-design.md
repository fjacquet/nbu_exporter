# Feature 1 — Multi-Site / Multi-Master Support — Design

**Date:** 2026-06-17
**Status:** Draft for review (direction approved; see deferrals doc)
**Scope:** Let one exporter instance scrape multiple NetBackup primary servers (one per
site), labelling every metric with a `site` identity — by adopting the family **snapshot
collection model**. Keystone for the multi-site request; per-client (Feature 3) and tape
(Feature 2) layer on top.
**Direction source:** [`2026-06-16-nbu-feature-deferrals.md`](2026-06-16-nbu-feature-deferrals.md)
(option (b), family standard) and the `exporter-standards` skill (`architecture.md`:
"one process, many backends" + identity label; snapshot model — `pstore` ADR-0005,
`ppdd` ADR-0001 / `pstore` ADR-0002).

## Background — why this is a refactor, not a label

Today `NbuCollector` is a **fetch-on-scrape** collector: one `client`, one `cfg`, one
`storageCache`; `Collect` calls `collectAllMetrics` (errgroup, live storage+jobs fetch) on
every Prometheus scrape (`internal/exporter/prometheus.go`). `main.go` registers exactly one
collector against one registry and serves it via `promhttp`. The config exposes a single
`nbuserver:` block.

The family standard says multi-backend support is **one process, many backends + an identity
label**, driven by a background **snapshot loop** (not fetch-on-scrape). nbu is the only
family member still on fetch-on-scrape (`architecture.md` flags this). So Feature 1 = bring
nbu onto that architecture, which also fixes a latent problem: with N masters, fetch-on-scrape
would multiply backend API load by the scrape frequency and let one slow master stall the
whole `/metrics` response.

## Goals

- `nbuservers:` — a list of primary servers, each with a required `site` identity.
- A single background collection loop polls **every** server on `server.collectionInterval` and
  publishes an **immutable snapshot**; `Collect` reads the latest snapshot (no live fetch).
- Every metric carries a `site` label (constant per target, emitted on every series).
- Per-target graceful degradation: a failed target emits `nbu_up{site=…}=0` and does not
  affect other targets or fail the scrape.
- Per-target API-version detection and storage cache (the snapshot store subsumes the cache).
- Serve `/metrics` before the first collection completes (`pstore` ADR-0007).

## Non-goals

- No new business metrics (tape/per-client are Features 2/3).
- No change to existing metric **names** or their non-`site` labels (only the `site` label is
  added — see Migration for the label-cardinality note).

## Design

### Config schema (`internal/models/Config.go`)

Replace the single `nbuserver:` block with a list, and add a top-level collection interval:

```yaml
server:
  collectionInterval: "5m"   # how often the loop polls every target (new; default 5m, matches today's storage-cache TTL)
nbuservers:
  - site: "paris"            # required identity, unique across the list
    host: "nbu-par.example.com"
    port: "1556"
    scheme: "https"
    apiKey: "..."
    # apiVersion optional -> per-target auto-detect (14.0->13.0->12.0->10.0)
  - site: "lyon"
    host: "nbu-lyon.example.com"
    ...
```

- `Validate()` enforces: ≥1 entry, each with non-empty unique `site`, and the existing
  per-server checks (host/port/scheme/apiKey, apiVersion format if set).
- **Back-compat:** if a legacy single `nbuserver:` block is present and `nbuservers:` is
  absent, map it to a one-element list with `site` defaulting to the server `host`. Log a
  deprecation warning. (Keeps existing single-site configs working; removes the breaking edge
  for the common case.)

### Snapshot collection model (new: `internal/exporter/snapshot.go`)

- `Snapshot` — an immutable value holding, per site, the already-aggregated results
  (`[]StorageMetricValue`, `[]StorageUnitInfo`, `*JobAggregator`, sub-collector results) plus
  per-site `up`/error state and the detected API version.
- `SnapshotStore` — an `atomic.Pointer[Snapshot]` (RWMutex-free pointer swap); `Load()` for
  readers, `Store()` for the loop.
- `Collector` (background): owns one `targetCollector` per site, each with its own
  `*NbuClient`, version detector, and cache. On each tick it fans out across targets with an
  `errgroup` + `SetLimit(min(len, NumCPU))`, builds a fresh `Snapshot`, and swaps it in.
  Per-target panics/errors are captured into that site's `up=0` + error, never aborting the
  cycle.

### Prometheus collector (`prometheus.go`)

- `NbuCollector` becomes a thin **snapshot reader**: `Collect` does `snap := store.Load()`
  then, for each site in `snap`, emits the existing const metrics with the site's label set.
- Every `MustNewConstMetric` call gains the `site` label value; every `prometheus.NewDesc`
  variable-label list gains `"site"` as the **first** label (label-key consistency invariant,
  `ppdd` ADR-0006 — `site` present on every series of every family). Shared label assembly
  stays centralized (the metric-key `.Labels()` helpers in `metrics.go` prepend `site`).
- `nbu_up` / `nbu_last_scrape_timestamp_seconds` become per-site.

### main.go lifecycle

- Build the background `Collector` from `nbuservers`, start its loop in a goroutine, and
  register a snapshot-reading `NbuCollector`. **Serve HTTP immediately** — before the first
  cycle finishes — so `/metrics` is up at once (returns only `nbu_up{site}=0`/nothing until the
  first snapshot lands). Hot-reload (SIGHUP/fsnotify) rebuilds the target set and swaps.

## Migration & cardinality

- `site` multiplies every series by the number of sites — bounded and intended (2 here).
- Dashboards/alerts gain a `site` template variable (separate dashboard spec:
  [`2026-06-17-multisite-dashboards-design.md`](2026-06-17-multisite-dashboards-design.md)).
- A new ADR records the snapshot-model adoption + identity label (mirrors `pstore` 0002/0005).

## Testing (TDD)

- `SnapshotStore` load/store concurrency (race test); `Snapshot` immutability.
- Collection loop: 2 mock targets → snapshot has both sites; one target failing → that
  site `up=0`, the other still present (graceful degradation).
- `Collect` emits every metric with a `site` label; label-key consistency test (every series
  of a family carries `site`).
- Back-compat: legacy `nbuserver:` config maps to one site (`site=host`) with a deprecation
  warning.
- "Serve before first collect": HTTP `/metrics` responds before the loop's first tick.
- `make ci` green (incl. the 70% coverage gate).

## Acceptance criteria

- One exporter, `nbuservers:` with two entries → `/metrics` shows every series twice, once per
  `site`, and a down master shows only `nbu_up{site=…}=0` without affecting the other site.
- Backend API load is independent of scrape frequency (driven by `collectionInterval`).
- Existing single-site configs keep working (legacy block → one site).
- No metric renamed; only `site` added.

## Decisions (resolved 2026-06-17)

1. **Config back-compat:** **auto-map** the legacy single `nbuserver:` block — when present
   and `nbuservers:` is absent, treat it as a one-entry list with `site` defaulting to the
   server `host`, and log a deprecation warning. Existing single-site configs keep working;
   no forced migration. New configs use `nbuservers:` with an explicit `site`.
2. **`collectionInterval` default:** **`5m`**, matching today's storage-cache TTL so per-master
   API load stays close to current behaviour. Operators can tune it; metrics may be up to one
   interval stale (expected for the snapshot model).
3. **Identity label:** **`site`** (1 master per site is the NetBackup reality), on every
   series, first in the label set.
