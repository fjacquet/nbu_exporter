#!/usr/bin/env python3
"""Generate grafana/nbu-overview.json — a fully-templated bilingual (FR / EN)
NetBackup overview dashboard for the nbu_exporter metric set.

Run:  python3 grafana/build_overview.py
"""
import json

DS = "${datasource}"
panels = []
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


# ---------------------------------------------------------------- Health row
panels.append(row("Santé / Health", 0))
up_map = [
    {"type": "value", "options": {"0": {"text": "DOWN", "color": "red"}, "1": {"text": "UP", "color": "green"}}},
]
panels.append(stat("Disponibilité / Availability", "nbu_up", 0, 1, 5, 4,
                   mappings=up_map, color_mode="background",
                   thresholds=[{"color": "red", "value": None}, {"color": "green", "value": 1}]))
panels.append(stat("Version API / API version", "nbu_api_version", 5, 1, 4, 4,
                   text_mode="name", legend="{{version}}"))
panels.append(stat("Fraîcheur scrape / Scrape staleness",
                   "time() - nbu_last_scrape_timestamp_seconds", 9, 1, 6, 4,
                   unit="s", legend="{{source}}",
                   thresholds=[{"color": "green", "value": None}, {"color": "yellow", "value": 300}, {"color": "red", "value": 900}]))
panels.append(timeseries("Latence API / API latency",
                         [target("nbu_response_time_ms", "réponse / response")],
                         15, 1, 9, 4, unit="ms"))

# --------------------------------------------------------------- Storage row
panels.append(row("Stockage / Storage", 5))
panels.append(gauge("% Utilisé / % Used",
                    'sum by (name) (nbu_disk_bytes{name=~"$storage_unit",size="used"}) '
                    '/ sum by (name) (nbu_disk_bytes{name=~"$storage_unit"}) * 100',
                    0, 6, 8, 8))
panels.append(timeseries("Capacité utilisée / Used capacity",
                         [target('sum by (name) (nbu_disk_bytes{name=~"$storage_unit",size="used"})', "{{name}} used"),
                          target('nbu_disk_capacity_bytes{name=~"$storage_unit"}', "{{name}} total")],
                         8, 6, 16, 8, unit="bytes"))
panels.append(table_info("Unités de stockage / Storage units",
                         'nbu_storage_info{name=~"$storage_unit"}', 0, 14, 16, 7))
panels.append(stat("Jobs simultanés max / Max concurrent jobs",
                   'nbu_storage_max_concurrent_jobs{name=~"$storage_unit"}', 16, 14, 8, 7,
                   legend="{{name}}", text_mode="auto"))

# ------------------------------------------------------------------ Jobs row
panels.append(row("Sauvegardes / Jobs", 21))
panels.append(stat("Taux de succès BACKUP / Backup success rate",
                   'sum(nbu_status_count{action="BACKUP",status="0"}) '
                   '/ clamp_min(sum(nbu_status_count{action="BACKUP"}), 1) * 100',
                   0, 22, 6, 8, unit="percent", color_mode="background",
                   thresholds=[{"color": "red", "value": None}, {"color": "yellow", "value": 90}, {"color": "green", "value": 99}]))
panels.append(piechart("États des jobs / Job states",
                       "sum by (state) (nbu_jobs_state_count)", 6, 22, 9, 8))
panels.append(barchart("Jobs par politique / Jobs by policy",
                       'sum by (policy_type) (nbu_jobs_count{action="BACKUP",policy_type=~"$policy_type"})',
                       15, 22, 9, 8))
panels.append(timeseries("Volume sauvegardé / Backup volume",
                         [target('sum by (policy_type) (nbu_jobs_bytes{action="BACKUP",policy_type=~"$policy_type"})', "{{policy_type}}")],
                         0, 30, 12, 8, unit="bytes", stack=True))
panels.append(barchart("Jobs en file / Queued jobs",
                       'sum by (reason) (nbu_jobs_queued_count)', 12, 30, 12, 8, legend="{{reason}}"))

# -------------------------------------------------------------- Duration row
panels.append(row("Durées / Durations", 38))
panels.append(timeseries("Durée jobs p50/p95 / Job duration p50/p95",
                         [target('histogram_quantile(0.95, sum by (le, policy_type) (nbu_job_duration_seconds_bucket{policy_type=~"$policy_type"}))', "p95 {{policy_type}}"),
                          target('histogram_quantile(0.50, sum by (le, policy_type) (nbu_job_duration_seconds_bucket{policy_type=~"$policy_type"}))', "p50 {{policy_type}}")],
                         0, 39, 16, 8, unit="s", fill=0))
panels.append(stat("Fichiers traités / Files processed",
                   'sum(nbu_jobs_files_count{policy_type=~"$policy_type"})', 16, 39, 4, 8))
panels.append(stat("Dédup moyenne / Mean dedup ratio",
                   'avg(nbu_jobs_dedup_ratio{policy_type=~"$policy_type"})', 20, 39, 4, 8, unit="none"))

# --------------------------------------------------------------- Templating
templating = {"list": [
    {
        "name": "datasource",
        "type": "datasource",
        "query": "prometheus",
        "label": "Datasource",
        "current": {},
        "hide": 0,
        "refresh": 1,
    },
    {
        "name": "storage_unit",
        "type": "query",
        "datasource": ds(),
        "label": "Storage unit",
        "query": {"query": "label_values(nbu_disk_bytes, name)", "refId": "StandardVariableQuery"},
        "includeAll": True,
        "multi": True,
        "allValue": ".*",
        "current": {"text": "All", "value": "$__all"},
        "refresh": 2,
        "sort": 1,
    },
    {
        "name": "policy_type",
        "type": "query",
        "datasource": ds(),
        "label": "Policy type",
        "query": {"query": "label_values(nbu_jobs_count, policy_type)", "refId": "StandardVariableQuery"},
        "includeAll": True,
        "multi": True,
        "allValue": ".*",
        "current": {"text": "All", "value": "$__all"},
        "refresh": 2,
        "sort": 1,
    },
]}

dashboard = {
    "uid": "nbu-overview",
    "title": "NetBackup — Vue d'ensemble / Overview",
    "tags": ["netbackup", "nbu", "backup"],
    "schemaVersion": 39,
    "version": 1,
    "editable": True,
    "graphTooltip": 1,
    "time": {"from": "now-24h", "to": "now"},
    "timepicker": {},
    "timezone": "",
    "refresh": "1m",
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

with open("grafana/nbu-overview.json", "w") as f:
    json.dump(dashboard, f, indent=2)
    f.write("\n")
print(f"wrote grafana/nbu-overview.json with {len(panels)} panels")
