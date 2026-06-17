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


# The site template variable contract (multi-site dashboards). Every dashboard must
# carry one `site` selector and filter every query by it, so series from multiple
# NetBackup primaries neither collapse together nor double-count.
SITE_FILTER = 'site=~"$site"'
SITE_VAR_QUERY = "label_values(nbu_up, site)"


def _var_query(var):
    """The PromQL query string of a Grafana query variable (dict or bare string form)."""
    q = var.get("query", "")
    return q.get("query", "") if isinstance(q, dict) else q


def check_site_wiring(dash):
    """Return a sorted list of multi-site wiring problems for the dashboard.

    Enforces the multi-site dashboard contract:
      - exactly one `site` query variable, sourced from label_values(nbu_up, site),
        multi-value + includeAll, and the first non-datasource templating entry;
      - every panel target expression is filtered by site=~"$site".
    """
    problems = []
    tmpl = dash.get("templating", {}).get("list", [])

    site_vars = [v for v in tmpl if v.get("name") == "site"]
    if not site_vars:
        problems.append("missing 'site' template variable")
    else:
        if len(site_vars) > 1:
            problems.append("duplicate 'site' template variable")
        sv = site_vars[0]
        if SITE_VAR_QUERY not in _var_query(sv):
            problems.append(f"site var not sourced from {SITE_VAR_QUERY}")
        if not sv.get("multi") or not sv.get("includeAll"):
            problems.append("site var must be multi-value + includeAll")
        non_ds = [v for v in tmpl if v.get("type") != "datasource"]
        if not non_ds or non_ds[0].get("name") != "site":
            problems.append("site var must be the first non-datasource templating entry")

    for panel in dash.get("panels", []):
        for expr in _iter_exprs(panel):
            if SITE_FILTER not in expr:
                problems.append(f"expr not filtered by site: {expr[:70]}")

    return sorted(set(problems))
