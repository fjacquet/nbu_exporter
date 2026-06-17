# NBU Multi-Site Request — Direction & Deferral Rationale

**Date:** 2026-06-16
**Status:** Accepted direction (informs the maintainer reply to the requester)
**Companion:** [`2026-06-16-nbu-10x-and-roadmap-design.md`](2026-06-16-nbu-10x-and-roadmap-design.md)

## Purpose

The first PR ([Feature 5](2026-06-16-nbu-10x-and-roadmap-design.md)) is deliberately tiny.
This file records **why every other item in the request is out of scope of that PR**, the
direction agreed for each, and what unblocks it — so the scope decision is documented once,
not re-litigated, and so the requester gets a clear "yes, and here is the order."

## Sequencing principle

Three things drive the order, in priority:

1. **A working v3.0 path is a prerequisite for everything.** Until auto-detect works on
   NBU 10.x (Feature 5), every downstream feature would be built and tested against a
   version the operator's appliance does not speak. Feature 5 goes first, alone.
2. **Follow the family standard, don't re-derive it.** Multi-backend support has one
   canonical answer in this exporter family ("one process, many backends" + an identity
   label + the background **snapshot collection model** — `pstore` ADR-0005, `ppdd`
   ADR-0001 / `pstore` ADR-0002). Feature 1 is specified to that standard, not to a
   cheaper per-repo shortcut.
3. **KISS / DRY.** The interim alternative for Feature 1 (multi-target fan-out bolted onto
   the current fetch-on-scrape model) is a throwaway that the family already tracks as
   drift. Adopting the snapshot model once is simpler and more functional than building a
   multi-target path twice.

## Dependency spine

```
Feature 5 (auto-detect)            ← PR #1, no dependencies, unblocks everything
   │
   ├─► NBU 10.x support PR         ← API version=10.0 + jobs cursor pagination
   │                                  (supersedes the operator's mistaken "v3.0 fixes")
   │
   └─► Feature 1 (snapshot model + nbuservers[] + `site` label)   ← keystone refactor
            │
            ├─► Feature 3 (per-client) ──► Feature 4 (per-client "no backup in 25h" alert)
            └─► Feature 2 (tape) ────────► Feature 4 (tape alerts)
```

## Per-feature direction

### "v3.0 validation fixes" (operator's "done" work) — DO NOT accept; premise was false

The operator proposed version-guarding the opt-in collectors, suppressing
`nbu_jobs_dedup_ratio`, and a startup warning, on the belief that NBU 10.x lacks those
endpoints/fields. Validated against the real NBU 10.3 spec (`docs/veritas-10.3/`), that
premise is **false**: at API `version=10.0` the opt-in endpoints all exist and the jobs
schema includes both `dedupRatio` and `jobQueueReason`. His symptoms come from the exporter
requesting a bogus/legacy version (`3.0`) and from **broken jobs cursor pagination** — not
from missing features. **Do not merge those fixes.** The correct, grounded fix is the
**NBU 10.x support** work: negotiate `version=10.0` and switch jobs to cursor pagination —
see [`2026-06-16-nbu-10x-support-design.md`](2026-06-16-nbu-10x-support-design.md).

### Feature 1 — Multi-master / multi-site — keystone, deferred

**Decision:** option (b) — one exporter process, a `nbuservers:[]` array, a required
`site` identity label (1:1 with master, per the NetBackup topology), emitted on **every**
series via the shared label builders and obeying the label-key consistency invariant
(`ppdd` ADR-0006). Per-target version detection; the per-scrape storage cache is replaced
by the snapshot store.

**Why deferred:** done canonically this is not "add a label" — it is bringing nbu onto the
family **snapshot collection model** (background loop → immutable snapshot → both export
paths read the snapshot), which nbu does not yet have. That is a real refactor and the
keystone the rest of the request layers on; it must not be rushed into the first PR.

**Answers to the requester's sub-questions:**

- *(a) vs (b)?* → (b). Instance-per-master is not the family pattern.
- *Where does the `site` label go?* → an identity label on every series via `metrics.go`
  builders — not just in `StorageMetricKey`/`JobMetricKey`, and not a process-wide const
  label (the process now serves multiple targets).
- *`nbuservers:[]` with a required per-entry identity, each its own collector?* → yes;
  fan-out is an `errgroup` with a concurrency limit, not unbounded goroutines.
- *Version detection per entry?* → yes, per target.
- *Cache per entry or shared?* → neither; the snapshot store subsumes the per-scrape cache.

### Feature 2 — Tape / robot metrics — REST, opt-in, deferred

**Decision:** use the REST API, not CLI shell-out. The endpoints exist under **`/storage/`**
in the checked-in specs (`/storage/tape-media`, `/storage/drives`,
`/storage/robots-device-hosts` — present in both the 10.5/API-12.0 and 11.0/API-13.0
bundles), **not** under the `/media/*` paths the request assumed. Ship as an opt-in
sub-collector per ADR-0002.

**Why CLI shell-out is rejected:** shelling out to `bpmedialist` / `vmquery` would force
the exporter to run *on* a master/media server with the binaries installed, breaking the
remote HTTP-only model the whole project is built on.

**No longer blocked on the requester:** the NBU 10.3 spec is now checked in
(`docs/veritas-10.3/`, API `version=10.0`). `/storage/drives` is confirmed present;
`/storage/tape-media` and the robot endpoints still need a closer read of
`docs/veritas-10.3/storage.yaml` — but that is our-side spec work now, not an appliance `curl`.

### Feature 3 — Per-client metrics — opt-in + allowlist, deferred

**Decision:** `ClientName` is already in `JobAttributes`; exposing it as a label is gated
behind an opt-in config flag with an optional client **allowlist** to bound cardinality
(hundreds of clients otherwise), and a long lookback window so "last successful backup"
always has a value regardless of scrape interval. Naturally carries the `site` label once
Feature 1 lands, so it is sequenced after the keystone.

### Feature 4 — Alerting rules — separate optional file, deferred

**Decision:** ship environment-specific rules in a separate optional file
(`deploy/prometheus/rules-tape.yml` / `rules-perclient.yml`), keeping the generic
`nbu.rules.yml` clean. The per-client "no backup in 25h" alert depends on Feature 3; tape
alerts depend on Feature 2; generic alerting improvements can land independently of both.

## What the requester should hear

- **Yes** to all five directions; the only "no" is CLI-based tape (REST instead).
- His "v3.0 fixes" are **not** needed or merged — the premise was false (see above). The real
  fix is the NBU 10.x support work (API `version=10.0` + jobs cursor pagination), which we are
  doing ourselves.
- His "v3.0" was the exporter's Helm/release version, not the NetBackup API; NBU 10.3 actually
  speaks API `version=10.0`. No ask-back required — we verified everything against the 10.3
  spec he is running.
- The order is set by dependencies — Feature 1 (multi-site) is the keystone, done to the
  family standard so it lands once, correctly.
