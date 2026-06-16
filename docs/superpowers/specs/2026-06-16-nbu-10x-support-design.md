# NetBackup 10.x Support: API `version=10.0` + Jobs Cursor Pagination — Design

**Date:** 2026-06-16
**Status:** Approved (pending implementation plan)
**Scope:** Make the exporter work correctly against NetBackup 10.x by (1) negotiating the
real API media-type `version=10.0` instead of the bogus `3.0`, and (2) fixing jobs
pagination, which is cursor-based on all modern NetBackup and is currently broken.
**Supersedes:** the "v3.0 graceful-degradation" item in
[`2026-06-16-nbu-feature-deferrals.md`](2026-06-16-nbu-feature-deferrals.md) — that item was
based on a false premise (see Background).

## Background

An external operator (NBU 10.3, two sites) reported degraded/absent metrics and asked for
fixes. His "API v3.0" label turned out to be the exporter's **Helm/release** version, not a
NetBackup API version — he is not authoritative on the NBU API. Grounding against the real
NBU 10.3 OpenAPI bundle (`docs/veritas-10.3/`, added for this work) overturned the original
"graceful degradation" plan:

1. **The version table is wrong for NBU 10.x.** The 10.3 API media type is
   `application/vnd.netbackup+json;version=10.0` (with a few legacy endpoints at `6.0`/`9.0`);
   there is **no `3.0`** anywhere in the bundle. The exporter maps NBU 10.0–10.4 to API
   `3.0` (`models.APIVersion30`), a version the appliance does not document. Detection tries
   `14.0→13.0→12.0→3.0`; a 10.3 box 406s the first three and *tolerates* `3.0` for
   backward-compat but answers with a legacy reduced representation — the apparent "missing
   fields". The real sequence is **`10.0` (NBU 10.x) → `12.0` (10.5) → `13.0` (11.0) →
   `14.0` (11.2)**.

2. **`dedupRatio` and `jobQueueReason` are present** in the 10.3 jobs schema at
   `version=10.0`, correctly typed — so no field is "absent" and no metric suppression is
   needed. The model validation (jobs attributes + storage) is otherwise fully compatible,
   including the `float64` `dedupRatio`/`acceleratorOptimization` from the issue-#17 fix.

3. **Jobs pagination is cursor-based and the exporter breaks on it.** `GET /admin/jobs` uses
   `page[after]`/`page[before]` string cursors and returns `meta.pagination.next`/`prev` as
   **strings** plus `rangeTruncated` (bool) — confirmed in the 10.3, 10.5 **and** 11.0
   bundles, where the endpoint description notes *"a breaking change… starting with version
   9.0."* The exporter declares `Next int`/`Offset int`/`Last int`/… (`models/Jobs.go`) and
   sends `page[offset]` (`exporter/netbackup.go`). Consequences:
   - `next` (string) unmarshalled into `Next int` **fails** → the whole jobs fetch errors →
     all `nbu_jobs_*` metrics silently vanish (graceful degradation hides it). This is the
     root cause of "Failed to fetch job metrics" (issue #13: storage but no jobs on NBU 11.0).
   - Even if tolerated, `offset == last` never advances past page 1.

   This affects **every modern NetBackup** (≥ API 9.0: 10.0/12.0/13.0/14.0), not just 10.x.

## Goals

- Negotiate API `version=10.0` for NBU 10.x; drop the bogus `3.0` entirely.
- Fix jobs pagination to cursor semantics so `nbu_jobs_*` metrics work on all modern NBU.
- Restore full job metrics for the operator's NBU 10.3 environment using our own spec — no
  dependency on the operator.

## Non-goals

- **No change to storage pagination** — storage remains offset-based (verified correct in
  10.3 at `version=10.0`). Do not "fix" both the same way.
- No new metrics, no metric-name/label changes, no new collectors.
- No multi-site/tape/per-client work (separate items).

## Design

### Section 1 — Version model (`3.0` → `10.0`)

The detector is data-driven (it iterates `models.SupportedAPIVersions`), so behavior changes
flow from one slice.

- `models/Config.go`: replace `APIVersion30 = "3.0"` with `APIVersion100 = "10.0"`
  (comment: NetBackup 10.0–10.4); `SupportedAPIVersions = [14.0, 13.0, 12.0, 10.0]`; fix the
  `SetDefaults` doc-comment ladder.
- `exporter/version_detector.go` + `exporter/prometheus.go`: doc-comment ladders
  `14.0 → 13.0 → 12.0 → 10.0`. No logic change.
- `internal/testutil`: rename the `APIVersion30` constant → `APIVersion100` (`"10.0"`); fixes
  the `exporter/test_common.go` alias.
- Tests (mechanical `3.0`→`10.0`): `Config_test.go` (supported-list, constants, valid/
  supported cases), `version_detection_integration_test.go` (ladder + fallback),
  `end_to_end_test.go`, `metrics_consistency_test.go`, `performance_test.go`, the
  `api_compatibility_test.go` fixture map.
- Docs: detection ladder in README/CLAUDE.md/config-examples, and
  `docs/config-examples/config-netbackup-10.0.yaml` (today it tells NBU 10.x users
  `apiVersion: "3.0"` — the exact wrong value → `"10.0"`).

### Section 2 — Jobs cursor pagination (the core fix)

Jobs only. Storage's offset pagination is untouched; the jobs and storage paths will be
cleanly separated so a shared handler (if any) keeps storage on offset semantics.

- **Model** (`models/Jobs.go`): replace the jobs `Meta.Pagination` integer fields with the
  cursor shape — `Limit int`, `Next string`, `Prev string`, `RangeTruncated bool`.
- **Request** (`exporter/netbackup.go` `FetchJobDetails`): drop `page[offset]`; send
  `page[after]=<cursor>` (omitted on the first page); signature `offset int` → `cursor string`.
- **Loop** (`FetchJobDetails` / `HandlePagination` jobs driver): start with an empty cursor,
  fetch a page, then follow `meta.pagination.next` until it is empty (or `data` is empty),
  instead of comparing `offset == last`.
- `initiatorId` is absent from the 10.x jobs schema; the Go `InitiatorID string` is harmless
  (stays empty, never read) — left as-is.

## Testing (TDD)

- **Cursor regression (unit):** a jobs payload with a string `next` cursor unmarshals into
  the new model (the current `Next int` makes this fail first — RED).
- **Multi-page cursor loop (integration):** a mock `/admin/jobs` that returns page 1 with a
  non-empty `next` and page 2 with an empty `next`; assert the loop follows the cursor,
  sends `page[after]` on the second request (not `page[offset]`), and aggregates jobs from
  **both** pages. Current offset code stops after page 1 — RED.
- **Storage unchanged:** existing storage offset-pagination tests stay green.
- **Version:** `10.0` in `SupportedAPIVersions`; detection ladder `14.0→13.0→12.0→10.0`;
  validation accepts `10.0` and no longer lists `3.0`.
- **Cross-version job fixtures:** the per-version jobs fixtures
  (`testdata/api-versions/jobs-response-*.json`) move to cursor pagination (cursor applies to
  all versions ≥ 9.0); storage fixtures stay offset. Rename `*-v3.json` → `*-v10.json`.
- `make ci` green (fmt/vet/lint/test-race/govulncheck + 70% coverage).

## Acceptance criteria

- Against a mock cursor-paginated `/admin/jobs`, the exporter follows `next` across pages and
  emits `nbu_jobs_*` for all jobs (not just page 1).
- With `apiVersion` omitted against an NBU 10.x appliance, detection resolves `10.0`, jobs are
  fetched via cursors and storage via offset — full metrics, no `apiVersion: "3.0"` workaround.
- No `3.0` remains in the supported versions, detection ladder, or shipped config examples.
- No change to emitted metric names or labels.

## References

- Model validation (this session): storage fully compatible; job attributes compatible; jobs
  pagination cursor-based and broken — `models/Jobs.go` pagination struct,
  `exporter/netbackup.go` `FetchJobDetails`.
- Spec evidence: `docs/veritas-10.3/admin.yaml` `/admin/jobs` (`page[after]`/`page[before]`,
  string `next`, `rangeTruncated`, "breaking change since version 9.0"); media-type
  `version=10.0`.
- Family standard: `exporter-standards` — "absent, never zero" (architecture.md), live-cluster
  validation; ADR-0002 (opt-in collectors, graceful degradation).
- Related: `2026-06-16-nbu-10x-and-roadmap-design.md` (Feature 5, shipped),
  `2026-06-16-nbu-feature-deferrals.md` (direction; this supersedes its v3.0 item).
