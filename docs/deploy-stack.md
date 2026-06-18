# Deploying the full monitoring stack

This guide deploys nbu_exporter, Prometheus, and Grafana as a single docker-compose stack with all dashboards and alerting rules pre-wired.

**Result:** 7 Grafana dashboards (Overview, Jobs, Storage, Data Protection, Tape, Lifecycle, Multi-site) + Prometheus alerting rules, all running in 3 containers.

---

## Prerequisites

| Requirement | Details |
|---|---|
| NetBackup version | 10.0 or later (10.5+ for tape/disk metrics) |
| NetBackup API key | Generated from NBU Admin Console → API Keys |
| Network access | Exporter host → NBU master on port 1556 |
| Docker + Compose | v2.x (`docker compose version`) |

---

## 1. Get the repo

```bash
git clone https://github.com/fjacquet/nbu_exporter.git
cd nbu_exporter
```

---

## 2. Generate a NetBackup API key

In the NetBackup Admin Console or web UI: **Security → API Keys → Add**.

The key needs read access to: jobs, storage, catalog, alerts.

---

## 3. Configure

### Single site

```bash
cp docs/config-examples/config-auto-detect.yaml config.yaml
```

Edit `config.yaml` and set your values:

```yaml
server:
    host: "0.0.0.0"
    port: "9440"
    uri: "/metrics"
    scrapingInterval: "1h"
    logName: "log/nbu-exporter.log"

nbuserver:
    scheme: "https"
    uri: "/netbackup"
    host: "nbu-master.my.domain"   # ← NBU master FQDN
    port: "1556"
    apiKey: "your-api-key-here"    # ← generated above
    insecureSkipVerify: false

collectors:
    tape:
        enabled: true    # requires NBU 10.5+; set false if older
    perClient:
        enabled: false   # opt-in: see "Optional collectors" below
```

### Two sites (multi-site)

```bash
cp docs/config-examples/config-multisite.yaml config.yaml
```

Edit to replace the two `host` / `apiKey` / `site` placeholders. Each `site` value becomes a label on every metric — keep it short (e.g. `paris`, `lyon`).

---

## 4. Start the stack

```bash
docker compose up -d
```

This starts:
- **nbu_exporter** on port 9440
- **Prometheus** on port 9090
- **Grafana** on port 3000 (login: `admin` / `admin`)

---

## 5. Verify

```bash
# Exporter health
curl http://localhost:9440/health

# Metrics endpoint (should list ~20+ nbu_* metrics)
curl -s http://localhost:9440/metrics | grep "^nbu_" | cut -d'{' -f1 | sort -u

# Prometheus scraping (should return "1")
curl -s 'http://localhost:9090/api/v1/query?query=up{job="netbackup"}' \
  | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['result'][0]['value'][1])"
```

Open Grafana at `http://localhost:3000` → Dashboards → Browse → **NetBackup** folder.

---

## 6. Alerting rules

Prometheus loads four rule files automatically (mounted into `/etc/prometheus/rules/`
by docker-compose and listed under `rule_files:` in `prometheus.yml`):

| File | Alerts |
|---|---|
| `deploy/prometheus/nbu.rules.yml` | Core: exporter down, scrape failures, metrics stale, storage full |
| `deploy/prometheus/rules-perclient.yml` | Per-client compliance: backup staleness, tape-copy / replication freshness, failure rate |
| `deploy/prometheus/rules-tape.yml` | Tape & disk health: drive down/disabled, scratch-pool low, disk-pool volume degraded |
| `deploy/prometheus/rules-multisite.yml` | Inter-site backup-volume divergence |

The per-client, tape, and multi-site rules reference the opt-in `nbu_client_*` and
`nbu_tape_*` / `nbu_disk_pool_*` metrics — enable the matching collectors (see §7) for
them to fire.

To send alerts, add an Alertmanager config to `prometheus.yml`:

```yaml
alerting:
  alertmanagers:
    - static_configs:
        - targets: ['alertmanager:9093']
```

Key alerts:

| Alert | Default threshold | Severity |
|---|---|---|
| `NbuClientBackupStale` | 25 h without successful backup | warning |
| `NbuClientBackupCritical` | 48 h without successful backup | critical |
| `NbuClientNoRecentTapeCopy` | 26 h without DUPLICATION | warning |
| `NbuClientNoRecentReplication` | 28 h without IMPORT | warning |
| `NbuTapeDriveDown` | ≥1 drive DOWN for 15 min | warning |
| `NbuTapeDriveDisabled` | ≥1 drive DISABLED for 30 min | warning |
| `NbuTapePoolScratchLow` | Scratch pool < 5 volumes | warning |
| `NbuDiskPoolVolumeDegraded` | disk-pool volume not UP for 10 min | warning |
| `NbuInterSiteDivergence` | Site < 30 % of cross-site average | warning |

---

## 7. Optional collectors

### Tape metrics (NBU 10.5+)

Enabled by default in the example config above. Adds:
- `nbu_tape_drives_count` — drive status per robot
- `nbu_tape_media_count` — media inventory per pool
- `nbu_disk_pool_volume_count` — disk pool volume states

### Per-client lifecycle metrics

Enable to track per-client compliance (required for the Lifecycle and Multi-site dashboards to show per-client rows):

```yaml
collectors:
  perClient:
    enabled: true
    allowlist:         # leave empty to collect all clients
      - srv-db-01
      - srv-app-01
```

Adds:
- `nbu_client_jobs_count{site, client, action, status}` — job counts per client
- `nbu_client_last_job_success_seconds{site, client, policy, action}` — timestamp of last success

---

## 8. Dashboards

| Dashboard | What it shows |
|---|---|
| **Overview** | Health tiles per site, backup success rate, storage usage, alert summary |
| **Jobs** | Success rate, volume, duration p50/p95, dedup ratio, queued jobs |
| **Storage** | Capacity gauges per storage unit, used vs total |
| **Data Protection** | Alerts by severity, malware scans, catalog SLOs |
| **Tape & Disks** | Drive status, media inventory, pool levels, disk pool volumes |
| **Lifecycle** | Per-client compliance history (state timeline), BACKUP / DUPLICATION / IMPORT success rates, overdue clients |
| **Multi-site** | Cross-site backup volume, replication rate, divergence ratio, per-site comparison |

All dashboards have a `$site` variable (top left) to filter by site or show all at once.

---

## 9. Multi-site Prometheus setup

For two separate exporter instances (one per site), use a single Prometheus with two scrape targets:

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'netbackup'
    scrape_interval: 60s
    scrape_timeout: 30s
    static_configs:
      - targets:
          - 'nbu-exporter-paris:9440'
          - 'nbu-exporter-lyon:9440'
```

Each exporter tags its metrics with `site` from its config. Prometheus aggregates them; Grafana filters with `$site`.

---

## 10. Production hardening

```yaml
# config.yaml — recommended for production
nbuserver:
    insecureSkipVerify: false   # always validate NBU TLS certificate
```

```bash
# Change Grafana default password immediately
GF_ADMIN_PASSWORD=<strong-password> docker compose up -d
```

Consider:
- Running the stack behind a reverse proxy (nginx/Traefik) with TLS termination
- Storing the NBU API key in a secrets manager and injecting via env var
- Persisting Prometheus data with a named volume (already configured in docker-compose.yml)
- Setting `collectionInterval: "5m"` (default) and `scrapingInterval: "1h"` — don't scrape more often than NBU's job window

---

## Troubleshooting

### Exporter starts but no `nbu_*` metrics

```bash
docker compose logs nbu_exporter | grep -E "ERROR|version"
```

Common causes:
- Wrong API key → regenerate in NBU UI
- NBU API not reachable → check firewall on port 1556
- `insecureSkipVerify: false` with self-signed cert → add CA or set to `true` temporarily to diagnose

### Tape metrics missing

```bash
curl -s http://localhost:9440/metrics | grep nbu_tape
```

Empty → NBU version < 10.5, or `collectors.tape.enabled: false` in config.

### Grafana dashboards empty

1. Check Prometheus is scraping: `http://localhost:9090/targets`
2. Check the datasource in Grafana: Settings → Data Sources → Prometheus → Test
3. Query `nbu_up` in Explore — if no data, scraping isn't working

### Check-rules (optional)

```bash
# Validate alerting rules syntax (requires promtool in PATH)
make check-rules
```

---

## Related docs

- [Configuration reference](getting-started/configuration.md)
- [All metrics](metrics.md)
- [Docker usage](docker.md)
