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
