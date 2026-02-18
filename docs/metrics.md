# Metrics Reference

## Job Metrics

| Metric | Labels | Description |
|--------|--------|-------------|
| `nbu_jobs_count` | `action`, `policy_type`, `status` | Number of jobs by type, policy, and status |
| `nbu_jobs_size_bytes` | `action`, `policy_type`, `status` | Total bytes transferred by jobs |
| `nbu_jobs_status_count` | `action`, `status` | Job count aggregated by action and status |

## Storage Metrics

| Metric | Labels | Description |
|--------|--------|-------------|
| `nbu_storage_bytes` | `name`, `type`, `size` | Storage unit capacity |

The `size` label is either `free` (available capacity in bytes) or `used` (used capacity in bytes).

!!! note
    Tape storage units are excluded from metrics collection.

## System Metrics

| Metric | Labels | Description |
|--------|--------|-------------|
| `nbu_api_version` | `version` | Currently active NetBackup API version (13.0, 12.0, or 3.0) |

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

## Grafana Dashboard

A pre-built Grafana dashboard is included in the `grafana/` directory:

1. Import `grafana/NBU Statistics-1629904585394.json` into your Grafana instance
2. Configure the Prometheus datasource
3. View NetBackup job statistics and storage utilization
