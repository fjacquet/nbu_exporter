# Dashboards

The quickstart demo stack ships with Grafana dashboards and a Prometheus datasource that are
provisioned automatically — no manual import or datasource setup is required.

## Auto-provisioned datasource

The Prometheus datasource is provisioned from
`grafana/provisioning/datasources/datasource.yml`. It is named **Prometheus**, has the UID
`prometheus`, points at `http://prometheus:9090` (the in-stack Prometheus service), and is marked as
the default datasource. Panels reference it through the `${datasource}` template variable rather than
hard-coding a UID, so the dashboards keep working if you re-point them at another Prometheus.

## NetBackup folder

Dashboards are loaded from `grafana/*.json` via file-based provisioning, configured in
`grafana/provisioning/dashboards/dashboards.yml`. The provider drops every dashboard into a Grafana
folder named **NetBackup** so the demo dashboards stay grouped together. The compose files mount the
JSON read-only into `/var/lib/grafana/dashboards`; Grafana re-scans that folder every 30 seconds, so
editing a JSON file and restarting (or just saving — the provider polls) updates the dashboard in
place. Edits made in the Grafana UI are allowed (`allowUiUpdates: true`) but are not written back to
the JSON files.

## NetBackup Overview dashboard

The main dashboard is `grafana/nbu-overview.json` ("NetBackup — Vue d'ensemble / Overview"). Panel
titles are bilingual (French / English).

### Template variables

- `${datasource}` — selects the Prometheus datasource (defaults to the provisioned **Prometheus**).
- `$storage_unit` — filters the storage panels to one or more storage units.
- `$policy_type` — filters the jobs panels to one or more policy types.

### Panel groups

The dashboard is organised into collapsible rows:

- **Health / availability** — availability (`up`), detected API version, scrape staleness, and API
  latency.
- **Storage capacity** — percent used, used-capacity trend, a per-unit storage table, and max
  concurrent jobs.
- **Jobs** — backup success rate, job-state breakdown, jobs by policy, backup volume, and queued
  jobs.
- **Durations** — job duration p50/p95, files processed, and mean deduplication ratio.

### NetBackup 11.2 opt-in collector panels

When the optional NetBackup 11.2 collectors are enabled in `config.yaml`, the overview surfaces the
extra metrics they emit through dedicated panels for **alerts**, **malware** scan results, **catalog**
health, and **SLO** compliance. These panels stay empty (no data) until the corresponding 11.2
collectors are turned on and the appliance returns the data, so they are safe to leave in place on
older NetBackup versions.

## Legacy NBU Statistics dashboard

`grafana/NBU Statistics-1629904585394.json` ("NBU Statistics") is the original community dashboard,
mounted into the stack as `nbu-statistics.json`. It is kept for continuity with existing setups;
new deployments should prefer the **NetBackup Overview** dashboard, which uses the templated
datasource and the extended metric set.
