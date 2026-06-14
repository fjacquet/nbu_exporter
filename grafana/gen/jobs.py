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
