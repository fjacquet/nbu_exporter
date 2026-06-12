# Metrics Reference

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

## System Metrics

| Metric | Labels | Description |
|--------|--------|-------------|
| `nbu_up` | — | `1` if any collection succeeded, `0` if all collections failed |
| `nbu_api_version` | `version` | Currently active NetBackup API version (13.0, 12.0, or 3.0) |
| `nbu_response_time_ms` | — | NetBackup API response time in milliseconds |
| `nbu_last_scrape_timestamp_seconds` | `source` | Unix timestamp of the last successful collection (`source`: `storage` or `jobs`) |

## Label Encoding

Metrics use pipe-delimited keys internally, split into labels:

- **Storage**: `name|type|size` (e.g., "pool1|AdvancedDisk|free")
- **Jobs**: `action|policy_type|status` (e.g., "BACKUP|Standard|0")

## Prometheus Configuration

Add to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'netbackup'
    static_configs:
      - targets: ['localhost:2112']
    scrape_interval: 60s
    scrape_timeout: 30s
```

Alerting rules are provided in `grafana/alerts.yml` (load via `rule_files`).

## Grafana Dashboard

Two dashboards live in the `grafana/` directory:

- `grafana/nbu-overview.json` — fully templated overview (health, storage, jobs,
  durations). Recommended; works on any server via the `$storage_unit` /
  `$policy_type` template variables.
- `grafana/NBU Statistics-1629904585394.json` — the original 2021 dashboard, kept
  for reference.

To import: load the JSON into Grafana and select your Prometheus datasource.
