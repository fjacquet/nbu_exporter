# Grafana Dashboard Redesign — Design

**Date:** 2026-06-14
**Status:** Approved (pending implementation plan)
**Branch base:** `feat/nbu-11.2-validation` (needs the 11.2 collector metric names)
**Scope:** Replace the single hand-drifted overview dashboard with four focused,
generated, cross-linked dashboards; retire the legacy dashboard.

## Background

`grafana/build_overview.py` generates `grafana/nbu-overview.json` and is the intended
source of truth. The NetBackup 11.2 work, however, **hand-edited the JSON** to add the
alerts/malware/catalog/SLO panels without updating the generator, so re-running the
generator would silently delete those panels. The legacy `NBU Statistics-*.json`
duplicates storage and per-policy job views.

The redesign makes the generator authoritative again, splits the content into focused
domain dashboards (the chosen "focus and logic" structure), and retires the legacy file.

## Goals

- Generator is the single source of truth for every dashboard (no hand-edited JSON).
- Four focused, cross-linked dashboards: Overview, Jobs, Storage, Data Protection.
- Each panel earns its place: consistent units/thresholds, RED-style framing
  (success rate + failures prominent), no cross-dashboard redundancy.
- Bilingual FR/EN panel titles retained.
- Legacy `NBU Statistics-*.json` removed; its content absorbed.

## Non-goals

- No Go/exporter changes; no new metrics.
- No edits to the quickstart-stack compose/provisioning (separate branch — see
  "Provisioning coordination").

## Architecture — generator module

Refactor the single script into a small, focused Python module under `grafana/gen/`,
plus an orchestrator. Plain stdlib Python 3 (no deps), matching the existing script.

| File | Responsibility |
|---|---|
| `grafana/gen/__init__.py` | empty package marker |
| `grafana/gen/panels.py` | panel/layout helpers lifted verbatim from `build_overview.py`: `nid`, `ds`, `gridpos`, `target`, `row`, `stat`, `gauge`, `timeseries`, `piechart`, `barchart`, `table_info` |
| `grafana/gen/variables.py` | template-variable builders (`datasource_var`, `storage_unit_var`, `policy_type_var`), a `dashboard_links()` helper (tag-based cross-links), and a `dashboard(uid, title, panels, templating, links)` assembler that returns the full dashboard dict (schemaVersion 39, tags, time, refresh, annotations) |
| `grafana/gen/overview.py` | `build() -> dict` for the Overview dashboard |
| `grafana/gen/jobs.py` | `build() -> dict` for the Jobs dashboard |
| `grafana/gen/storage.py` | `build() -> dict` for the Storage dashboard |
| `grafana/gen/dataprotection.py` | `build() -> dict` for the Data Protection dashboard |
| `grafana/build_dashboards.py` | orchestrator: calls each `build()`, writes the four JSONs, prints a summary; replaces `build_overview.py` |

`grafana/build_overview.py` is removed (`git rm`); `build_dashboards.py` supersedes it.

Note on panel IDs: the current `nid()` uses a module-global counter. With multiple
dashboards generated in one process, IDs must be unique **within** each dashboard but
may repeat across dashboards. `panels.py` exposes `reset_ids()` which each dashboard
`build()` calls first, so every dashboard numbers its panels from 1 independently.

## Dashboards

All four: `${datasource}` variable, tag `netbackup`, `schemaVersion: 39`,
`time: now-24h`, `refresh: 1m`, bilingual FR/EN titles, and a `links` block (tag-based
`netbackup`) so every dashboard cross-links to the others. Panels reuse the existing
helper styles.

### Overview (`uid: nbu-overview`, var: datasource)
One-screen health + KPIs that route to depth.
- Health row: `nbu_up` (UP/DOWN bg), `nbu_api_version`, scrape staleness
  (`time() - nbu_last_scrape_timestamp_seconds`), `nbu_response_time_ms` timeseries.
- KPI row (stat tiles):
  - Backup success rate: `sum(nbu_status_count{action="BACKUP",status="0"}) / clamp_min(sum(nbu_status_count{action="BACKUP"}),1) * 100`
  - Failed backups (24h): `sum(nbu_status_count{action="BACKUP"}) - sum(nbu_status_count{action="BACKUP",status="0"})`
  - Active alerts: `sum(nbu_alerts_count)` (and by severity via a small barchart)
  - Storage % used: `sum(nbu_disk_bytes{size="used"}) / clamp_min(sum(nbu_disk_bytes),1) * 100`
  - Infected files: `sum(nbu_malware_files_infected)`
- Drill-down handled by the dashboard `links` block (to Jobs / Storage / Data Protection).

### Jobs (`uid: nbu-jobs`, var: policy_type)
- Backup success rate (bg stat, thresholds red<90<yellow<99 green).
- Job states piechart: `sum by (state) (nbu_jobs_state_count)`.
- Jobs by policy barchart: `sum by (policy_type) (nbu_jobs_count{action="BACKUP",policy_type=~"$policy_type"})`.
- Backup volume (stacked timeseries, bytes): `sum by (policy_type) (nbu_jobs_bytes{action="BACKUP",policy_type=~"$policy_type"})`.
- Queued jobs by reason barchart: `sum by (reason) (nbu_jobs_queued_count)`.
- Duration p50/p95 timeseries (s): `histogram_quantile(0.95|0.50, sum by (le, policy_type) (nbu_job_duration_seconds_bucket{policy_type=~"$policy_type"}))`.
- Files processed: `sum(nbu_jobs_files_count{policy_type=~"$policy_type"})`; Mean dedup: `avg(nbu_jobs_dedup_ratio{policy_type=~"$policy_type"})`.

### Storage (`uid: nbu-storage`, var: storage_unit)
- % used gauge per unit: `sum by (name) (nbu_disk_bytes{name=~"$storage_unit",size="used"}) / sum by (name) (nbu_disk_bytes{name=~"$storage_unit"}) * 100`.
- Used vs total capacity timeseries (bytes): `sum by (name) (nbu_disk_bytes{name=~"$storage_unit",size="used"})` and `nbu_disk_capacity_bytes{name=~"$storage_unit"}`.
- Storage units table: `nbu_storage_info{name=~"$storage_unit"}`.
- Max concurrent jobs: `nbu_storage_max_concurrent_jobs{name=~"$storage_unit"}`;
  Max fragment size (bytes): `nbu_storage_max_fragment_size_bytes{name=~"$storage_unit"}`.

### Data Protection (`uid: nbu-dataprotection`, var: datasource only)
Home of the 11.2 collectors.
- Alerts by severity (barchart): `sum by (severity) (nbu_alerts_count)`; by category table.
- Malware scanned vs infected (timeseries): `nbu_malware_files_scanned`, `nbu_malware_files_infected`.
- Malware scan status (barchart): `sum by (status) (nbu_malware_scan_count)`.
- Catalog posture (barchart): `sum by (malware_status) (nbu_catalog_images_count)` and by `anomaly_status`.
- SLO count (stat): `sum(nbu_slo_count)`.

## Legacy removal

`git rm "grafana/NBU Statistics-1629904585394.json"`. Storage and per-policy job views
are covered by the Storage and Jobs dashboards.

## File layout

Generated JSONs stay flat in `grafana/`: `nbu-overview.json`, `nbu-jobs.json`,
`nbu-storage.json`, `nbu-dataprotection.json`.

## Provisioning coordination (cross-branch)

The quickstart-stack branch (`feat/quickstart-stack`) provisions Grafana by mounting
two specific dashboard files. After this redesign there are four dashboards and no
legacy file. Follow-up (on the quickstart branch, at reconciliation time): switch the
Grafana dashboard mount from individual files to mounting the dashboard set, so adding
or removing a dashboard needs no compose change. Documented here; not built on this
branch.

## Verification

No Go code. Acceptance checks:
- `python3 grafana/build_dashboards.py` runs clean and writes the four JSONs.
- `python3 -m json.tool` validates each generated JSON.
- A metric-reference check: every `expr` in the generated JSONs references only known
  `nbu_*` metric names (the exporter's metric set on this branch). Implemented as a
  small assertion in a `grafana/gen/validate.py` helper run by `build_dashboards.py`
  (or a standalone check invoked in verification), failing on any unknown `nbu_*` name.
- `docs/dashboards.md` updated to describe the four-dashboard layout (if present on the
  branch; otherwise note for the docs follow-up).
- Manual eyeball via the demo stack is the final confidence check (needs a live NBU).

## Risks

- **Panel ID collisions across dashboards:** mitigated by `reset_ids()` per dashboard.
- **Metric-name typos:** mitigated by the metric-reference validation step.
- **Cross-branch drift with quickstart provisioning:** mitigated by the documented
  directory-mount follow-up; until then the demo provisions whatever files are mounted.
- **`graphTooltip`/schema parity:** keep `schemaVersion: 39` and the existing helper
  output shape so the JSON imports cleanly into the provisioned Grafana.
