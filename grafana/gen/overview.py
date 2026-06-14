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
