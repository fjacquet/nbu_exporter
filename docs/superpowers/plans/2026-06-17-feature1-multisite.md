# Feature 1 — Multi-Site (Snapshot Collection Model) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax. **Dispatch one task per subagent (small), gate each on `make ci`, and commit per task.** Background subagents in this environment stall on a 600s no-progress watchdog for large tasks — if a dispatch stalls, reconcile git state and finish that task inline. Never start the next task before the current one is green + committed.

**Goal:** One exporter instance scrapes multiple NetBackup primary servers (one per site), labelling every metric with `site`, by adopting the family snapshot collection model.

**Architecture:** A background loop polls every configured master on `collectionInterval` (default 5m), builds an immutable `Snapshot`, and atomic-swaps it into a `SnapshotStore`. `NbuCollector.Collect` becomes a snapshot *reader* (no live fetch) and emits each site's metrics with a `site` label. Storage stays offset-paginated, jobs stay cursor-paginated (unchanged from #34).

**Tech Stack:** Go 1.26, prometheus/client_golang, errgroup, testify; `make ci` gate (fmt/vet/lint/test-race/govulncheck + 70% coverage). RTK: use `rtk proxy go test ...` / `rtk proxy make ci`. Commit trailer: `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`.

**Spec:** `docs/superpowers/specs/2026-06-17-feature1-multisite-design.md` · **ADR:** `docs/adr/0004-multisite-snapshot-collection-model.md`

---

## File Structure

- `internal/models/Config.go` — add `NbuServers []NbuServerConfig` + `Server.CollectionInterval`; extract `NbuServerConfig` struct (the current inline `NbuServer` fields + `Site`); auto-map legacy `nbuserver:` in `SetDefaults`; validate unique non-empty `site`.
- `internal/exporter/snapshot.go` *(new)* — `SiteSnapshot`, `Snapshot`, `SnapshotStore` (atomic pointer swap). Pure data + store.
- `internal/exporter/collector_loop.go` *(new)* — `TargetCollector` (one per site: client + detector + cache) and `CollectionLoop` (errgroup fan-out → build `Snapshot` → `store.Store`).
- `internal/exporter/prometheus.go` — `site` prepended to every `*prometheus.Desc` label list; `Collect` reads `SnapshotStore` and emits per-site; `expose*` helpers take a `site` arg.
- `internal/exporter/metrics.go` — unchanged keys; site is prepended at emission (not in `.Labels()`).
- `main.go` — build targets, start the loop, serve HTTP before first collect.
- Docs: `docs/config-examples/`, `README.md`, `CHANGELOG.md`.

**Site-threading convention (used throughout):** the `*Key.Labels()` helpers stay site-free; the `expose*` functions receive `site string` and emit with `withSite(site, key.Labels())`, where `withSite` prepends. Every `NewDesc` variable-label list gets `"site"` as its **first** element.

---

## Task 1: Config — `nbuservers[]` array + `collectionInterval` + legacy auto-map

**Files:** Modify `internal/models/Config.go`; Test `internal/models/Config_test.go`.

- [ ] **Step 1: Write failing tests**

Add to `internal/models/Config_test.go`:

```go
func TestConfigNbuServersAutoMapLegacy(t *testing.T) {
	// Legacy single nbuserver: with no nbuservers[] -> one entry, site defaults to host.
	cfg := createConfigWithAPIVersion("13.0") // existing helper: sets NbuServer.Host=testServerNBUMaster
	if err := cfg.Validate(); err != nil {
		t.Fatalf("legacy config should validate: %v", err)
	}
	if len(cfg.NbuServers) != 1 {
		t.Fatalf("expected 1 auto-mapped server, got %d", len(cfg.NbuServers))
	}
	if cfg.NbuServers[0].Site != cfg.NbuServer.Host {
		t.Errorf("site = %q, want host %q", cfg.NbuServers[0].Site, cfg.NbuServer.Host)
	}
}

func TestConfigNbuServersRequireUniqueSite(t *testing.T) {
	cfg := createConfigWithAPIVersion("13.0")
	cfg.NbuServers = []NbuServerConfig{
		{Site: "paris", Host: "a", Port: "1556", Scheme: "https", APIKey: "k"},
		{Site: "paris", Host: "b", Port: "1556", Scheme: "https", APIKey: "k"},
	}
	if err := cfg.Validate(); err == nil {
		t.Error("duplicate site should fail validation")
	}
}

func TestConfigCollectionIntervalDefault(t *testing.T) {
	cfg := createConfigWithAPIVersion("13.0")
	cfg.SetDefaults()
	if cfg.Server.CollectionInterval != "5m" {
		t.Errorf("CollectionInterval default = %q, want 5m", cfg.Server.CollectionInterval)
	}
}
```

- [ ] **Step 2: Run — expect FAIL**

Run: `rtk proxy go test ./internal/models/ -run 'NbuServers|CollectionInterval' -v`
Expected: FAIL (`NbuServers`/`NbuServerConfig`/`CollectionInterval` undefined).

- [ ] **Step 3: Implement the model change**

In `internal/models/Config.go`: add `CollectionInterval string `+"`yaml:\"collectionInterval\"`"+`` to the `Server` struct; define an exported `NbuServerConfig` struct containing the existing `NbuServer` fields **plus** `Site string `+"`yaml:\"site\"`"+``; add `NbuServers []NbuServerConfig `+"`yaml:\"nbuservers\"`"+`` to `Config`. Keep the legacy inline `NbuServer` field for back-compat.

In `SetDefaults()`: after the existing defaults, add:

```go
	if c.Server.CollectionInterval == "" {
		c.Server.CollectionInterval = "5m"
	}
	// Auto-map a legacy single nbuserver: block to a one-entry nbuservers[] list.
	if len(c.NbuServers) == 0 && c.NbuServer.Host != "" {
		legacy := NbuServerConfig{ /* copy all NbuServer fields */ Site: c.NbuServer.Host}
		c.NbuServers = []NbuServerConfig{legacy}
		// (logging.LogWarn deprecation notice — import the logging pkg or return a flag the caller logs)
	}
```

In `Validate()`: after `SetDefaults()`, validate `len(c.NbuServers) >= 1`, each `Site` non-empty and unique, and run the existing per-server checks (host/port/scheme/apiKey, apiVersion format) over **every** entry (refactor `validateNBUServerConfig` to take an `NbuServerConfig`).

- [ ] **Step 4: Run — expect PASS** (`rtk proxy go test ./internal/models/ -run 'NbuServers|CollectionInterval' -v`).

- [ ] **Step 5: `make ci` checkpoint** — `rtk proxy make ci` green (existing collector code still reads `cfg.NbuServer`; unchanged behaviour).

- [ ] **Step 6: Commit** — `feat(config): add nbuservers[] + collectionInterval; auto-map legacy nbuserver`.

---

## Task 2: `site` label on every metric (single-site value)

**Files:** Modify `internal/exporter/prometheus.go` (Descs + `expose*`); Test `internal/exporter/prometheus_test.go`.

This task adds the label end-to-end while still single-site (site value = `cfg.NbuServers[0].Site`), keeping the tree green before the snapshot model lands.

- [ ] **Step 1: Write failing test** — assert a known metric carries a `site` label:

```go
func TestExposeMetricsCarriesSiteLabel(t *testing.T) {
	// Gather from the registry and assert nbu_up has label site="<configured>".
	// (Mirror the existing gather-based assertions in prometheus_test.go.)
}
```

(Use the existing registry-gather test helper in `prometheus_test.go`; assert `site` is present on `nbu_up`.)

- [ ] **Step 2: Run — expect FAIL.**

- [ ] **Step 3: Implement** — add a helper near the `expose*` funcs:

```go
func withSite(site string, labels []string) []string {
	return append([]string{site}, labels...)
}
```

Prepend `"site"` as the **first** element of every variable-label list in `NewNbuCollector` (e.g. `[]string{"site", "name", "type", "size"}`, `[]string{"site"}` for `nbuUp`, `[]string{"site", "version"}`, etc. — all ~16 Descs). Give `exposeMetrics`/`exposeStorageMetrics`/`exposeStorageUnitMetrics`/`exposeJobAggregateMetrics` a `site string` parameter and wrap every `MustNewConstMetric` label slice with `withSite(site, …)` (and `withSite(site, u.InfoLabels())`, `withSite(site, []string{u.Name, u.Type})`, etc.). In `Collect`, pass `c.cfg.NbuServers[0].Site`.

- [ ] **Step 4: Run** the affected tests; update any existing prometheus_test assertions that build expected label sets to include `site`.

- [ ] **Step 5: `make ci` checkpoint** green.

- [ ] **Step 6: Commit** — `feat(metrics): add site label to every series (single-site)`.

---

## Task 3: `Snapshot` + `SnapshotStore` types

**Files:** Create `internal/exporter/snapshot.go`; Test `internal/exporter/snapshot_test.go`.

- [ ] **Step 1: Write failing test**

```go
func TestSnapshotStoreLoadStore(t *testing.T) {
	var s SnapshotStore
	if s.Load() != nil { t.Fatal("zero store should Load() nil") }
	snap := &Snapshot{Sites: map[string]*SiteSnapshot{"paris": {Up: true}}}
	s.Store(snap)
	got := s.Load()
	if got == nil || !got.Sites["paris"].Up { t.Fatal("Load did not return stored snapshot") }
}
```

- [ ] **Step 2: Run — expect FAIL.**

- [ ] **Step 3: Implement** `internal/exporter/snapshot.go`:

```go
package exporter

import "sync/atomic"

// SiteSnapshot holds one site's already-aggregated collection results.
type SiteSnapshot struct {
	Site          string
	APIVersion    string
	Up            bool
	StorageErr    error
	JobsErr       error
	StorageMetrics []StorageMetricValue
	StorageUnits   []StorageUnitInfo
	JobAgg         *JobAggregator
	LastStorageScrape time.Time
	LastJobsScrape    time.Time
}

// Snapshot is an immutable point-in-time view across all sites.
type Snapshot struct {
	Sites map[string]*SiteSnapshot
}

// SnapshotStore holds the latest Snapshot behind an atomic pointer swap.
type SnapshotStore struct{ p atomic.Pointer[Snapshot] }

func (s *SnapshotStore) Store(snap *Snapshot) { s.p.Store(snap) }
func (s *SnapshotStore) Load() *Snapshot      { return s.p.Load() }
```

(Add the `time` import.)

- [ ] **Step 4: Run — expect PASS.**
- [ ] **Step 5: `make ci`** green.
- [ ] **Step 6: Commit** — `feat(exporter): add immutable Snapshot + atomic SnapshotStore`.

---

## Task 4: Per-target collector + background collection loop

**Files:** Create `internal/exporter/collector_loop.go`; Test `internal/exporter/collector_loop_test.go`.

- [ ] **Step 1: Write failing test** — two mock targets; one healthy, one failing; assert the built snapshot has both sites and the failing one has `Up=false`:

```go
func TestCollectionLoopBuildsPerSiteSnapshot(t *testing.T) {
	// Spin two httptest servers (one returns valid storage/jobs, one returns 500).
	// Build a CollectionLoop with two TargetCollectors and call collectOnce(ctx).
	// Assert store.Load().Sites has both sites; healthy Up=true, failing Up=false;
	// healthy site's JobAgg is non-nil. (Mirror netbackup_test httptest patterns.)
}
```

- [ ] **Step 2: Run — expect FAIL.**

- [ ] **Step 3: Implement** `collector_loop.go`:
  - `TargetCollector` holds `site string`, `client *NbuClient`, `cfg models.Config` (single-server view for that target), `cache *StorageCache`. A `func (tc *TargetCollector) collect(ctx) *SiteSnapshot` runs storage + jobs via the existing `FetchStorageFull` / `FetchAllJobsFull` (reuse `collectAllMetrics`-style errgroup) and sets `Up = storageErr==nil || jobsErr==nil`, capturing errors per source (graceful degradation).
  - `CollectionLoop` holds `[]*TargetCollector`, `store *SnapshotStore`, `interval time.Duration`. `collectOnce(ctx)` fans out across targets with `errgroup` + `SetLimit(min(len, runtime.NumCPU()))`, assembles `Snapshot{Sites: …}`, and calls `store.Store`. `Run(ctx)` calls `collectOnce` immediately, then on a `time.Ticker(interval)` until `ctx.Done()`.

- [ ] **Step 4: Run — expect PASS.**
- [ ] **Step 5: `make ci`** green.
- [ ] **Step 6: Commit** — `feat(exporter): background per-site collection loop building snapshots`.

---

## Task 5: `Collect` reads the snapshot; wire `main.go` (serve-before-collect)

**Files:** Modify `internal/exporter/prometheus.go` (`Collect`, constructor), `main.go`; Tests in both packages.

- [ ] **Step 1: Write failing test** — a snapshot-backed collector emits all sites' metrics:

```go
func TestCollectorEmitsAllSitesFromSnapshot(t *testing.T) {
	// Build a SnapshotStore with two SiteSnapshots (paris, lyon), construct the
	// snapshot-reading NbuCollector over it, gather the registry, and assert
	// nbu_up appears for site="paris" AND site="lyon".
}
```

- [ ] **Step 2: Run — expect FAIL.**

- [ ] **Step 3: Implement** — give `NbuCollector` a `store *SnapshotStore` (drop the live `client`/`storageCache` from the scrape path). `Collect` becomes:

```go
func (c *NbuCollector) Collect(ch chan<- prometheus.Metric) {
	snap := c.store.Load()
	if snap == nil { return } // first cycle not done yet; /metrics is up, just empty
	for site, ss := range snap.Sites {
		c.exposeMetrics(ch, site, ss) // per-site emission using Task 2's site-aware expose*
	}
}
```

Refactor `exposeMetrics` to take `(ch, site string, ss *SiteSnapshot)` and emit `nbu_up{site}` from `ss.Up`, version/last-scrape/storage/jobs from `ss`. In `main.go`: build `[]*TargetCollector` from `cfg.NbuServers`, create `SnapshotStore` + `CollectionLoop`, `go loop.Run(ctx)`, register the snapshot-reading collector, and **start `ListenAndServe` immediately** (do not block on the first collection). Hot-reload rebuilds the loop's targets.

- [ ] **Step 4: Run** the new test + full package; fix call sites.
- [ ] **Step 5: `make ci`** green (this is the largest task — if a subagent stalls mid-way, reconcile and finish inline).
- [ ] **Step 6: Commit** — `feat(exporter): snapshot-reading collector + background loop wired in main`.

---

## Task 6: Docs + config examples + CHANGELOG

**Files:** `docs/config-examples/` (new `config-multisite.yaml`), `README.md`, `CHANGELOG.md`, `docs/config-examples/README.md`.

- [ ] **Step 1:** Add `config-multisite.yaml` showing `server.collectionInterval` + a two-entry `nbuservers:` list with `site:`; document legacy auto-map.
- [ ] **Step 2:** README: note multi-site support + the `site` label. `docs/config-examples/README.md`: add the multi-site example row.
- [ ] **Step 3:** `CHANGELOG.md` `[Unreleased]` → Added: multi-site support (nbuservers[], `site` label, snapshot collection model) referencing ADR-0004.
- [ ] **Step 4:** `make ci` green.
- [ ] **Step 5: Commit** — `docs: multi-site config example + README + changelog`.

---

## Final

- [ ] Whole-implementation review (spec coverage: config array ✓, 5m interval ✓, site label ✓, snapshot model ✓, per-site degradation ✓, serve-before-collect ✓, storage/jobs pagination unchanged ✓).
- [ ] `make ci` green; manual sanity: `config-multisite.yaml` with two mock servers → `/metrics` shows every series per `site`.
- [ ] superpowers:finishing-a-development-branch.

## Acceptance criteria (from spec)

- One exporter + two `nbuservers:` entries → `/metrics` shows every series twice (per `site`); a down master shows only `nbu_up{site=…}=0` without affecting the other site.
- Backend API load driven by `collectionInterval` (5m default), not scrape frequency.
- Legacy single `nbuserver:` config still works (one site, `site`=host).
- No metric renamed; only `site` added. Storage offset + jobs cursor pagination unchanged.
