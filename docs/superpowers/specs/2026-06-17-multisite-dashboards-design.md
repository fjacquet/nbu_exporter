# Multi-Site Grafana Dashboards — Design

**Date:** 2026-06-17
**Status:** Draft for review (depends on Feature 1 shipping the `site` label)
**Scope:** Add a `site` template variable and per-site filtering to the generated Grafana
dashboards, so the multi-site exporter (Feature 1) is usable in one Grafana view.
**Depends on:** [`2026-06-17-feature1-multisite-design.md`](2026-06-17-feature1-multisite-design.md)
— every metric must already carry a `site` label before these dashboard changes are useful.

## Background

Dashboards are **generated** — `grafana/build_dashboards.py` + the `grafana/gen/` module
(`variables.py`, `panels.py`, `overview.py`, `jobs.py`, `storage.py`, `dataprotection.py`)
are the single source of truth; the JSON under `grafana/*.json` is build output and must not
be hand-edited (see `2026-06-14-dashboard-redesign-design.md`). `gen/variables.py` already has
template-variable builders (`datasource_var`, `storage_unit_var`, `policy_type_var`) and the
`dashboard(...)` assembler. Stdlib Python 3, no deps.

Today every panel query is single-site (no `site` label exists yet). Once Feature 1 adds
`site`, the dashboards need a `site` selector and `site`-aware queries; otherwise series from
multiple masters collapse together or double-count.

## Goals

- A **`site` template variable** (multi-value + "All") on every dashboard, populated from the
  metrics (`label_values(nbu_up, site)`), placed first in the variable list.
- Every panel query filtered by `site=~"$site"` and, where a panel shows per-series detail,
  `site` added to the legend / `by (site, …)` grouping.
- A per-site **status row** on the overview (e.g. `nbu_up{site=~"$site"}` as a stat/table) so a
  down master is obvious.
- Regenerated JSON committed alongside the generator change (build output stays in sync).

## Non-goals

- No new metrics or panels for tape/per-client (Features 2/3 — separate dashboard work later).
- No change to the redesign's structure (overview / jobs / storage / dataprotection split).

## Design

### Generator (`grafana/gen/`)

- `variables.py`: add `site_var(datasource)` returning a Grafana `query` variable
  (`label_values(nbu_up, site)`, `multi: true`, `includeAll: true`, `current: All`,
  refresh on time-range change). Export it from the variable set used by each dashboard.
- Each dashboard builder (`overview.py`, `jobs.py`, `storage.py`, `dataprotection.py`): include
  `site_var(...)` as the **first** templating entry (before datasource-scoped ones), and thread
  `site=~"$site"` into every PromQL expression the builder emits. Centralize the filter so each
  builder composes label matchers through one helper (avoid hand-editing each expr string).
- `panels.py`: where a panel already groups (`by (storage_unit)`, `by (policyType)`, …), add
  `site` to the grouping and to the legend format (`{{site}} / {{…}}`) so multi-site series are
  distinguishable.
- `overview.py`: add a compact "Sites" status row near the top — `nbu_up{site=~"$site"}` as a
  stat panel repeated by `site` (or a small table), plus `nbu_last_scrape_timestamp_seconds`
  age per site.

### Build & validation

- Regenerate: `uv run python grafana/build_dashboards.py` (stdlib; `uv run` per project Python
  convention), then commit the regenerated `grafana/*.json`.
- `grafana/gen/validate.py` (existing) must pass: confirm referenced metrics/labels exist and
  the `site` variable is wired on every dashboard.

## Testing / verification

- Run the existing generator validation (`validate.py`) — green.
- Diff the regenerated JSON: every dashboard gains exactly one `site` templating var (first),
  and every targeted expression contains `site=~"$site"`.
- Manual (compose stack): with the multi-site exporter feeding two `site` values, the `site`
  dropdown lists both, "All" shows both, selecting one filters every panel, and a stopped
  master shows `nbu_up=0` for just that site.

## Acceptance criteria

- One `site` variable (multi + All) on overview / jobs / storage / dataprotection, first in the
  list, sourced from `label_values(nbu_up, site)`.
- No panel double-counts across sites; per-series panels show `site` in the legend.
- Generator remains the single source of truth (no hand-edited JSON); regenerated output
  committed.

## Open questions for review

1. Variable source metric: `nbu_up` (always present per site) vs a broader metric — `nbu_up`
   recommended (one series per site regardless of which collectors are enabled).
2. Overview "Sites" row: repeated stat panels vs a single status table — which reads better
   for 2–handful of sites? (Suggest a small table.)
