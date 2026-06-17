"""Backup lifecycle dashboard: per-client compliance across all phases (BACKUP → DUPLICATION → IMPORT)."""

from grafana.gen.panels import ds, nid, gridpos, reset_ids, piechart, state_timeline, bar_gauge, text_panel
from grafana.gen.variables import dashboard, client_filter_var

# ── Lifecycle flow diagram (Markdown) ─────────────────────────────────────────
_FLOW_MD = """\
## Cycle de vie d'une sauvegarde / Backup Lifecycle Flow

```
┌──────────────────────────────────────────────────────────────────────────────┐
│  Client                  Site Principal / Primary Site          Site distant  │
│  ──────                  ───────────────────────────────        ────────────  │
│                                                                               │
│  [Données]               ┌─────────────────┐                                 │
│     │                    │  NBU Master 1   │                                  │
│     │── BACKUP ──────────▶  (sauvegarde)   │                                  │
│                          │                 │── DUPLICATION ──▶ [Bande / Tape] │
│                          │                 │                                  │
│                          │                 │── AIR ──────────▶ NBU Master 2   │
│                          └─────────────────┘                       │          │
│                                                                    │          │
│                                                               IMPORT reçu     │
│                                                                               │
└──────────────────────────────────────────────────────────────────────────────┘
```

| Phase | Métrique | SLA |
|-------|---------|-----|
| BACKUP | `nbu_client_last_job_success_seconds{action="BACKUP"}` | < 25h |
| DUPLICATION | `nbu_client_last_job_success_seconds{action="DUPLICATION"}` | < 26h |
| IMPORT (AIR) | `nbu_client_last_job_success_seconds{action="IMPORT"}` | < 28h |
"""


def _stat_bg(title, description, expr, x, y, w, h, thresholds, unit="none"):
    return {
        "type": "stat",
        "id": nid(),
        "title": title,
        "description": description,
        "datasource": ds(),
        "gridPos": gridpos(x, y, w, h),
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
        "targets": [{"datasource": ds(), "expr": expr, "instant": True, "refId": "A"}],
    }


def _stat_sparkline(title, description, expr, x, y, w, h, unit="none", thresholds=None):
    if thresholds is None:
        thresholds = [{"color": "text", "value": None}]
    return {
        "type": "stat",
        "id": nid(),
        "title": title,
        "description": description,
        "datasource": ds(),
        "gridPos": gridpos(x, y, w, h),
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
        "targets": [{"datasource": ds(), "expr": expr, "refId": "A"}],
    }


def _gauge(title, description, expr, x, y, w, h, legend=""):
    return {
        "id": nid(),
        "type": "gauge",
        "title": title,
        "description": description,
        "gridPos": {"x": x, "y": y, "w": w, "h": h},
        "datasource": ds(),
        "targets": [{"datasource": ds(), "expr": expr, "instant": True, "refId": "A", "legendFormat": legend}],
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


def _table_lifecycle():
    return {
        "id": nid(),
        "type": "table",
        "title": "Ancienneté de la dernière réussite par client et phase / Age of last success per client & phase",
        "description": (
            "Ancienneté (en secondes) de la dernière réussite pour chaque client/politique/phase. "
            "Jaune = proche du seuil (>23h), Orange = dépassé 24h, Rouge = violation SLA (>25h). "
            "Trier la colonne Ancienneté pour voir les clients les plus critiques en premier."
        ),
        "gridPos": {"x": 0, "y": 46, "w": 24, "h": 12},
        "datasource": ds(),
        "targets": [{"datasource": ds(), "expr": 'sort_desc(time() - nbu_client_last_job_success_seconds{client=~"$client_filter", site=~"$site"})', "instant": True, "format": "table", "refId": "A"}],
        "transformations": [
            {
                "id": "organize",
                "options": {
                    "excludeByName": {"Time": True, "__name__": True},
                    "indexByName": {"client": 0, "policy": 1, "action": 2, "site": 3, "Value": 4},
                    "renameByName": {
                        "client": "Client",
                        "policy": "Politique / Policy",
                        "action": "Phase",
                        "site": "Site",
                        "Value": "Ancienneté / Age",
                    },
                },
            }
        ],
        "options": {
            "showHeader": True,
            "sortBy": [{"displayName": "Ancienneté / Age", "desc": True}],
            "footer": {"show": False},
        },
        "fieldConfig": {
            "defaults": {"custom": {"align": "left", "displayMode": "color-text", "filterable": True}},
            "overrides": [
                {
                    "matcher": {"id": "byName", "options": "Ancienneté / Age"},
                    "properties": [
                        {"id": "unit", "value": "s"},
                        {"id": "custom.displayMode", "value": "color-background"},
                        {
                            "id": "thresholds",
                            "value": {
                                "mode": "absolute",
                                "steps": [
                                    {"color": "green", "value": None},
                                    {"color": "yellow", "value": 82800},
                                    {"color": "orange", "value": 86400},
                                    {"color": "red", "value": 90000},
                                ],
                            },
                        },
                        {"id": "color", "value": {"mode": "thresholds"}},
                    ],
                },
                {
                    "matcher": {"id": "byName", "options": "Phase"},
                    "properties": [
                        {
                            "id": "mappings",
                            "value": [
                                {"type": "value", "options": {"BACKUP": {"text": "Sauvegarde primaire", "color": "blue", "index": 0}}},
                                {"type": "value", "options": {"DUPLICATION": {"text": "Copie bande", "color": "purple", "index": 1}}},
                                {"type": "value", "options": {"IMPORT": {"text": "Réplication AIR", "color": "orange", "index": 2}}},
                            ],
                        }
                    ],
                },
                {"matcher": {"id": "byName", "options": "Site"}, "properties": [{"id": "custom.width", "value": 100}]},
            ],
        },
    }


def _table_overdue(title, description, action_filter, x, y, w, h, value_col, age_threshold):
    return {
        "id": nid(),
        "type": "table",
        "title": title,
        "description": description,
        "gridPos": {"x": x, "y": y, "w": w, "h": h},
        "datasource": ds(),
        "targets": [{"datasource": ds(), "expr": f'sort_desc(time() - nbu_client_last_job_success_seconds{{action="{action_filter}", client=~"$client_filter", site=~"$site"}}) > {age_threshold}', "instant": True, "format": "table", "refId": "A"}],
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


def _timeseries_phase_bars(y):
    return {
        "id": nid(),
        "type": "timeseries",
        "title": "Jobs réussis par phase / Successful jobs by phase",
        "description": "Nombre de jobs status=0 par type dans la fenêtre de scraping. Les barres représentent le rythme d'activité de chaque phase du cycle de vie.",
        "gridPos": {"x": 0, "y": y, "w": 16, "h": 9},
        "datasource": ds(),
        "targets": [
            {"datasource": ds(), "expr": 'sum by (action) (nbu_client_jobs_count{status="0", action=~"BACKUP|DUPLICATION|IMPORT|RESTORE", client=~"$client_filter", site=~"$site"})', "refId": "A", "legendFormat": "{{action}}"}
        ],
        "options": {
            "tooltip": {"mode": "multi", "sort": "desc"},
            "legend": {"displayMode": "table", "placement": "right", "calcs": ["lastNotNull", "max", "sum"]},
        },
        "fieldConfig": {
            "defaults": {
                "unit": "short",
                "custom": {"drawStyle": "bars", "fillOpacity": 70, "lineWidth": 0, "barAlignment": 0, "spanNulls": False, "showPoints": "never"},
            },
            "overrides": [
                {"matcher": {"id": "byName", "options": "BACKUP"}, "properties": [{"id": "color", "value": {"mode": "fixed", "fixedColor": "blue"}}]},
                {"matcher": {"id": "byName", "options": "DUPLICATION"}, "properties": [{"id": "color", "value": {"mode": "fixed", "fixedColor": "purple"}}]},
                {"matcher": {"id": "byName", "options": "IMPORT"}, "properties": [{"id": "color", "value": {"mode": "fixed", "fixedColor": "orange"}}]},
                {"matcher": {"id": "byName", "options": "RESTORE"}, "properties": [{"id": "color", "value": {"mode": "fixed", "fixedColor": "green"}}]},
            ],
        },
    }


def _timeseries_volume(y):
    return {
        "id": nid(),
        "type": "timeseries",
        "title": "Volume transféré par phase / Data transferred by phase",
        "description": "Octets transférés par type de job. Note : le filtre Client ne s'applique pas ici — nbu_jobs_bytes n'a pas de label client.",
        "gridPos": {"x": 0, "y": y, "w": 24, "h": 8},
        "datasource": ds(),
        "targets": [
            {"datasource": ds(), "expr": 'sum by (action) (nbu_jobs_bytes{action=~"BACKUP|DUPLICATION|IMPORT", site=~"$site"})', "refId": "A", "legendFormat": "{{action}}"}
        ],
        "options": {
            "tooltip": {"mode": "multi", "sort": "desc"},
            "legend": {"displayMode": "table", "placement": "bottom", "calcs": ["lastNotNull", "max"]},
        },
        "fieldConfig": {
            "defaults": {
                "unit": "decbytes",
                "custom": {"drawStyle": "line", "fillOpacity": 20, "lineWidth": 2, "spanNulls": False},
            },
            "overrides": [
                {"matcher": {"id": "byName", "options": "BACKUP"}, "properties": [{"id": "color", "value": {"mode": "fixed", "fixedColor": "blue"}}]},
                {"matcher": {"id": "byName", "options": "DUPLICATION"}, "properties": [{"id": "color", "value": {"mode": "fixed", "fixedColor": "purple"}}]},
                {"matcher": {"id": "byName", "options": "IMPORT"}, "properties": [{"id": "color", "value": {"mode": "fixed", "fixedColor": "orange"}}]},
            ],
        },
    }


def _timeseries_duration(y):
    return {
        "id": nid(),
        "type": "timeseries",
        "title": "Durée P95 et P50 des jobs BACKUP par politique / P95 & P50 BACKUP duration per policy",
        "description": "Percentiles 50 (médiane) et 95 de la durée des jobs BACKUP par type de politique. Une dérive vers le haut indique un ralentissement (charge réseau, volumétrie croissante).",
        "gridPos": {"x": 0, "y": y, "w": 24, "h": 8},
        "datasource": ds(),
        "targets": [
            {"datasource": ds(), "expr": 'histogram_quantile(0.95, sum by (policy_type, le) (rate(nbu_job_duration_seconds_bucket{action="BACKUP", site=~"$site"}[$__rate_interval])))', "refId": "A", "legendFormat": "P95 {{policy_type}}"},
            {"datasource": ds(), "expr": 'histogram_quantile(0.50, sum by (policy_type, le) (rate(nbu_job_duration_seconds_bucket{action="BACKUP", site=~"$site"}[$__rate_interval])))', "refId": "B", "legendFormat": "P50 {{policy_type}}"},
        ],
        "options": {
            "tooltip": {"mode": "multi", "sort": "desc"},
            "legend": {"displayMode": "table", "placement": "bottom", "calcs": ["lastNotNull", "max"]},
        },
        "fieldConfig": {
            "defaults": {"unit": "s", "custom": {"drawStyle": "line", "fillOpacity": 10, "lineWidth": 2, "spanNulls": True}, "color": {"mode": "palette-classic"}},
            "overrides": [],
        },
    }


def build():
    reset_ids()
    panels = [
        # ── Schéma du cycle de vie / Lifecycle flow diagram ───────────────────
        text_panel(
            "Schéma du cycle de vie / Lifecycle Flow",
            _FLOW_MD,
            0, 0, 24, 10,
        ),

        # ── Row 1: KPIs ───────────────────────────────────────────────────────
        {"id": nid(), "type": "row", "title": "Vue d'ensemble / Overview", "gridPos": {"x": 0, "y": 10, "w": 24, "h": 1}, "collapsed": False},
        _stat_bg(
            "Clients sauvegardés (24h)",
            "Clients ayant eu au moins un backup primaire réussi dans les dernières 24h.",
            'count(nbu_client_last_job_success_seconds{action="BACKUP", client=~"$client_filter", site=~"$site"} > (time() - 86400)) or on() vector(0)',
            0, 11, 6, 4,
            [{"color": "red", "value": None}, {"color": "yellow", "value": 1}, {"color": "green", "value": 5}],
        ),
        _stat_bg(
            "Clients sans backup (>25h)",
            "Clients en violation du SLA journalier — dernier backup primaire > 25h.",
            'count(time() - nbu_client_last_job_success_seconds{action="BACKUP", client=~"$client_filter", site=~"$site"} > 90000) or on() vector(0)',
            6, 11, 6, 4,
            [{"color": "green", "value": None}, {"color": "yellow", "value": 1}, {"color": "red", "value": 3}],
        ),
        _stat_sparkline(
            "Copies bande (DUPLICATION)",
            "Jobs de duplication vers bande réussis dans la fenêtre de scraping.",
            'sum(nbu_client_jobs_count{action="DUPLICATION", status="0", client=~"$client_filter", site=~"$site"}) or on() vector(0)',
            12, 11, 6, 4,
        ),
        _stat_sparkline(
            "Réplications IMPORT (AIR)",
            "Jobs IMPORT réussis (réplication inter-site via Auto Image Replication).",
            'sum(nbu_client_jobs_count{action="IMPORT", status="0", client=~"$client_filter", site=~"$site"}) or on() vector(0)',
            18, 11, 6, 4,
        ),

        # ── Row 2: Taux de réussite par phase ─────────────────────────────────
        {"id": nid(), "type": "row", "title": "Taux de réussite par phase / Success Rate per Phase", "gridPos": {"x": 0, "y": 15, "w": 24, "h": 1}, "collapsed": False},
        _gauge(
            "Sauvegarde primaire / Primary Backup",
            "Taux de succès des jobs BACKUP dans la fenêtre de scraping. Objectif : >90%.",
            'sum(nbu_client_jobs_count{action="BACKUP", status="0", client=~"$client_filter", site=~"$site"}) / clamp_min(sum(nbu_client_jobs_count{action="BACKUP", client=~"$client_filter", site=~"$site"}), 1) * 100',
            0, 16, 8, 6,
            legend="BACKUP",
        ),
        _gauge(
            "Copie bande / Tape Duplication",
            "Taux de succès des jobs DUPLICATION (copie vers bande via SLP). Objectif : >90%.",
            'sum(nbu_client_jobs_count{action="DUPLICATION", status="0", client=~"$client_filter", site=~"$site"}) / clamp_min(sum(nbu_client_jobs_count{action="DUPLICATION", client=~"$client_filter", site=~"$site"}), 1) * 100',
            8, 16, 8, 6,
            legend="DUPLICATION",
        ),
        _gauge(
            "Réplication inter-site (IMPORT)",
            "Taux de succès des jobs IMPORT (Auto Image Replication). Objectif : >90%.",
            'sum(nbu_client_jobs_count{action="IMPORT", status="0", client=~"$client_filter", site=~"$site"}) / clamp_min(sum(nbu_client_jobs_count{action="IMPORT", client=~"$client_filter", site=~"$site"}), 1) * 100',
            16, 16, 8, 6,
            legend="IMPORT",
        ),

        # ── Row 3: Conformité clients dans le temps ───────────────────────────
        {"id": nid(), "type": "row", "title": "Conformité BACKUP dans le temps / BACKUP Compliance Timeline", "gridPos": {"x": 0, "y": 22, "w": 24, "h": 1}, "collapsed": False},
        state_timeline(
            "Clients conformes (BACKUP < 25h) / Compliant clients (BACKUP < 25h)",
            'clamp_max(time() - nbu_client_last_job_success_seconds{action="BACKUP", client=~"$client_filter", site=~"$site"} < 90000, 1)',
            0, 23, 16, 12,
            legend="{{client}}",
            thresholds=[{"color": "red", "value": None}, {"color": "green", "value": 0.5}],
        ),
        piechart(
            "Distribution de conformité / Compliance distribution",
            'clamp_max(sum(nbu_client_last_job_success_seconds{action="BACKUP", client=~"$client_filter", site=~"$site"} > (time() - 90000)) or on() vector(0), 9999)',
            16, 23, 8, 6,
            legend="Conformes",
        ),
        bar_gauge(
            "Clients par ancienneté du backup / Clients by backup age",
            'sort_desc(time() - nbu_client_last_job_success_seconds{action="BACKUP", client=~"$client_filter", site=~"$site"})',
            16, 29, 8, 6,
            unit="s",
            thresholds=[
                {"color": "green", "value": None},
                {"color": "yellow", "value": 82800},
                {"color": "orange", "value": 86400},
                {"color": "red", "value": 90000},
            ],
            legend="{{client}}",
        ),

        # ── Row 4: Cycle de vie complet par client ────────────────────────────
        {"id": nid(), "type": "row", "title": "Cycle de vie complet par client / Full Lifecycle per Client", "gridPos": {"x": 0, "y": 35, "w": 24, "h": 1}, "collapsed": False},
        _table_lifecycle(),

        # ── Row 5: Activité dans le temps ─────────────────────────────────────
        {"id": nid(), "type": "row", "title": "Activité dans le temps / Activity Over Time", "gridPos": {"x": 0, "y": 58, "w": 24, "h": 1}, "collapsed": False},
        _timeseries_phase_bars(59),
        # Piechart of current window jobs by phase (next to the timeseries)
        piechart(
            "Répartition des jobs par phase / Jobs by phase",
            'sum by (action) (nbu_client_jobs_count{status="0", client=~"$client_filter", site=~"$site"})',
            16, 59, 8, 9,
            legend="{{action}}",
        ),
        _timeseries_volume(68),

        # ── Row 6: Clients en alerte ──────────────────────────────────────────
        {"id": nid(), "type": "row", "title": "Clients en alerte / Clients in Alert", "gridPos": {"x": 0, "y": 76, "w": 24, "h": 1}, "collapsed": False},
        _table_overdue(
            "Clients dont le backup dépasse 25h / Clients with backup older than 25h",
            "Clients en violation du SLA — dernier BACKUP > 25h. Les plus critiques en tête.",
            "BACKUP", 0, 77, 12, 10, "Retard backup / Overdue", 90000,
        ),
        _table_overdue(
            "Clients sans copie bande récente (>26h) / Clients missing tape copy (>26h)",
            "Clients dont la dernière duplication vers bande date de plus de 26h.",
            "DUPLICATION", 12, 77, 12, 10, "Retard copie bande / Tape copy overdue", 93600,
        ),

        # ── Row 7: Performance ────────────────────────────────────────────────
        {"id": nid(), "type": "row", "title": "Performance / Job Duration", "gridPos": {"x": 0, "y": 87, "w": 24, "h": 1}, "collapsed": False},
        _timeseries_duration(88),
    ]

    return dashboard(
        "nbu-lifecycle",
        "NetBackup — Cycle de vie des sauvegardes / Backup Lifecycle",
        panels,
        extra_vars=[client_filter_var()],
    )
