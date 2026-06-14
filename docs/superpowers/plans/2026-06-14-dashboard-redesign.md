# Grafana Dashboard Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the single hand-drifted overview dashboard with four focused, generated, cross-linked dashboards (Overview, Jobs, Storage, Data Protection) and retire the legacy dashboard.

**Architecture:** Refactor `build_overview.py` into a small stdlib-Python generator package `grafana/gen/` (shared panel helpers + per-dashboard builders) driven by an orchestrator `grafana/build_dashboards.py`. A metric-reference validator fails the build on any unknown `nbu_*` metric name. The generator is the single source of truth; no hand-edited JSON.

**Tech Stack:** Python 3 stdlib (`json`, `re`), Grafana schemaVersion 39, Prometheus.

**Spec:** `docs/superpowers/specs/2026-06-14-dashboard-redesign-design.md`
**Branch:** `feat/dashboard-redesign` (off `feat/nbu-11.2-validation`).

**Conventions for every task:**
- Pure Python stdlib, no pip deps. Run with the system `python3` (fall back to `uv run python3` only if needed).
- After each builder task, regenerate and validate: `python3 grafana/build_dashboards.py` then `python3 -m json.tool <file> >/dev/null`.
- Commit messages use Conventional Commits + trailer `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`.
- Bilingual FR/EN panel titles ("FR / EN").

---

## File Structure

| File | Responsibility | Action |
|---|---|---|
| `grafana/gen/__init__.py` | package marker | Create |
| `grafana/gen/panels.py` | panel/layout helpers + `reset_ids()` | Create |
| `grafana/gen/variables.py` | template-var builders, `dashboard_links()`, `dashboard()` assembler | Create |
| `grafana/gen/validate.py` | `KNOWN_METRICS` + `check_dashboard()` metric-reference validator | Create |
| `grafana/gen/overview.py` | `build()` for Overview | Create |
| `grafana/gen/jobs.py` | `build()` for Jobs | Create |
| `grafana/gen/storage.py` | `build()` for Storage | Create |
| `grafana/gen/dataprotection.py` | `build()` for Data Protection | Create |
| `grafana/build_dashboards.py` | orchestrator: build + validate + write all four | Create |
| `grafana/build_overview.py` | superseded | Delete (git rm) |
| `grafana/nbu-overview.json` | regenerated (slimmed to KPIs + links) | Overwrite (generated) |
| `grafana/nbu-jobs.json`, `nbu-storage.json`, `nbu-dataprotection.json` | new generated dashboards | Create (generated) |
| `grafana/NBU Statistics-1629904585394.json` | legacy | Delete (git rm) |
| `docs/metrics.md` | note the four-dashboard layout | Modify |

---

## Task 1: Generator package — panel helpers

**Files:**
- Create: `grafana/gen/__init__.py`, `grafana/gen/panels.py`

- [ ] **Step 1: Create the package marker**

`grafana/gen/__init__.py`: empty file (one trailing newline).

- [ ] **Step 2: Create `grafana/gen/panels.py`**

Lift the helper functions from the existing `grafana/build_overview.py` verbatim, add `reset_ids()`, and make `DS` a module constant. Full content:

```python
"""Shared Grafana panel/layout helpers for the nbu_exporter dashboards."""

DS = "${datasource}"
_id = 0


def reset_ids():
    """Reset the per-dashboard panel id counter. Call at the start of each build()."""
    global _id
    _id = 0


def nid():
    global _id
    _id += 1
    return _id


def ds():
    return {"type": "prometheus", "uid": DS}


def gridpos(x, y, w, h):
    return {"x": x, "y": y, "w": w, "h": h}


def target(expr, legend="", instant=False):
    return {
        "datasource": ds(),
        "expr": expr,
        "legendFormat": legend,
        "range": not instant,
        "instant": instant,
        "refId": "A",
    }


def row(title, y):
    return {
        "type": "row",
        "id": nid(),
        "title": title,
        "collapsed": False,
        "gridPos": gridpos(0, y, 24, 1),
        "panels": [],
    }


def stat(title, expr, x, y, w, h, unit="none", mappings=None, thresholds=None,
         text_mode="auto", legend="", color_mode="value"):
    return {
        "type": "stat",
        "id": nid(),
        "title": title,
        "datasource": ds(),
        "gridPos": gridpos(x, y, w, h),
        "fieldConfig": {
            "defaults": {
                "unit": unit,
                "mappings": mappings or [],
                "color": {"mode": "thresholds"},
                "thresholds": {
                    "mode": "absolute",
                    "steps": thresholds or [{"color": "green", "value": None}],
                },
            },
            "overrides": [],
        },
        "options": {
            "colorMode": color_mode,
            "graphMode": "area",
            "justifyMode": "auto",
            "orientation": "auto",
            "reduceOptions": {"calcs": ["lastNotNull"], "fields": "", "values": False},
            "textMode": text_mode,
        },
        "targets": [target(expr, legend, instant=True)],
    }


def gauge(title, expr, x, y, w, h, legend="{{name}}"):
    return {
        "type": "gauge",
        "id": nid(),
        "title": title,
        "datasource": ds(),
        "gridPos": gridpos(x, y, w, h),
        "fieldConfig": {
            "defaults": {
                "unit": "percent",
                "min": 0,
                "max": 100,
                "color": {"mode": "thresholds"},
                "thresholds": {
                    "mode": "absolute",
                    "steps": [
                        {"color": "green", "value": None},
                        {"color": "yellow", "value": 80},
                        {"color": "red", "value": 90},
                    ],
                },
            },
            "overrides": [],
        },
        "options": {
            "reduceOptions": {"calcs": ["lastNotNull"], "fields": "", "values": False},
            "orientation": "auto",
            "showThresholdLabels": False,
            "showThresholdMarkers": True,
        },
        "targets": [target(expr, legend, instant=True)],
    }


def timeseries(title, targets, x, y, w, h, unit="none", stack=False, fill=10):
    return {
        "type": "timeseries",
        "id": nid(),
        "title": title,
        "datasource": ds(),
        "gridPos": gridpos(x, y, w, h),
        "fieldConfig": {
            "defaults": {
                "unit": unit,
                "custom": {
                    "drawStyle": "line",
                    "fillOpacity": fill,
                    "showPoints": "never",
                    "stacking": {"mode": "normal" if stack else "none"},
                    "lineWidth": 1,
                },
                "color": {"mode": "palette-classic"},
            },
            "overrides": [],
        },
        "options": {
            "legend": {"displayMode": "table", "placement": "bottom", "calcs": ["mean", "lastNotNull", "max"]},
            "tooltip": {"mode": "multi", "sort": "desc"},
        },
        "targets": targets,
    }


def piechart(title, expr, x, y, w, h, legend="{{state}}"):
    return {
        "type": "piechart",
        "id": nid(),
        "title": title,
        "datasource": ds(),
        "gridPos": gridpos(x, y, w, h),
        "fieldConfig": {"defaults": {"color": {"mode": "palette-classic"}}, "overrides": []},
        "options": {
            "legend": {"displayMode": "table", "placement": "right", "values": ["value", "percent"]},
            "pieType": "donut",
            "reduceOptions": {"calcs": ["lastNotNull"], "fields": "", "values": False},
        },
        "targets": [target(expr, legend, instant=True)],
    }


def barchart(title, expr, x, y, w, h, legend="{{policy_type}}", unit="none"):
    return {
        "type": "barchart",
        "id": nid(),
        "title": title,
        "datasource": ds(),
        "gridPos": gridpos(x, y, w, h),
        "fieldConfig": {
            "defaults": {"unit": unit, "color": {"mode": "palette-classic"}},
            "overrides": [],
        },
        "options": {
            "orientation": "horizontal",
            "showValue": "auto",
            "legend": {"showLegend": False},
            "xTickLabelRotation": 0,
        },
        "targets": [target(expr, legend, instant=True)],
    }


def table_info(title, expr, x, y, w, h):
    return {
        "type": "table",
        "id": nid(),
        "title": title,
        "datasource": ds(),
        "gridPos": gridpos(x, y, w, h),
        "fieldConfig": {"defaults": {"custom": {"filterable": True}}, "overrides": []},
        "options": {"showHeader": True},
        "targets": [{
            "datasource": ds(),
            "expr": expr,
            "format": "table",
            "instant": True,
            "refId": "A",
        }],
        "transformations": [
            {"id": "labelsToFields", "options": {}},
            {"id": "organize", "options": {"excludeByName": {"Time": True, "Value": True, "__name__": True, "job": True, "instance": True}}},
        ],
    }
```

- [ ] **Step 3: Smoke-test the helpers**

Run:
```bash
python3 -c "from grafana.gen import panels as p; p.reset_ids(); a=p.stat('t','nbu_up',0,0,4,4); b=p.row('r',0); assert a['id']==1 and b['id']==2; p.reset_ids(); c=p.row('r2',0); assert c['id']==1; print('panels OK')"
```
Expected: `panels OK`.
(If `grafana` is not importable as a package from the repo root, instead run the command with `PYTHONPATH=.`: `PYTHONPATH=. python3 -c "..."`.)

- [ ] **Step 4: Commit**

```bash
git add grafana/gen/__init__.py grafana/gen/panels.py
git commit -m "feat(grafana): add generator panel-helper module"
```

---

## Task 2: Template variables, dashboard assembler, and validator

**Files:**
- Create: `grafana/gen/variables.py`, `grafana/gen/validate.py`

- [ ] **Step 1: Create `grafana/gen/variables.py`**

```python
"""Template-variable builders, cross-dashboard links, and the dashboard assembler."""

from grafana.gen.panels import ds


def datasource_var():
    return {
        "name": "datasource",
        "type": "datasource",
        "query": "prometheus",
        "label": "Datasource",
        "current": {},
        "hide": 0,
        "refresh": 1,
    }


def _query_var(name, label, metric, value_label):
    return {
        "name": name,
        "type": "query",
        "datasource": ds(),
        "label": label,
        "query": {"query": f"label_values({metric}, {value_label})", "refId": "StandardVariableQuery"},
        "includeAll": True,
        "multi": True,
        "allValue": ".*",
        "current": {"text": "All", "value": "$__all"},
        "refresh": 2,
        "sort": 1,
    }


def storage_unit_var():
    return _query_var("storage_unit", "Storage unit", "nbu_disk_bytes", "name")


def policy_type_var():
    return _query_var("policy_type", "Policy type", "nbu_jobs_count", "policy_type")


def dashboard_links():
    """Tag-based links so every nbu dashboard cross-links to the others."""
    return [{
        "type": "dashboards",
        "title": "NetBackup",
        "tags": ["netbackup"],
        "asDropdown": True,
        "includeVars": False,
        "keepTime": True,
        "targetBlank": False,
        "icon": "external link",
    }]


def dashboard(uid, title, panels, extra_vars=None):
    """Assemble a full dashboard dict. `extra_vars` are appended after the datasource var."""
    templating = {"list": [datasource_var()] + (extra_vars or [])}
    return {
        "uid": uid,
        "title": title,
        "tags": ["netbackup", "nbu", "backup"],
        "schemaVersion": 39,
        "version": 1,
        "editable": True,
        "graphTooltip": 1,
        "time": {"from": "now-24h", "to": "now"},
        "timepicker": {},
        "timezone": "",
        "refresh": "1m",
        "links": dashboard_links(),
        "templating": templating,
        "annotations": {"list": [{
            "name": "Annotations & Alerts",
            "type": "dashboard",
            "datasource": {"type": "grafana", "uid": "-- Grafana --"},
            "enable": True,
            "iconColor": "red",
        }]},
        "panels": panels,
    }
```

- [ ] **Step 2: Create `grafana/gen/validate.py`**

```python
"""Validate that generated dashboards reference only known nbu_* metrics."""

import re

# The exporter's metric set (base names). Histogram suffixes are normalized below.
KNOWN_METRICS = {
    "nbu_up",
    "nbu_api_version",
    "nbu_last_scrape_timestamp_seconds",
    "nbu_response_time_ms",
    "nbu_disk_bytes",
    "nbu_disk_capacity_bytes",
    "nbu_storage_max_concurrent_jobs",
    "nbu_storage_max_fragment_size_bytes",
    "nbu_storage_info",
    "nbu_jobs_bytes",
    "nbu_jobs_count",
    "nbu_status_count",
    "nbu_jobs_state_count",
    "nbu_jobs_files_count",
    "nbu_jobs_dedup_ratio",
    "nbu_jobs_queued_count",
    "nbu_job_duration_seconds",
    "nbu_alerts_count",
    "nbu_malware_files_scanned",
    "nbu_malware_files_infected",
    "nbu_malware_scan_count",
    "nbu_catalog_images_count",
    "nbu_slo_count",
}

_METRIC_RE = re.compile(r"nbu_[a-z0-9_]+")
_HIST_SUFFIX = re.compile(r"_(bucket|sum|count)$")


def _normalize(metric):
    # Map histogram series back to their base metric name.
    base = _HIST_SUFFIX.sub("", metric)
    # nbu_jobs_count must NOT be stripped to nbu_jobs; only strip hist suffixes
    # when the stripped base is a known histogram metric.
    if base == "nbu_job_duration_seconds":
        return base
    return metric


def _iter_exprs(panel):
    for tgt in panel.get("targets", []):
        if "expr" in tgt:
            yield tgt["expr"]
    for sub in panel.get("panels", []):  # row sub-panels
        yield from _iter_exprs(sub)


def check_dashboard(dash):
    """Return a sorted list of unknown nbu_* metric names referenced by the dashboard."""
    unknown = set()
    for panel in dash.get("panels", []):
        for expr in _iter_exprs(panel):
            for token in _METRIC_RE.findall(expr):
                name = _normalize(token)
                if name not in KNOWN_METRICS:
                    unknown.add(token)
    return sorted(unknown)
```

- [ ] **Step 3: Test the validator catches a bad metric and passes a good one**

Run:
```bash
PYTHONPATH=. python3 -c "
from grafana.gen import validate as v
good = {'panels':[{'targets':[{'expr':'sum(nbu_jobs_count)'},{'expr':'histogram_quantile(0.95, nbu_job_duration_seconds_bucket)'}]}]}
bad  = {'panels':[{'targets':[{'expr':'sum(nbu_bogus_metric)'}]}]}
assert v.check_dashboard(good)==[], v.check_dashboard(good)
assert v.check_dashboard(bad)==['nbu_bogus_metric'], v.check_dashboard(bad)
print('validator OK')
"
```
Expected: `validator OK`.

- [ ] **Step 4: Commit**

```bash
git add grafana/gen/variables.py grafana/gen/validate.py
git commit -m "feat(grafana): add template-var builders, dashboard assembler, metric validator"
```

---

## Task 3: Overview dashboard builder

**Files:**
- Create: `grafana/gen/overview.py`

- [ ] **Step 1: Create `grafana/gen/overview.py`**

```python
"""Overview dashboard: one-screen health + headline KPIs, links to domain dashboards."""

from grafana.gen import panels as p
from grafana.gen.variables import dashboard

UP_MAP = [{"type": "value", "options": {"0": {"text": "DOWN", "color": "red"},
                                        "1": {"text": "UP", "color": "green"}}}]


def build():
    p.reset_ids()
    out = []

    out.append(p.row("Santé / Health", 0))
    out.append(p.stat("Disponibilité / Availability", "nbu_up", 0, 1, 5, 4,
                      mappings=UP_MAP, color_mode="background",
                      thresholds=[{"color": "red", "value": None}, {"color": "green", "value": 1}]))
    out.append(p.stat("Version API / API version", "nbu_api_version", 5, 1, 4, 4,
                      text_mode="name", legend="{{version}}"))
    out.append(p.stat("Fraîcheur scrape / Scrape staleness",
                      "time() - nbu_last_scrape_timestamp_seconds", 9, 1, 6, 4,
                      unit="s", legend="{{source}}",
                      thresholds=[{"color": "green", "value": None}, {"color": "yellow", "value": 300}, {"color": "red", "value": 900}]))
    out.append(p.timeseries("Latence API / API latency",
                            [p.target("nbu_response_time_ms", "réponse / response")],
                            15, 1, 9, 4, unit="ms"))

    out.append(p.row("Indicateurs clés / Key indicators", 5))
    out.append(p.stat("Taux succès BACKUP / Backup success rate",
                      'sum(nbu_status_count{action="BACKUP",status="0"}) '
                      '/ clamp_min(sum(nbu_status_count{action="BACKUP"}), 1) * 100',
                      0, 6, 5, 5, unit="percent", color_mode="background",
                      thresholds=[{"color": "red", "value": None}, {"color": "yellow", "value": 90}, {"color": "green", "value": 99}]))
    out.append(p.stat("Échecs BACKUP / Failed backups",
                      'sum(nbu_status_count{action="BACKUP"}) '
                      '- sum(nbu_status_count{action="BACKUP",status="0"})',
                      5, 6, 5, 5,
                      thresholds=[{"color": "green", "value": None}, {"color": "red", "value": 1}]))
    out.append(p.stat("Stockage % utilisé / Storage % used",
                      'sum(nbu_disk_bytes{size="used"}) / clamp_min(sum(nbu_disk_bytes), 1) * 100',
                      10, 6, 4, 5, unit="percent",
                      thresholds=[{"color": "green", "value": None}, {"color": "yellow", "value": 80}, {"color": "red", "value": 90}]))
    out.append(p.stat("Fichiers infectés / Infected files",
                      "sum(nbu_malware_files_infected)", 14, 6, 5, 5,
                      thresholds=[{"color": "green", "value": None}, {"color": "red", "value": 1}]))
    out.append(p.barchart("Alertes par sévérité / Alerts by severity",
                          "sum by (severity) (nbu_alerts_count)", 19, 6, 5, 5,
                          legend="{{severity}}"))

    return dashboard("nbu-overview", "NetBackup — Vue d'ensemble / Overview", out)
```

- [ ] **Step 2: Generate and validate (temporary one-off until the orchestrator exists)**

Run:
```bash
PYTHONPATH=. python3 -c "
import json
from grafana.gen import overview
from grafana.gen import validate as v
d = overview.build()
u = v.check_dashboard(d)
assert u == [], f'unknown metrics: {u}'
json.dumps(d)  # must serialize
print('overview OK, panels:', len(d['panels']))
"
```
Expected: `overview OK, panels: 11`.

- [ ] **Step 3: Commit**

```bash
git add grafana/gen/overview.py
git commit -m "feat(grafana): add Overview dashboard builder"
```

---

## Task 4: Jobs dashboard builder

**Files:**
- Create: `grafana/gen/jobs.py`

- [ ] **Step 1: Create `grafana/gen/jobs.py`**

```python
"""Jobs dashboard: backup outcomes, states, volume, queue, durations, dedup."""

from grafana.gen import panels as p
from grafana.gen.variables import dashboard, policy_type_var


def build():
    p.reset_ids()
    out = []

    out.append(p.row("Résultats / Outcomes", 0))
    out.append(p.stat("Taux succès BACKUP / Backup success rate",
                      'sum(nbu_status_count{action="BACKUP",status="0"}) '
                      '/ clamp_min(sum(nbu_status_count{action="BACKUP"}), 1) * 100',
                      0, 1, 6, 8, unit="percent", color_mode="background",
                      thresholds=[{"color": "red", "value": None}, {"color": "yellow", "value": 90}, {"color": "green", "value": 99}]))
    out.append(p.piechart("États des jobs / Job states",
                          "sum by (state) (nbu_jobs_state_count)", 6, 1, 9, 8))
    out.append(p.barchart("Jobs par politique / Jobs by policy",
                          'sum by (policy_type) (nbu_jobs_count{action="BACKUP",policy_type=~"$policy_type"})',
                          15, 1, 9, 8))

    out.append(p.row("Volume & file / Volume & queue", 9))
    out.append(p.timeseries("Volume sauvegardé / Backup volume",
                            [p.target('sum by (policy_type) (nbu_jobs_bytes{action="BACKUP",policy_type=~"$policy_type"})', "{{policy_type}}")],
                            0, 10, 12, 8, unit="bytes", stack=True))
    out.append(p.barchart("Jobs en file / Queued jobs",
                          "sum by (reason) (nbu_jobs_queued_count)", 12, 10, 12, 8, legend="{{reason}}"))

    out.append(p.row("Durées & efficacité / Durations & efficiency", 18))
    out.append(p.timeseries("Durée jobs p50/p95 / Job duration p50/p95",
                            [p.target('histogram_quantile(0.95, sum by (le, policy_type) (nbu_job_duration_seconds_bucket{policy_type=~"$policy_type"}))', "p95 {{policy_type}}"),
                             p.target('histogram_quantile(0.50, sum by (le, policy_type) (nbu_job_duration_seconds_bucket{policy_type=~"$policy_type"}))', "p50 {{policy_type}}")],
                            0, 19, 16, 8, unit="s", fill=0))
    out.append(p.stat("Fichiers traités / Files processed",
                      'sum(nbu_jobs_files_count{policy_type=~"$policy_type"})', 16, 19, 4, 8))
    out.append(p.stat("Dédup moyenne / Mean dedup ratio",
                      'avg(nbu_jobs_dedup_ratio{policy_type=~"$policy_type"})', 20, 19, 4, 8, unit="none"))

    return dashboard("nbu-jobs", "NetBackup — Sauvegardes / Jobs", out, extra_vars=[policy_type_var()])
```

- [ ] **Step 2: Generate and validate**

Run:
```bash
PYTHONPATH=. python3 -c "
import json
from grafana.gen import jobs
from grafana.gen import validate as v
d = jobs.build(); u = v.check_dashboard(d)
assert u == [], f'unknown metrics: {u}'; json.dumps(d)
print('jobs OK, panels:', len(d['panels']))
"
```
Expected: `jobs OK, panels: 11`.

- [ ] **Step 3: Commit**

```bash
git add grafana/gen/jobs.py
git commit -m "feat(grafana): add Jobs dashboard builder"
```

---

## Task 5: Storage dashboard builder

**Files:**
- Create: `grafana/gen/storage.py`

- [ ] **Step 1: Create `grafana/gen/storage.py`**

```python
"""Storage dashboard: capacity utilization, units, limits."""

from grafana.gen import panels as p
from grafana.gen.variables import dashboard, storage_unit_var


def build():
    p.reset_ids()
    out = []

    out.append(p.row("Capacité / Capacity", 0))
    out.append(p.gauge("% Utilisé / % Used",
                       'sum by (name) (nbu_disk_bytes{name=~"$storage_unit",size="used"}) '
                       '/ sum by (name) (nbu_disk_bytes{name=~"$storage_unit"}) * 100',
                       0, 1, 8, 8))
    out.append(p.timeseries("Capacité utilisée / Used capacity",
                            [p.target('sum by (name) (nbu_disk_bytes{name=~"$storage_unit",size="used"})', "{{name}} used"),
                             p.target('nbu_disk_capacity_bytes{name=~"$storage_unit"}', "{{name}} total")],
                            8, 1, 16, 8, unit="bytes"))

    out.append(p.row("Unités / Units", 9))
    out.append(p.table_info("Unités de stockage / Storage units",
                            'nbu_storage_info{name=~"$storage_unit"}', 0, 10, 16, 8))
    out.append(p.stat("Jobs simultanés max / Max concurrent jobs",
                      'nbu_storage_max_concurrent_jobs{name=~"$storage_unit"}', 16, 10, 4, 8,
                      legend="{{name}}"))
    out.append(p.stat("Taille fragment max / Max fragment size",
                      'nbu_storage_max_fragment_size_bytes{name=~"$storage_unit"}', 20, 10, 4, 8,
                      unit="bytes", legend="{{name}}"))

    return dashboard("nbu-storage", "NetBackup — Stockage / Storage", out, extra_vars=[storage_unit_var()])
```

- [ ] **Step 2: Generate and validate**

Run:
```bash
PYTHONPATH=. python3 -c "
import json
from grafana.gen import storage
from grafana.gen import validate as v
d = storage.build(); u = v.check_dashboard(d)
assert u == [], f'unknown metrics: {u}'; json.dumps(d)
print('storage OK, panels:', len(d['panels']))
"
```
Expected: `storage OK, panels: 7`.

- [ ] **Step 3: Commit**

```bash
git add grafana/gen/storage.py
git commit -m "feat(grafana): add Storage dashboard builder"
```

---

## Task 6: Data Protection dashboard builder

**Files:**
- Create: `grafana/gen/dataprotection.py`

- [ ] **Step 1: Create `grafana/gen/dataprotection.py`**

```python
"""Data Protection dashboard: alerts, malware scans, catalog posture, SLOs (11.2 collectors)."""

from grafana.gen import panels as p
from grafana.gen.variables import dashboard


def build():
    p.reset_ids()
    out = []

    out.append(p.row("Alertes / Alerts", 0))
    out.append(p.barchart("Alertes par sévérité / Alerts by severity",
                          "sum by (severity) (nbu_alerts_count)", 0, 1, 12, 8, legend="{{severity}}"))
    out.append(p.table_info("Alertes par catégorie / Alerts by category",
                            "sum by (category, severity) (nbu_alerts_count)", 12, 1, 12, 8))

    out.append(p.row("Malware / Malware", 9))
    out.append(p.timeseries("Fichiers scannés vs infectés / Files scanned vs infected",
                            [p.target("nbu_malware_files_scanned", "scannés / scanned"),
                             p.target("nbu_malware_files_infected", "infectés / infected")],
                            0, 10, 12, 8))
    out.append(p.barchart("Statut des scans / Scan status",
                          "sum by (status) (nbu_malware_scan_count)", 12, 10, 12, 8, legend="{{status}}"))

    out.append(p.row("Catalogue & SLO / Catalog & SLO", 18))
    out.append(p.barchart("Posture malware / Malware posture",
                          "sum by (malware_status) (nbu_catalog_images_count)", 0, 19, 8, 8, legend="{{malware_status}}"))
    out.append(p.barchart("Posture anomalies / Anomaly posture",
                          "sum by (anomaly_status) (nbu_catalog_images_count)", 8, 19, 8, 8, legend="{{anomaly_status}}"))
    out.append(p.stat("SLO configurés / Configured SLOs",
                      "sum(nbu_slo_count)", 16, 19, 8, 8))

    return dashboard("nbu-dataprotection", "NetBackup — Protection des données / Data Protection", out)
```

- [ ] **Step 2: Generate and validate**

Run:
```bash
PYTHONPATH=. python3 -c "
import json
from grafana.gen import dataprotection as dp
from grafana.gen import validate as v
d = dp.build(); u = v.check_dashboard(d)
assert u == [], f'unknown metrics: {u}'; json.dumps(d)
print('dataprotection OK, panels:', len(d['panels']))
"
```
Expected: `dataprotection OK, panels: 10`.

- [ ] **Step 3: Commit**

```bash
git add grafana/gen/dataprotection.py
git commit -m "feat(grafana): add Data Protection dashboard builder"
```

---

## Task 7: Orchestrator + remove old generator and legacy dashboard

**Files:**
- Create: `grafana/build_dashboards.py`
- Delete: `grafana/build_overview.py`, `grafana/NBU Statistics-1629904585394.json`
- Overwrite (generated): `grafana/nbu-overview.json`
- Create (generated): `grafana/nbu-jobs.json`, `grafana/nbu-storage.json`, `grafana/nbu-dataprotection.json`

- [ ] **Step 1: Create `grafana/build_dashboards.py`**

```python
#!/usr/bin/env python3
"""Generate all nbu_exporter Grafana dashboards from grafana/gen/.

Run from the repo root:  python3 grafana/build_dashboards.py
Each dashboard is validated against the known nbu_* metric set before writing.
"""
import json
import sys

from grafana.gen import overview, jobs, storage, dataprotection
from grafana.gen.validate import check_dashboard

OUTPUTS = [
    ("grafana/nbu-overview.json", overview.build),
    ("grafana/nbu-jobs.json", jobs.build),
    ("grafana/nbu-storage.json", storage.build),
    ("grafana/nbu-dataprotection.json", dataprotection.build),
]


def main():
    failures = []
    for path, build in OUTPUTS:
        dash = build()
        unknown = check_dashboard(dash)
        if unknown:
            failures.append(f"{path}: unknown metrics {unknown}")
            continue
        with open(path, "w") as f:
            json.dump(dash, f, indent=2)
            f.write("\n")
        print(f"wrote {path} ({len(dash['panels'])} panels)")
    if failures:
        for msg in failures:
            print(f"ERROR: {msg}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
```

- [ ] **Step 2: Remove the old generator and legacy dashboard**

```bash
git rm grafana/build_overview.py
git rm "grafana/NBU Statistics-1629904585394.json"
```

- [ ] **Step 3: Generate all four dashboards**

Run: `PYTHONPATH=. python3 grafana/build_dashboards.py`
Expected output (order):
```
wrote grafana/nbu-overview.json (11 panels)
wrote grafana/nbu-jobs.json (11 panels)
wrote grafana/nbu-storage.json (7 panels)
wrote grafana/nbu-dataprotection.json (10 panels)
```

- [ ] **Step 4: Validate every generated JSON parses**

Run:
```bash
for f in grafana/nbu-overview.json grafana/nbu-jobs.json grafana/nbu-storage.json grafana/nbu-dataprotection.json; do
  python3 -m json.tool "$f" >/dev/null && echo "$f OK"
done
```
Expected: four `... OK` lines.

- [ ] **Step 5: Confirm uids are unique and tags present**

Run:
```bash
PYTHONPATH=. python3 -c "
import json, glob
uids=[]
for f in glob.glob('grafana/nbu-*.json'):
    d=json.load(open(f)); uids.append(d['uid']); assert 'netbackup' in d['tags'], f
assert len(uids)==len(set(uids)), uids
print('uids unique + tagged:', sorted(uids))
"
```
Expected: `uids unique + tagged: ['nbu-dataprotection', 'nbu-jobs', 'nbu-overview', 'nbu-storage']`.

- [ ] **Step 6: Commit**

```bash
git add grafana/build_dashboards.py grafana/nbu-overview.json grafana/nbu-jobs.json grafana/nbu-storage.json grafana/nbu-dataprotection.json
git commit -m "feat(grafana): generate four cross-linked dashboards, retire legacy + build_overview.py"
```

---

## Task 8: Documentation

**Files:**
- Modify: `docs/metrics.md`

- [ ] **Step 1: Add a dashboards note to `docs/metrics.md`**

Read `docs/metrics.md` first. Add a short "## Dashboards" section near the top or bottom (match the file's structure) stating:
- Dashboards are generated by `python3 grafana/build_dashboards.py` (never hand-edit the JSON).
- The four dashboards and their uids/focus: Overview (`nbu-overview`), Jobs (`nbu-jobs`), Storage (`nbu-storage`), Data Protection (`nbu-dataprotection`).
- They cross-link via the `netbackup` tag and use the `${datasource}` (+ `storage_unit`/`policy_type`) template variables.
- The legacy "NBU Statistics" dashboard was retired; its views live in Storage and Jobs.

Keep lines ≤120 chars (`.markdownlint.json`).

- [ ] **Step 2: Verify markdown line length**

Run: `awk 'length > 120 {print FILENAME":"NR": "length}' docs/metrics.md`
Expected: no output (no lines over 120).

- [ ] **Step 3: Commit**

```bash
git add docs/metrics.md
git commit -m "docs(metrics): document the generated four-dashboard layout"
```

---

## Task 9: Final verification gate

**Files:** none (verification)

- [ ] **Step 1: Regenerate clean and confirm no drift**

Run:
```bash
PYTHONPATH=. python3 grafana/build_dashboards.py
git diff --stat -- grafana/
```
Expected: generator runs clean; `git diff` shows no changes (the committed JSON already matches generator output — proving the JSON is generated, not hand-edited).

- [ ] **Step 2: Confirm legacy + old generator are gone**

Run:
```bash
test ! -f grafana/build_overview.py && test ! -f "grafana/NBU Statistics-1629904585394.json" && echo "cleanup OK"
ls grafana/nbu-*.json
```
Expected: `cleanup OK` and the four `nbu-*.json` files listed.

- [ ] **Step 3: Re-run the metric validator across all four**

Run:
```bash
PYTHONPATH=. python3 -c "
import json, glob
from grafana.gen.validate import check_dashboard
bad={f:check_dashboard(json.load(open(f))) for f in glob.glob('grafana/nbu-*.json')}
bad={k:v for k,v in bad.items() if v}
assert not bad, bad
print('all dashboards reference only known metrics')
"
```
Expected: `all dashboards reference only known metrics`.

- [ ] **Step 4: Final commit if any fixups were needed**

```bash
git add -A
git commit -m "chore(grafana): dashboard redesign verification fixups"
```

---

## Self-Review notes (for the executor)

- All Python is stdlib only — no `pip install`. If `from grafana.gen import ...` fails from the repo root, prefix commands with `PYTHONPATH=.` (the plan already does this in verification steps).
- `reset_ids()` MUST be the first line of every dashboard `build()` so panel ids are unique within each dashboard (they may repeat across dashboards — that is fine).
- Do not hand-edit any `grafana/nbu-*.json` — they are generator output. Task 9 Step 1 proves this by regenerating and asserting an empty `git diff`.
- The metric validator's `_normalize` only strips histogram suffixes for `nbu_job_duration_seconds`; it must NOT turn `nbu_jobs_count` into `nbu_jobs`. Keep that guard.
- Leave `.serena/project.yml` and any untracked `docs/veritas-11.2/` unstaged throughout.
- Cross-branch follow-up (NOT this branch): the quickstart-stack Grafana provisioning must be updated to mount all four dashboards (ideally a directory mount) and drop the legacy file. Documented in the spec; do not edit compose here.
