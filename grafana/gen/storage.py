"""Storage dashboard: capacity utilization, units, limits."""

from grafana.gen import panels as p
from grafana.gen.variables import dashboard, storage_unit_var


def build():
    p.reset_ids()
    out = []

    out.append(p.row("Capacité / Capacity", 0))
    out.append(p.gauge("% Utilisé / % Used",
                       'sum by (name) (nbu_disk_bytes{name=~"$storage_unit",size="used"}) '
                       '/ sum by (name) (nbu_disk_bytes{name=~"$storage_unit"}) * 100',
                       0, 1, 8, 8))
    out.append(p.timeseries("Capacité utilisée / Used capacity",
                            [p.target('sum by (name) (nbu_disk_bytes{name=~"$storage_unit",size="used"})', "{{name}} used"),
                             p.target('nbu_disk_capacity_bytes{name=~"$storage_unit"}', "{{name}} total")],
                            8, 1, 16, 8, unit="bytes"))

    out.append(p.row("Unités / Units", 9))
    out.append(p.table_info("Unités de stockage / Storage units",
                            'nbu_storage_info{name=~"$storage_unit"}', 0, 10, 16, 8))
    out.append(p.stat("Jobs simultanés max / Max concurrent jobs",
                      'nbu_storage_max_concurrent_jobs{name=~"$storage_unit"}', 16, 10, 4, 8,
                      legend="{{name}}"))
    out.append(p.stat("Taille fragment max / Max fragment size",
                      'nbu_storage_max_fragment_size_bytes{name=~"$storage_unit"}', 20, 10, 4, 8,
                      unit="bytes", legend="{{name}}"))

    return dashboard("nbu-storage", "NetBackup — Stockage / Storage", out, extra_vars=[storage_unit_var()])
