"""Data Protection dashboard: alerts, malware scans, catalog posture, SLOs (11.2 collectors)."""

from grafana.gen import panels as p
from grafana.gen.variables import dashboard


def build():
    p.reset_ids()
    out = []

    out.append(p.row("Alerts / Alertes", 0))
    out.append(p.barchart("Alerts by severity / Alertes par sévérité",
                          "sum by (severity) (nbu_alerts_count)", 0, 1, 12, 8, legend="{{severity}}"))
    out.append(p.table_info("Alerts by category / Alertes par catégorie",
                            "sum by (category, severity) (nbu_alerts_count)", 12, 1, 12, 8))

    out.append(p.row("Malware", 9))
    out.append(p.timeseries("Files scanned vs infected / Fichiers scannés vs infectés",
                            [p.target("nbu_malware_files_scanned", "scanned / scannés"),
                             p.target("nbu_malware_files_infected", "infected / infectés")],
                            0, 10, 12, 8))
    out.append(p.barchart("Scan status / Statut des scans",
                          "sum by (status) (nbu_malware_scan_count)", 12, 10, 12, 8, legend="{{status}}"))

    out.append(p.row("Catalog & SLO / Catalogue & SLO", 18))
    out.append(p.barchart("Malware posture / Posture malware",
                          "sum by (malware_status) (nbu_catalog_images_count)", 0, 19, 8, 8, legend="{{malware_status}}"))
    out.append(p.barchart("Anomaly posture / Posture anomalies",
                          "sum by (anomaly_status) (nbu_catalog_images_count)", 8, 19, 8, 8, legend="{{anomaly_status}}"))
    out.append(p.stat("Configured SLOs / SLO configurés",
                      "sum(nbu_slo_count)", 16, 19, 8, 8))

    return dashboard("nbu-dataprotection", "NetBackup — Data Protection / Protection des données", out)
