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


def site_var():
    # One series per NetBackup primary regardless of which collectors are enabled,
    # so nbu_up is the most reliable source for the site list.
    return _query_var("site", "Site", "nbu_up", "site")


def storage_unit_var():
    return _query_var("storage_unit", "Storage unit", "nbu_disk_bytes", "name")


def policy_type_var():
    return _query_var("policy_type", "Policy type", "nbu_jobs_count", "policy_type")


def client_filter_var():
    # Sourced from the opt-in per-client metric; resolves to no clients (".*" allValue)
    # when collectors.perClient is disabled, so lifecycle/multi-site panels degrade cleanly.
    return _query_var("client_filter", "Client", "nbu_client_last_job_success_seconds", "client")


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
    """Assemble a full dashboard dict.

    The `site` selector is wired onto every dashboard as the first non-datasource
    variable; `extra_vars` (dashboard-specific selectors) follow it.
    """
    templating = {"list": [datasource_var(), site_var()] + (extra_vars or [])}
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
