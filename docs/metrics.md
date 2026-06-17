# Metrics Reference

!!! info "The `site` label"
    **Every metric below also carries a `site` label** — its first label — identifying the
    NetBackup primary it came from. Single-site deployments have one `site` value; a
    `nbuservers:` list produces one per configured site. The per-metric tables list only the
    *additional* labels, so a `—` in the Labels column means the series carries `site` only.

## Job Metrics

| Metric | Labels | Description |
|--------|--------|-------------|
| `nbu_jobs_count` | `action`, `policy_type`, `status` | Number of jobs by type, policy, and status |
| `nbu_jobs_bytes` | `action`, `policy_type`, `status` | Total bytes transferred by jobs |
| `nbu_status_count` | `action`, `status` | Job count aggregated by action and status |
| `nbu_jobs_state_count` | `action`, `state` | Job count per lifecycle state (e.g. `ACTIVE`, `QUEUED`, `DONE`) |
| `nbu_jobs_queued_count` | `action`, `reason` | Queued job count per NetBackup queue reason code |
| `nbu_jobs_files_count` | `action`, `policy_type` | Total number of files processed by jobs |
| `nbu_jobs_dedup_ratio` | `action`, `policy_type` | Mean deduplication ratio across jobs (emitted only when jobs exist) |
| `nbu_job_duration_seconds` | `action`, `policy_type` | Histogram of completed job durations in seconds |

The `nbu_job_duration_seconds` histogram covers completed jobs only (those with an
`EndTime` after `StartTime`). Bucket upper bounds (seconds): 60, 300, 900, 1800,
3600, 7200, 14400, 28800, 86400.

## Storage Metrics

| Metric | Labels | Description |
|--------|--------|-------------|
| `nbu_disk_bytes` | `name`, `type`, `size` | Storage unit capacity; `size` is `free` or `used` |
| `nbu_disk_capacity_bytes` | `name`, `type` | Authoritative total capacity reported by the API |
| `nbu_storage_max_concurrent_jobs` | `name`, `type` | Maximum concurrent jobs the unit accepts |
| `nbu_storage_max_fragment_size_bytes` | `name`, `type` | Maximum fragment size in bytes |
| `nbu_storage_info` | `name`, `type`, `subtype`, `is_cloud`, `worm_capable`, `use_worm`, `replication_capable`, `instant_access` | Storage unit capabilities (value always `1`) |

`nbu_disk_capacity_bytes` is a separate metric (not a `size="total"` label) so that
`sum(nbu_disk_bytes{name=X})` continues to equal total capacity (free + used).

!!! note
    Tape storage units are excluded from metrics collection.

## Opt-in collectors

These metrics are exposed by optional sub-collectors. **All default to disabled** — enable
only the ones your appliance and account permissions support. They are graceful: a failing
endpoint is logged and skipped without affecting core storage/jobs metrics or flipping
`nbu_up` to `0`. The alerts/malware/catalog/SLO collectors require NetBackup 11.2 (API
`version=14.0`); the **`tape`** collector works on older releases too (`/storage/drives` on
NBU 10.0+, `/storage/tape-media` and `/storage/robots-device-hosts` on 10.5+).

| Metric | Type | Labels | Source endpoint | Source API attribute |
|--------|------|--------|-----------------|----------------------|
| `nbu_alerts_count` | Gauge | `severity`, `category` | `GET /manage/alerts` | Alerts grouped by `severity` + `category` |
| `nbu_malware_files_scanned` | Gauge | — | `GET /malware/latest-scan-results` | Sum of `numberOfFilesScanned` |
| `nbu_malware_files_infected` | Gauge | — | `GET /malware/latest-scan-results` | Sum of `numberOfFilesImpacted` |
| `nbu_malware_scan_count` | Gauge | `status` | `GET /malware/latest-scan-results` | Results grouped by `scanState` |
| `nbu_catalog_images_count` | Gauge | `malware_status`, `anomaly_status` | `GET /catalog/images` | `meta.pagination.count` per filter combination |
| `nbu_slo_count` | Gauge | — | `GET /servicecatalog/slos` | Total number of `data[]` entries |
| `nbu_tape_drives_count` | Gauge | `state`, `drive_type`, `robot_type` | `GET /storage/drives` | Drives grouped by `driveStatus`/`driveType`/`robotType` |
| `nbu_tape_drive_info` | Gauge | `drive_name`, `media_server`, `drive_type`, `robot_number`, `state` | `GET /storage/drives` | One series per drive (value always `1`) |
| `nbu_tape_media_count` | Gauge | `media_type`, `status` | `GET /storage/tape-media` | Tape volumes grouped by `mediaType` + `mediaStatus` |
| `nbu_tape_robot_device_hosts` | Gauge | — | `GET /storage/robots-device-hosts` | Count of device hosts with robots configured |
| `nbu_client_last_successful_backup_timestamp_seconds` | Gauge | `client` | `GET /admin/jobs` (per allowlisted client) | `endTime` of the latest `status=0` BACKUP for the client |

Notes on the implemented attributes (these differ from early guesses):

- **Malware infected count** reads `numberOfFilesImpacted`, not
  `numberOfFilesInfected`. The 11.2 `latest-scan-results` response uses
  `numberOfFilesImpacted`.
- **Malware scan status** is grouped by `scanState` (enum values such as
  `SCAN_COMPLETED` / `SCAN_FAILED`), exposed via the `status` label.
- **Catalog posture** is collected with count-only queries (`page[limit]=1`,
  reading `meta.pagination.count`) issued once per curated combination of
  `malwareStatus` × `anomalyStatus`, keeping label cardinality bounded.
- **SLO count** is a single unlabeled gauge. The 11.2 SLO response has no
  per-SLO enforcement-type attribute, so the originally planned
  `enforcement_type` label was dropped.
- **Tape collector** (`collectors.tape`) is one collector over three endpoints with
  per-endpoint graceful degradation. Tape media is offset-paginated and aggregated
  client-side; `robot_type` is read from the per-drive attribute (there is no bulk
  robot-listing endpoint, so a standalone robot type/count metric is not exposed);
  `mediaStatus` is a free-form NetBackup status string. `drive_state` is the
  `driveStatus` value with the `DRIVE_STATUS_` prefix stripped (`UP`/`DOWN`/`MIXED`/`DISABLED`).
- **Per-client collector** (`collectors.perClient`) requires an explicit `allowlist` because the
  `client` label is high-cardinality (hundreds of clients) — an **empty allowlist emits nothing**.
  For each allowlisted client it issues one targeted `/admin/jobs` query
  (`filter clientName eq '<c>' and jobType eq 'BACKUP' and status eq 0`, `sort=-endTime`,
  `page[limit]=1`) and emits that job's `endTime`, so it always reflects the last success with no
  lookback gap. Feed a "no successful backup in N hours" alert with
  `time() - nbu_client_last_successful_backup_timestamp_seconds > N*3600`.

### Enabling the collectors

Add a `collectors` block to your `config.yaml` (each collector is a
`{ enabled: false }` toggle; all default to disabled):

```yaml
collectors:
  alerts:  { enabled: false }
  malware: { enabled: false }
  catalog: { enabled: false }
  slo:     { enabled: false }
  tape:    { enabled: false }
  perClient:
    enabled: false
    allowlist: []   # exact client names; empty => no per-client series
```

!!! note "Job metrics and the missing 11.2 `admin.yaml`"
    Job metrics (`GET /admin/jobs`) are validated against the NetBackup 11.0
    (`version=13.0`) spec because `admin.yaml` is absent from the local
    `docs/veritas-11.2/` bundle. The endpoint is backward-compatible under
    `version=14.0`, so the existing job metrics remain correct. Obtaining the
    11.2 `admin.yaml` is a follow-up item to confirm no new job attributes were
    added.

## System Metrics

| Metric | Labels | Description |
|--------|--------|-------------|
| `nbu_up` | — | `1` if any collection succeeded, `0` if all collections failed |
| `nbu_api_version` | `version` | Currently active NetBackup API version (14.0, 13.0, 12.0, or 10.0) |
| `nbu_response_time_ms` | — | NetBackup API response time in milliseconds |
| `nbu_last_scrape_timestamp_seconds` | `source` | Unix timestamp of the last successful collection (`source`: `storage` or `jobs`) |

## Label Encoding

Metrics use pipe-delimited keys internally, split into labels:

- **Storage**: `name|type|size` (e.g., "pool1|AdvancedDisk|free")
- **Jobs**: `action|policy_type|status` (e.g., "BACKUP|Standard|0")

The `site` label is prepended at emission time (it is not part of these internal keys), so every
emitted series is `site` followed by the labels above.

## Prometheus Configuration

Add to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'netbackup'
    static_configs:
      - targets: ['localhost:9440']
    scrape_interval: 60s
    scrape_timeout: 30s
```

Alerting rules are provided in `grafana/alerts.yml` (load via `rule_files`).

## Dashboards

The Grafana dashboards are **generated** by `python3 grafana/build_dashboards.py`
(pure Python stdlib). Never hand-edit the JSON in `grafana/` — the generator is the
single source of truth and a metric-reference validator fails the build on any
unknown `nbu_*` name. Regenerate after changing the builders in `grafana/gen/`.

Four focused dashboards live in the `grafana/` directory:

| File | uid | Focus |
|------|-----|-------|
| `grafana/nbu-overview.json` | `nbu-overview` | One-screen health + headline KPIs |
| `grafana/nbu-jobs.json` | `nbu-jobs` | Backup outcomes, states, volume, queue, durations, dedup |
| `grafana/nbu-storage.json` | `nbu-storage` | Capacity utilization, storage units, limits |
| `grafana/nbu-dataprotection.json` | `nbu-dataprotection` | Alerts, malware scans, catalog posture, SLOs (11.2) |

The dashboards cross-link to each other via the shared `netbackup` tag (a tag-based
dashboard-links dropdown) and use the `${datasource}` template variable so they work
on any server. Jobs adds a `policy_type` variable and Storage adds a `storage_unit`
variable for per-unit / per-policy filtering.

The legacy "NBU Statistics" dashboard (the original 2021 export) was retired; its
views now live in the Storage and Jobs dashboards.

To import: load the JSON into Grafana and select your Prometheus datasource.
