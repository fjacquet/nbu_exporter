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
