"""Overview dashboard: one-screen health + headline KPIs, links to domain dashboards."""

from grafana.gen import panels as p
from grafana.gen.variables import dashboard

UP_MAP = [{"type": "value", "options": {"0": {"text": "DOWN", "color": "red"},
                                        "1": {"text": "UP", "color": "green"}}}]


def build():
    p.reset_ids()
    out = []

    # Per-site availability: one tile per NetBackup primary (repeats over $site), so a
    # down master is immediately obvious even with "All" sites selected.
    out.append(p.row("Sites", 0))
    out.append(p.stat("$site", "nbu_up", 0, 1, 4, 4,
                      mappings=UP_MAP, color_mode="background", legend="{{site}}", repeat="site",
                      thresholds=[{"color": "red", "value": None}, {"color": "green", "value": 1}]))

    out.append(p.row("Health / Santé", 5))
    out.append(p.stat("API version / Version API", "nbu_api_version", 0, 6, 6, 4,
                      text_mode="name", legend="{{version}}"))
    out.append(p.stat("Scrape staleness / Fraîcheur scrape",
                      "time() - nbu_last_scrape_timestamp_seconds", 6, 6, 8, 4,
                      unit="s", legend="{{source}}",
                      thresholds=[{"color": "green", "value": None}, {"color": "yellow", "value": 300}, {"color": "red", "value": 900}]))
    out.append(p.timeseries("API latency / Latence API",
                            [p.target("nbu_response_time_ms", "response / réponse")],
                            14, 6, 10, 4, unit="ms"))

    out.append(p.row("Key indicators / Indicateurs clés", 10))
    out.append(p.stat("Backup success rate / Taux succès BACKUP",
                      'sum(nbu_status_count{action="BACKUP",status="0"}) '
                      '/ clamp_min(sum(nbu_status_count{action="BACKUP"}), 1) * 100',
                      0, 11, 5, 5, unit="percent", color_mode="background",
                      thresholds=[{"color": "red", "value": None}, {"color": "yellow", "value": 90}, {"color": "green", "value": 99}]))
    out.append(p.stat("Failed backups / Échecs BACKUP",
                      'sum(nbu_status_count{action="BACKUP"}) '
                      '- sum(nbu_status_count{action="BACKUP",status="0"})',
                      5, 11, 5, 5,
                      thresholds=[{"color": "green", "value": None}, {"color": "red", "value": 1}]))
    out.append(p.stat("Storage % used / Stockage % utilisé",
                      'sum(nbu_disk_bytes{size="used"}) / clamp_min(sum(nbu_disk_bytes), 1) * 100',
                      10, 11, 4, 5, unit="percent",
                      thresholds=[{"color": "green", "value": None}, {"color": "yellow", "value": 80}, {"color": "red", "value": 90}]))
    out.append(p.stat("Infected files / Fichiers infectés",
                      "sum(nbu_malware_files_infected)", 14, 11, 5, 5,
                      thresholds=[{"color": "green", "value": None}, {"color": "red", "value": 1}]))
    out.append(p.barchart("Alerts by severity / Alertes par sévérité",
                          "sum by (severity) (nbu_alerts_count)", 19, 11, 5, 5,
                          legend="{{severity}}"))

    return dashboard("nbu-overview", "NetBackup — Overview / Vue d'ensemble", out)
