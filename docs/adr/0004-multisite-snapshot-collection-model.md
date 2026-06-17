# ADR-0004: Multi-site support via the snapshot collection model and a `site` identity label

- **Status:** Accepted (implemented)
- **Date:** 2026-06-17
- **Deciders:** Frederic Jacquet
- **Related:** `docs/superpowers/specs/2026-06-17-feature1-multisite-design.md`,
  `docs/superpowers/specs/2026-06-16-nbu-feature-deferrals.md`, ADR-0002, `exporter-standards`
  (`pstore` 0005 identity label; `ppdd` 0001 / `pstore` 0002 snapshot model; `pstore` 0007
  serve-before-first-collect; `ppdd` 0006 label-key consistency)

## Context

`nbu_exporter` is **fetch-on-scrape**: one `NbuClient`, one `cfg`, one storage cache;
`NbuCollector.Collect` fetches storage + jobs live on every Prometheus scrape, and a single
`nbuserver:` block is configured. Operators run **multiple NetBackup primary servers (one per
site)** and want a single exporter instance that labels metrics by site.

The exporter family's canonical answer for multi-backend is **"one process, many backends +
an identity label,"** driven by a **background snapshot collection loop** (not fetch-on-scrape)
— see `exporter-standards`. `nbu` is the only family member still on the fetch-on-scrape
model. Done naïvely (fetch-on-scrape × N masters), backend API load would scale with scrape
frequency and a single slow master could stall the whole `/metrics` response.

## Decision

Adopt the family snapshot model and multi-target configuration:

- **Snapshot loop:** a single background loop polls **every** configured master on
  `collectionInterval` (**default `5m`**, matching today's storage-cache TTL), builds an
  **immutable snapshot**, and atomic-pointer-swaps it into a `SnapshotStore`. The Prometheus
  collector's `Collect` reads the latest snapshot — no live fetch on scrape.
- **Config:** a `nbuservers:` array, each entry with a **required, unique `site`** plus the
  existing per-server fields. A legacy single `nbuserver:` block is **auto-mapped** to a
  one-entry list (`site` defaults to the host) with a deprecation warning — existing
  single-site configs keep working.
- **Identity label:** every metric series carries a **`site`** label (first in the label set),
  emitted via the shared label builders and obeying the label-key consistency invariant
  (`ppdd` 0006).
- **Per-target isolation:** each target has its own client, version detector, and cache (the
  snapshot store subsumes the per-scrape storage cache). A failed target emits
  `nbu_up{site=…}=0` and never aborts the cycle or affects other sites (graceful degradation).
- **Serve HTTP before the first collection** completes (`pstore` 0007), so `/metrics` is up
  immediately.

## Consequences

**Positive**

- One exporter instance serves N masters; backend API load is decoupled from scrape
  frequency; per-site fault isolation; brings `nbu` onto the family architecture.
- The `site` label makes multi-site dashboards/alerts a single Grafana view (a `site`
  template variable — see the dashboards spec).

**Negative / trade-offs**

- A **significant refactor** of `NbuCollector` (snapshot reader) and `main.go` (background
  loop + serve-before-collect), plus threading `site` into every `Desc`/metric key.
- `site` multiplies series count by the number of sites (bounded and intended).
- Config shape change, mitigated by the legacy auto-map so it is not a hard break.
