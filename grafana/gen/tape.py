"""Tape & disk-pool infrastructure dashboard (API v12.0+, NBU 10.5+)."""

from grafana.gen import panels as p
from grafana.gen.variables import dashboard

_TAPE_MD = """\
## Infrastructure bande et disques / Tape & Disk Pool Infrastructure

```
┌──────────────────────────────────────────────────────────────────────────────┐
│  NBU Master Server                    Robot bande / Tape Library             │
│  ──────────────────                   ──────────────────────────             │
│                                                                              │
│  [Job DUPLICATION] ──▶ Lecteur bande ──▶ [Cartouche FULL]                   │
│                        (Drive UP/DOWN)    [Cartouche PARTIAL]                │
│                                           [Pool Scratch ↓ alerte]           │
│                                                                              │
│  [Disk Pool] ──▶ Volume UP/DOWN/UNKNOWN                                      │
│                  (surveillance état et capacité)                             │
└──────────────────────────────────────────────────────────────────────────────┘
```

> **Pré-requis**: NetBackup 10.5+ (API v12.0) et `collectors.tape.enabled: true` dans la config.
"""


def build():
    p.reset_ids()
    out = []

    # ── Schéma infrastructure bande ───────────────────────────────────────────
    out.append(p.text_panel("Infrastructure bande / Tape Infrastructure", _TAPE_MD, 0, 0, 24, 8))

    # ── Row 1: Tape Drives ────────────────────────────────────────────────────
    out.append(p.row("Lecteurs bande / Tape Drives", 8))

    out.append(p.stat(
        "Lecteurs UP / Drives UP",
        'sum(nbu_tape_drives_count{status="DRIVE_STATUS_UP", site=~"$site"})',
        0, 9, 6, 4,
        thresholds=[{"color": "green", "value": None}],
    ))
    out.append(p.stat(
        "Lecteurs DOWN / Drives DOWN",
        'sum(nbu_tape_drives_count{status="DRIVE_STATUS_DOWN", site=~"$site"})',
        6, 9, 6, 4,
        thresholds=[
            {"color": "green", "value": None},
            {"color": "red", "value": 1},
        ],
    ))
    out.append(p.stat(
        "Total lecteurs / Total drives",
        'sum(nbu_tape_drives_count{site=~"$site"})',
        12, 9, 6, 4,
        thresholds=[{"color": "blue", "value": None}],
    ))
    out.append(p.piechart(
        "Lecteurs par statut / Drives by status",
        'sum by (status) (nbu_tape_drives_count{site=~"$site"})',
        18, 9, 6, 4,
        legend="{{status}}",
    ))

    out.append(p.barchart(
        "Lecteurs par type / Drives by type",
        'sum by (drive_type, robot_type, status) (nbu_tape_drives_count{site=~"$site"})',
        0, 13, 12, 8,
        legend="{{drive_type}} / {{robot_type}} / {{status}}",
    ))
    out.append(p.timeseries(
        "Évolution état lecteurs / Drive status over time",
        [
            p.target('sum by (status) (nbu_tape_drives_count{site=~"$site"})', "{{status}}"),
        ],
        12, 13, 12, 8,
    ))

    # ── Row 2: Tape Media ─────────────────────────────────────────────────────
    out.append(p.row("Inventaire cartouches / Tape Media", 21))

    out.append(p.stat(
        "Total cartouches / Total media",
        'sum(nbu_tape_media_count{site=~"$site"})',
        0, 22, 6, 4,
        thresholds=[{"color": "blue", "value": None}],
    ))
    out.append(p.piechart(
        "Cartouches par pool / Media by pool",
        'sum by (pool) (nbu_tape_media_count{site=~"$site"})',
        6, 22, 9, 8,
        legend="{{pool}}",
    ))
    out.append(p.piechart(
        "Cartouches par type / Media by type",
        'sum by (media_type) (nbu_tape_media_count{site=~"$site"})',
        15, 22, 9, 8,
        legend="{{media_type}}",
    ))

    out.append(p.barchart(
        "Cartouches par pool et type / Media by pool & type",
        'sum by (pool, media_type) (nbu_tape_media_count{site=~"$site"})',
        0, 26, 24, 8,
        legend="{{pool}} / {{media_type}}",
    ))

    # ── Row 3: Volume Pools ───────────────────────────────────────────────────
    out.append(p.row("Pools de volumes bande / Tape Volume Pools", 34))

    out.append(p.stat(
        "Cartouches partiellement pleines / Partially full media",
        'sum(nbu_tape_pool_partially_full{site=~"$site"})',
        0, 35, 8, 4,
        thresholds=[
            {"color": "green", "value": None},
            {"color": "yellow", "value": 5},
            {"color": "red", "value": 20},
        ],
    ))
    out.append(p.barchart(
        "Partiellement pleines par pool / Partially full by pool",
        'sum by (pool_name, pool_type) (nbu_tape_pool_partially_full{site=~"$site"})',
        8, 35, 16, 4,
        legend="{{pool_name}} ({{pool_type}})",
    ))

    out.append(p.timeseries(
        "Évolution cartouches partielles / Partially full media over time",
        [p.target('sum by (pool_name) (nbu_tape_pool_partially_full{site=~"$site"})', "{{pool_name}}")],
        0, 39, 24, 7,
    ))

    # ── Row 4: Disk Pool Volumes ──────────────────────────────────────────────
    out.append(p.row("Volumes disque / Disk Pool Volumes", 46))

    out.append(p.stat(
        "Volumes UP",
        'sum(nbu_disk_pool_volume_count{state="UP", site=~"$site"})',
        0, 47, 6, 4,
        thresholds=[{"color": "green", "value": None}],
    ))
    out.append(p.stat(
        "Volumes DOWN",
        'sum(nbu_disk_pool_volume_count{state="DOWN", site=~"$site"})',
        6, 47, 6, 4,
        thresholds=[
            {"color": "green", "value": None},
            {"color": "red", "value": 1},
        ],
    ))
    out.append(p.stat(
        "Volumes UNKNOWN",
        'sum(nbu_disk_pool_volume_count{state="UNKNOWN", site=~"$site"})',
        12, 47, 6, 4,
        thresholds=[
            {"color": "green", "value": None},
            {"color": "orange", "value": 1},
        ],
    ))
    out.append(p.stat(
        "Total volumes disque / Total disk volumes",
        'sum(nbu_disk_pool_volume_count{site=~"$site"})',
        18, 47, 6, 4,
        thresholds=[{"color": "blue", "value": None}],
    ))

    out.append(p.barchart(
        "Volumes par pool et état / Volumes by pool & state",
        'sum by (pool_name, storage_category, state) (nbu_disk_pool_volume_count{site=~"$site"})',
        0, 51, 16, 8,
        legend="{{pool_name}} / {{storage_category}} / {{state}}",
    ))
    out.append(p.piechart(
        "Répartition état volumes / Volume state distribution",
        'sum by (state) (nbu_disk_pool_volume_count{site=~"$site"})',
        16, 51, 8, 8,
        legend="{{state}}",
    ))

    out.append(p.timeseries(
        "État volumes disque dans le temps / Disk volume state over time",
        [p.target('sum by (pool_name, state) (nbu_disk_pool_volume_count{site=~"$site"})',
                  "{{pool_name}} / {{state}}")],
        0, 59, 24, 7,
    ))

    return dashboard("nbu-tape", "NetBackup — Bande & Disques / Tape & Disk Pools", out)
