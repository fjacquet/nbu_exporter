"""Multi-site comparison dashboard: cross-site backup volume, replication, and compliance."""

from grafana.gen import panels as p
from grafana.gen.variables import dashboard, client_filter_var

# ── Architecture diagram (Markdown) ──────────────────────────────────────────
_ARCH_MD = """\
## Architecture multi-sites / Multi-site Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  Site Principal / Primary Site               Site Distant / Remote Site      │
│  ─────────────────────────────               ──────────────────────────────  │
│                                                                              │
│  [Clients]                                    [Clients locaux]               │
│     │                                              │                         │
│     ├─── BACKUP ───▶ NBU Master 1                  ├─── BACKUP ──▶ NBU Master 2 │
│                           │                                      │           │
│                           │── DUPLICATION ──▶ [Robot bande]      │           │
│                           │                                      │           │
│                           └── AIR (IMPORT) ──────────────────────┘           │
│                                                   ▲                          │
│                                              Réplication                     │
│                                              inter-sites                     │
└─────────────────────────────────────────────────────────────────────────────┘
```

**Métriques clés** : `nbu_jobs_bytes`, `nbu_status_count`, `nbu_client_last_job_success_seconds`
**Alerte divergence** : si un site sauvegarde < 30% du volume moyen inter-sites → `NbuInterSiteDivergence`
"""


def _stat_bg(title, description, expr, x, y, w, h, unit="none", thresholds=None):
    if thresholds is None:
        thresholds = [{"color": "blue", "value": None}]
    return {
        "type": "stat",
        "id": p.nid(),
        "title": title,
        "description": description,
        "datasource": p.ds(),
        "gridPos": p.gridpos(x, y, w, h),
        "fieldConfig": {
            "defaults": {
                "unit": unit,
                "color": {"mode": "thresholds"},
                "thresholds": {"mode": "absolute", "steps": thresholds},
            },
            "overrides": [],
        },
        "options": {
            "colorMode": "background",
            "graphMode": "none",
            "justifyMode": "center",
            "orientation": "auto",
            "reduceOptions": {"calcs": ["lastNotNull"], "fields": "", "values": False},
            "textMode": "auto",
        },
        "targets": [p.target(expr, instant=True)],
    }


def _stat_value(title, description, expr, x, y, w, h, unit="none", thresholds=None):
    if thresholds is None:
        thresholds = [{"color": "blue", "value": None}]
    return {
        "type": "stat",
        "id": p.nid(),
        "title": title,
        "description": description,
        "datasource": p.ds(),
        "gridPos": p.gridpos(x, y, w, h),
        "fieldConfig": {
            "defaults": {
                "unit": unit,
                "color": {"mode": "thresholds"},
                "thresholds": {"mode": "absolute", "steps": thresholds},
            },
            "overrides": [],
        },
        "options": {
            "colorMode": "value",
            "graphMode": "area",
            "justifyMode": "center",
            "orientation": "auto",
            "reduceOptions": {"calcs": ["lastNotNull"], "fields": "", "values": False},
            "textMode": "auto",
        },
        "targets": [p.target(expr, instant=True)],
    }


def _gauge(title, description, expr, x, y, w, h, legend=""):
    return {
        "id": p.nid(),
        "type": "gauge",
        "title": title,
        "description": description,
        "gridPos": {"x": x, "y": y, "w": w, "h": h},
        "datasource": p.ds(),
        "targets": [{"datasource": p.ds(), "expr": expr, "instant": True, "refId": "A", "legendFormat": legend}],
        "options": {
            "reduceOptions": {"calcs": ["lastNotNull"], "fields": "", "values": False},
            "orientation": "auto",
            "showThresholdLabels": False,
            "showThresholdMarkers": True,
        },
        "fieldConfig": {
            "defaults": {
                "unit": "percent",
                "min": 0,
                "max": 100,
                "color": {"mode": "thresholds"},
                "thresholds": {
                    "mode": "absolute",
                    "steps": [
                        {"color": "red", "value": None},
                        {"color": "yellow", "value": 70},
                        {"color": "green", "value": 90},
                    ],
                },
            },
            "overrides": [],
        },
    }


def _table_missing(title, description, expr, x, y, w, h, value_col):
    return {
        "id": p.nid(),
        "type": "table",
        "title": title,
        "description": description,
        "gridPos": {"x": x, "y": y, "w": w, "h": h},
        "datasource": p.ds(),
        "targets": [{"datasource": p.ds(), "expr": expr, "instant": True, "format": "table", "refId": "A"}],
        "transformations": [
            {
                "id": "organize",
                "options": {
                    "excludeByName": {"Time": True, "__name__": True, "action": True},
                    "indexByName": {"site": 0, "client": 1, "policy": 2, "Value": 3},
                    "renameByName": {"site": "Site", "client": "Client", "policy": "Politique / Policy", "Value": value_col},
                },
            }
        ],
        "options": {
            "showHeader": True,
            "sortBy": [{"displayName": value_col, "desc": True}],
            "footer": {"show": True, "reducer": ["count"], "countRows": True},
        },
        "fieldConfig": {
            "defaults": {"custom": {"align": "left", "filterable": True}},
            "overrides": [
                {
                    "matcher": {"id": "byName", "options": value_col},
                    "properties": [
                        {"id": "unit", "value": "s"},
                        {"id": "custom.displayMode", "value": "color-background"},
                        {
                            "id": "thresholds",
                            "value": {
                                "mode": "absolute",
                                "steps": [
                                    {"color": "yellow", "value": None},
                                    {"color": "orange", "value": 172800},
                                    {"color": "red", "value": 259200},
                                ],
                            },
                        },
                        {"id": "color", "value": {"mode": "thresholds"}},
                    ],
                },
                {"matcher": {"id": "byName", "options": "Site"}, "properties": [{"id": "custom.width", "value": 100}]},
            ],
        },
    }


def build():
    p.reset_ids()
    out = []

    # ── Architecture diagram ──────────────────────────────────────────────────
    out.append(p.text_panel("Architecture multi-sites / Multi-site Architecture", _ARCH_MD, 0, 0, 24, 10))

    # ── Row 1: Vue d'ensemble multi-sites / Multi-site Overview ───────────────
    out.append(p.row("Vue d'ensemble multi-sites / Multi-site Overview", 10))

    out.append(_stat_value(
        "Total jobs (fenêtre de scraping)",
        "Nombre total de jobs toutes phases (BACKUP + DUPLICATION + IMPORT + RESTORE) sur tous les sites sélectionnés.",
        'sum(nbu_jobs_count{site=~"$site"})',
        0, 11, 5, 4,
        thresholds=[{"color": "blue", "value": None}],
    ))
    out.append(_stat_value(
        "Volume BACKUP total",
        "Total des octets transférés par les jobs BACKUP sur tous les sites.",
        'sum(nbu_jobs_bytes{action="BACKUP", site=~"$site"})',
        5, 11, 5, 4,
        unit="decbytes",
        thresholds=[{"color": "blue", "value": None}],
    ))
    out.append(_stat_bg(
        "Clients actifs (24h)",
        "Clients ayant eu au moins un BACKUP réussi dans les dernières 24h, tous sites.",
        'count(nbu_client_last_job_success_seconds{action="BACKUP", client=~"$client_filter", site=~"$site"} > (time() - 86400)) or on() vector(0)',
        10, 11, 5, 4,
        thresholds=[{"color": "red", "value": None}, {"color": "yellow", "value": 1}, {"color": "green", "value": 5}],
    ))
    out.append(_stat_bg(
        "Clients sans backup (>25h)",
        "Clients dont le dernier BACKUP dépasse 25h — violation du SLA journalier.",
        'count(time() - nbu_client_last_job_success_seconds{action="BACKUP", client=~"$client_filter", site=~"$site"} > 90000) or on() vector(0)',
        15, 11, 4, 4,
        thresholds=[{"color": "green", "value": None}, {"color": "yellow", "value": 1}, {"color": "red", "value": 3}],
    ))
    out.append(_stat_bg(
        "Clients sans réplication (>28h)",
        "Clients dont le dernier IMPORT dépasse 28h — potentiellement non répliqués sur le site secondaire.",
        'count(time() - nbu_client_last_job_success_seconds{action="IMPORT", client=~"$client_filter", site=~"$site"} > 100800) or on() vector(0)',
        19, 11, 5, 4,
        thresholds=[{"color": "green", "value": None}, {"color": "yellow", "value": 1}, {"color": "red", "value": 3}],
    ))

    # ── Row 2: Taux de réussite par site / Success Rate per Site ──────────────
    out.append(p.row("Taux de réussite par site / Success Rate per Site", 15))

    out.append(_gauge(
        "Taux BACKUP (tous sites)",
        "Ratio jobs BACKUP réussis / total dans la fenêtre de scraping, tous sites.",
        'sum(nbu_status_count{action="BACKUP", status="0", site=~"$site"}) / clamp_min(sum(nbu_status_count{action="BACKUP", site=~"$site"}), 1) * 100',
        0, 16, 8, 6,
        legend="BACKUP",
    ))
    out.append(_gauge(
        "Taux DUPLICATION (tous sites)",
        "Ratio jobs de copie bande (DUPLICATION) réussis / total.",
        'sum(nbu_status_count{action="DUPLICATION", status="0", site=~"$site"}) / clamp_min(sum(nbu_status_count{action="DUPLICATION", site=~"$site"}), 1) * 100',
        8, 16, 8, 6,
        legend="DUPLICATION",
    ))
    out.append(_gauge(
        "Taux IMPORT / AIR (tous sites)",
        "Ratio jobs IMPORT réussis / total. Mesure la fiabilité de la réplication inter-site.",
        'sum(nbu_status_count{action="IMPORT", status="0", site=~"$site"}) / clamp_min(sum(nbu_status_count{action="IMPORT", site=~"$site"}), 1) * 100',
        16, 16, 8, 6,
        legend="IMPORT",
    ))

    # ── Row 3: Volume de sauvegarde par site / Backup Volume per Site ─────────
    out.append(p.row("Volume de sauvegarde par site / Backup Volume per Site", 22))

    out.append(p.barchart(
        "Volume BACKUP par site / Backup volume per site",
        'sum by (site) (nbu_jobs_bytes{action="BACKUP", site=~"$site"})',
        0, 23, 12, 8,
        legend="{{site}}",
        unit="decbytes",
    ))
    out.append(p.timeseries(
        "Évolution volume BACKUP par site / Backup volume over time per site",
        [p.target('sum by (site) (nbu_jobs_bytes{action="BACKUP", site=~"$site"})', "{{site}}")],
        12, 23, 12, 8,
        unit="decbytes",
    ))

    # ── Row 4: Activité par site / Jobs per Site ──────────────────────────────
    out.append(p.row("Activité par site / Activity per Site", 31))

    out.append(p.barchart(
        "Nombre de jobs par site et phase / Job count per site and phase",
        'sum by (site, action) (nbu_jobs_count{site=~"$site"})',
        0, 32, 12, 8,
        legend="{{site}} — {{action}}",
    ))
    out.append(p.timeseries(
        "Jobs BACKUP dans le temps par site / BACKUP jobs over time per site",
        [p.target('sum by (site) (nbu_jobs_count{action="BACKUP", site=~"$site"})', "{{site}}")],
        12, 32, 12, 8,
    ))

    # ── Row 5: Conformité BACKUP par site / BACKUP Compliance Timeline ─────────
    out.append(p.row("Conformité BACKUP par site / BACKUP Compliance Timeline", 40))

    out.append(p.state_timeline(
        "Clients conformes BACKUP par site (< 25h) / Compliant clients per site",
        'clamp_max(time() - nbu_client_last_job_success_seconds{action="BACKUP", client=~"$client_filter", site=~"$site"} < 90000, 1)',
        0, 41, 16, 12,
        legend="{{site}} — {{client}}",
        thresholds=[{"color": "red", "value": None}, {"color": "green", "value": 0.5}],
    ))
    out.append(p.piechart(
        "Répartition jobs par site / Jobs distribution per site",
        'sum by (site) (nbu_jobs_count{action="BACKUP", site=~"$site"})',
        16, 41, 8, 6,
        legend="{{site}}",
    ))
    out.append(p.bar_gauge(
        "Taux BACKUP par site / BACKUP success rate per site",
        'sum by (site) (nbu_status_count{action="BACKUP", status="0", site=~"$site"}) / clamp_min(sum by (site) (nbu_status_count{action="BACKUP", site=~"$site"}), 1) * 100',
        16, 47, 8, 6,
        unit="percent",
        thresholds=[{"color": "red", "value": None}, {"color": "yellow", "value": 70}, {"color": "green", "value": 90}],
        legend="{{site}}",
    ))

    # ── Row 6: Réplication inter-sites / Inter-site Replication ──────────────
    out.append(p.row("Réplication inter-sites / Inter-site Replication", 53))

    out.append(_stat_value(
        "Volume IMPORT total",
        "Octets reçus via IMPORT (Auto Image Replication) sur tous les sites sélectionnés.",
        'sum(nbu_jobs_bytes{action="IMPORT", site=~"$site"}) or on() vector(0)',
        0, 54, 6, 4,
        unit="decbytes",
    ))
    out.append(_stat_value(
        "Jobs IMPORT réussis",
        "Nombre total de jobs IMPORT réussis dans la fenêtre de scraping.",
        'sum(nbu_status_count{action="IMPORT", status="0", site=~"$site"}) or on() vector(0)',
        6, 54, 6, 4,
    ))
    out.append(p.barchart(
        "Volume IMPORT par site / Replication volume per site",
        'sum by (site) (nbu_jobs_bytes{action="IMPORT", site=~"$site"})',
        12, 54, 12, 8,
        legend="{{site}}",
        unit="decbytes",
    ))
    out.append(p.timeseries(
        "Volume IMPORT dans le temps par site / Replication volume over time",
        [p.target('sum by (site) (nbu_jobs_bytes{action="IMPORT", site=~"$site"})', "{{site}}")],
        0, 58, 12, 8,
        unit="decbytes",
    ))
    out.append(p.timeseries(
        "Jobs IMPORT dans le temps par site / IMPORT jobs over time per site",
        [p.target('sum by (site) (nbu_status_count{action="IMPORT", site=~"$site"})', "{{site}}")],
        12, 58, 12, 8,
    ))

    # ── Row 7: Divergence inter-sites / Inter-site Divergence ─────────────────
    out.append(p.row("Divergence inter-sites / Inter-site Divergence", 66))

    out.append(p.timeseries(
        "Ratio BACKUP site / moyenne inter-sites (1.0 = normal, <0.3 = alerte)",
        [p.target(
            'sum by (site) (nbu_jobs_bytes{action="BACKUP", site=~"$site"}) / on() group_left() avg(sum by (site) (nbu_jobs_bytes{action="BACKUP", site=~"$site"}))',
            "{{site}}",
        )],
        0, 67, 12, 8,
        unit="percentunit",
    ))
    out.append(p.timeseries(
        "BACKUP vs IMPORT par site / BACKUP vs IMPORT bytes per site",
        [
            p.target('sum by (site) (nbu_jobs_bytes{action="BACKUP", site=~"$site"})', "BACKUP {{site}}"),
            p.target('sum by (site) (nbu_jobs_bytes{action="IMPORT", site=~"$site"})', "IMPORT {{site}}"),
        ],
        12, 67, 12, 8,
        unit="decbytes",
    ))

    # ── Row 8: Clients en alerte multi-sites / Multi-site Client Alerts ────────
    out.append(p.row("Clients en alerte / Clients in Alert", 75))

    out.append(_table_missing(
        "Clients sans backup récent (>25h) / Clients missing recent backup",
        "Liste des clients en violation du SLA de sauvegarde (>25h), avec le site d'appartenance.",
        'sort_desc(time() - nbu_client_last_job_success_seconds{action="BACKUP", client=~"$client_filter", site=~"$site"}) > 90000',
        0, 76, 12, 10,
        "Retard backup / Overdue",
    ))
    out.append(_table_missing(
        "Clients sans réplication récente (>28h) / Clients missing recent replication",
        "Clients dont la réplication IMPORT dépasse 28h — potentiellement non répliqués sur le site secondaire.",
        'sort_desc(time() - nbu_client_last_job_success_seconds{action="IMPORT", client=~"$client_filter", site=~"$site"}) > 100800',
        12, 76, 12, 10,
        "Retard réplication / Replication lag",
    ))

    # ── Row 9: Bande par site / Tape per Site (NBU 10.5+) ─────────────────────
    out.append(p.row("Bande par site / Tape per Site (NBU 10.5+)", 86))

    out.append(p.barchart(
        "Lecteurs UP par site / Tape drives UP per site",
        'sum by (site) (nbu_tape_drives_count{status="DRIVE_STATUS_UP", site=~"$site"})',
        0, 87, 8, 8,
        legend="{{site}}",
    ))
    out.append(p.barchart(
        "Cartouches par site / Tape media per site",
        'sum by (site) (nbu_tape_media_count{site=~"$site"})',
        8, 87, 8, 8,
        legend="{{site}}",
    ))
    out.append(p.timeseries(
        "Lecteurs DOWN dans le temps par site / DOWN drives over time per site",
        [p.target('sum by (site) (nbu_tape_drives_count{status="DRIVE_STATUS_DOWN", site=~"$site"})', "{{site}}")],
        16, 87, 8, 8,
    ))

    return dashboard(
        "nbu-multisite",
        "NetBackup — Comparaison multi-sites / Multi-site Comparison",
        out,
        extra_vars=[client_filter_var()],
    )
