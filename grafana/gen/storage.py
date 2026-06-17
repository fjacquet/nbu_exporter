"""Storage dashboard: capacity utilization, units, limits."""

from grafana.gen import panels as p
from grafana.gen.variables import dashboard, storage_unit_var


def build():
    p.reset_ids()
    out = []

    out.append(p.row("Capacity / Capacité", 0))
    out.append(p.gauge("% Used / % Utilisé",
                       'sum by (name) (nbu_disk_bytes{name=~"$storage_unit",size="used"}) '
                       '/ sum by (name) (nbu_disk_bytes{name=~"$storage_unit"}) * 100',
                       0, 1, 8, 8))
    out.append(p.timeseries("Used capacity / Capacité utilisée",
                            [p.target('sum by (name) (nbu_disk_bytes{name=~"$storage_unit",size="used"})', "{{name}} used"),
                             p.target('nbu_disk_capacity_bytes{name=~"$storage_unit"}', "{{name}} total")],
                            8, 1, 16, 8, unit="bytes"))

    out.append(p.row("Units / Unités", 9))
    out.append(p.table_info("Storage units / Unités de stockage",
                            'nbu_storage_info{name=~"$storage_unit"}', 0, 10, 16, 8))
    out.append(p.stat("Max concurrent jobs / Jobs simultanés max",
                      'nbu_storage_max_concurrent_jobs{name=~"$storage_unit"}', 16, 10, 4, 8,
                      legend="{{name}}"))
    out.append(p.stat("Max fragment size / Taille fragment max",
                      'nbu_storage_max_fragment_size_bytes{name=~"$storage_unit"}', 20, 10, 4, 8,
                      unit="bytes", legend="{{name}}"))

    return dashboard("nbu-storage", "NetBackup — Storage / Stockage", out, extra_vars=[storage_unit_var()])
