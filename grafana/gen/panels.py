"""Shared Grafana panel/layout helpers for the nbu_exporter dashboards."""

import re

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


# --- Multi-site query rewriting -------------------------------------------------
# Every dashboard filters its queries by the `site` template variable so series from
# multiple NetBackup primaries neither collapse together nor double-count. Instead of
# hand-editing each PromQL string, every expression flows through with_site(): it
# injects site=~"$site" into each nbu_* selector and adds `site` to any `by (...)`
# grouping. Both steps are idempotent, so already-wired exprs pass through unchanged.
SITE_MATCHER = 'site=~"$site"'

_SELECTOR_RE = re.compile(r"(nbu_[a-z0-9_]+)(\{[^{}]*\})?")
_BY_RE = re.compile(r"\bby\s*\(\s*([^)]*)\)")
# Vector aggregations that drop the `site` label unless it is named in `by (...)`.
# (Metric names like nbu_storage_max_… don't match — the keyword is `_`-bounded there.)
_AGG_RE = re.compile(r"\b(sum|avg|count|min|max|group|stddev|stdvar|topk|bottomk|count_values)\b")


def _filter_selectors(expr):
    def repl(m):
        name, braces = m.group(1), m.group(2)
        if braces is None:
            return f"{name}{{{SITE_MATCHER}}}"
        inner = braces[1:-1].strip()
        if re.search(r"(^|,)\s*site\s*=", inner):  # already filtered by site
            return m.group(0)
        return f"{name}{{{inner},{SITE_MATCHER}}}" if inner else f"{name}{{{SITE_MATCHER}}}"
    return _SELECTOR_RE.sub(repl, expr)


def _group_by_site(expr):
    def repl(m):
        labels = [g.strip() for g in m.group(1).split(",") if g.strip()]
        if "site" in labels:
            return m.group(0)
        return "by (" + ", ".join(["site"] + labels) + ")"
    return _BY_RE.sub(repl, expr)


def with_site(expr):
    """Filter an expression by the site variable and group per-site (idempotent)."""
    return _group_by_site(_filter_selectors(expr))


def _carries_site(expr):
    """True if the rewritten expr yields series that keep a distinguishing `site` label."""
    if "by (site" in expr:           # explicitly grouped per-site
        return True
    return not _AGG_RE.search(expr)  # no label-dropping aggregation wraps it


def _legend_with_site(expr, legend):
    """Prefix a legend with {{site}} when the series carry a per-site label."""
    if not _carries_site(expr) or "{{site}}" in legend:
        return legend
    return "{{site}} / " + legend if legend else "{{site}}"


def gridpos(x, y, w, h):
    return {"x": x, "y": y, "w": w, "h": h}


def target(expr, legend="", instant=False):
    expr = with_site(expr)
    return {
        "datasource": ds(),
        "expr": expr,
        "legendFormat": _legend_with_site(expr, legend),
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
         text_mode="auto", legend="", color_mode="value", repeat=None):
    panel = {
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
    if repeat:
        # Render one panel per value of the given template variable (e.g. site).
        panel["repeat"] = repeat
        panel["repeatDirection"] = "h"
    return panel


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
            "expr": with_site(expr),
            "format": "table",
            "instant": True,
            "refId": "A",
        }],
        "transformations": [
            {"id": "labelsToFields", "options": {}},
            {"id": "organize", "options": {"excludeByName": {"Time": True, "Value": True, "__name__": True, "job": True, "instance": True}}},
        ],
    }
