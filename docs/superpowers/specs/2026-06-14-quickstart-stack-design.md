# Observability Quickstart Stack — Design

**Date:** 2026-06-14
**Status:** Approved (pending implementation plan)
**Scope:** Bring `nbu_exporter`'s docker-compose demo up to the family canonical
"observability quickstart stack" (`exporter-standards` → `references/stack.md`).

## Background

The family standard requires every exporter to ship a one-command demo stack:
exporter + Prometheus + **Grafana** with auto-provisioned datasource and dashboards,
a ghcr-image compose variant, Prometheus alerting rules, and docs. `nbu_exporter`'s
current `docker-compose.yml` runs only the exporter + Prometheus — no Grafana, no
provisioning, no ghcr variant, no `rule_files` wiring — so `docker compose up` yields
metrics with nothing to view them in.

What already exists and is reused, not rebuilt:
- `grafana/nbu-overview.json` — fully-templated overview dashboard (uses a
  `${datasource}` variable and a `$storage_unit`/`$policy_type` template variable).
- `grafana/NBU Statistics-1629904585394.json` — legacy dashboard.
- `grafana/alerts.yml` — complete Prometheus alerting rules (`NbuExporterDown`,
  `NbuApiUnreachable`, `NbuMetricsStale`, backup SLOs).
- `prometheus.yml` — scrape jobs for the exporter and otel-collector.
- Hardening conventions on existing services (`no-new-privileges`, healthchecks,
  `restart: unless-stopped`).
- GoReleaser publishes `ghcr.io/fjacquet/nbu_exporter:latest`.

## Goals

- `docker compose up` brings up exporter + Prometheus + Grafana with the dashboard(s)
  and datasource auto-provisioned and the alerting rules loaded.
- A ghcr-image variant that pulls the published image instead of building.
- Alerting rules live at the canonical `deploy/prometheus/` path and are loaded by
  Prometheus.
- Docs describe the demo and the dashboards, wired into the mkdocs nav.

## Non-goals

- No changes to dashboard JSON content (this work only provisions existing files).
- No Go code changes; no new metrics.
- Not reconciling `grafana/build_overview.py` with the hand-edited
  `nbu-overview.json` (separate follow-up).
- Not renaming `docker-compose-otel.yaml` → `.yml` (trivial, optional follow-up).

## Branch / integration

New branch `feat/quickstart-stack` off `main`, its own PR, independent of PR #27.
This work provisions `grafana/nbu-overview.json` but never edits it, so there is no
file conflict with #27. The dashboard's 11.2 panels appear once #27 merges; the stack
functions regardless.

## Architecture / files

### `docker-compose.yml` (build-based demo)
- Remove the obsolete top-level `version: "3.8"` key.
- Keep `nbu_exporter` (built from `./Dockerfile`) and `prometheus` as-is, plus:
  - Prometheus: add a read-only mount of `./deploy/prometheus/nbu.rules.yml` →
    `/etc/prometheus/rules/nbu.rules.yml`.
  - New `grafana` service:
    - `image: grafana/grafana:latest`
    - `ports: ["3000:3000"]`
    - `environment`: `GF_SECURITY_ADMIN_USER=${GF_ADMIN_USER:-admin}`,
      `GF_SECURITY_ADMIN_PASSWORD=${GF_ADMIN_PASSWORD:-admin}`
    - `volumes`:
      - `./grafana/provisioning:/etc/grafana/provisioning:ro`
      - `./grafana/nbu-overview.json:/var/lib/grafana/dashboards/nbu-overview.json:ro`
      - `./grafana/NBU Statistics-1629904585394.json:/var/lib/grafana/dashboards/nbu-statistics.json:ro`
      - `grafana-storage:/var/lib/grafana`
    - `security_opt: [no-new-privileges:true]`, `restart: unless-stopped`,
      `depends_on: [prometheus]`, `networks: [monitoring]`
    - healthcheck: `wget --spider http://localhost:3000/api/health`
- Add `grafana-storage` to the `volumes:` block.

### `docker-compose.ghcr.yml` (image-based demo)
- Same topology and provisioning as `docker-compose.yml`, except the exporter service
  uses `image: ghcr.io/fjacquet/nbu_exporter:latest` and has no `build:` key.

### `deploy/prometheus/nbu.rules.yml`
- `git mv grafana/alerts.yml deploy/prometheus/nbu.rules.yml` (content unchanged).
- Update the header comment's load-path example to the new mount path.

### `prometheus.yml`
- Add:
  ```yaml
  rule_files:
    - /etc/prometheus/rules/nbu.rules.yml
  ```
- Keep existing scrape jobs unchanged.

### Grafana provisioning (new)
- `grafana/provisioning/datasources/datasource.yml`:
  ```yaml
  apiVersion: 1
  datasources:
    - name: Prometheus
      type: prometheus
      uid: prometheus
      access: proxy
      url: http://prometheus:9090
      isDefault: true
  ```
- `grafana/provisioning/dashboards/dashboards.yml`:
  ```yaml
  apiVersion: 1
  providers:
    - name: nbu
      orgId: 1
      folder: NetBackup
      type: file
      disableDeletion: false
      updateIntervalSeconds: 30
      allowUiUpdates: true
      options:
        path: /var/lib/grafana/dashboards
        foldersFromFilesStructure: false
  ```
  The `${datasource}` template variable in the dashboards resolves to the default
  provisioned Prometheus datasource automatically.

### Docs
- `docs/dashboards.md` (new): describe the overview and legacy dashboards, the panels,
  and how they map to exporter metrics; note the provisioned datasource and folder.
- `docs/deployment/docker.md` (update): add the `docker compose up` demo walkthrough
  for both `docker-compose.yml` (build) and `docker-compose.ghcr.yml` (pull), with the
  service URLs (exporter `:2112`, Prometheus `:9090`, Grafana `:3000` admin/admin).
- `mkdocs.yml`: add `Dashboards: dashboards.md` under the Deployment nav section.

## Verification

No Go code, so no unit tests. Acceptance checks:
- `docker compose -f docker-compose.yml config -q` → exit 0 (valid compose).
- `docker compose -f docker-compose.ghcr.yml config -q` → exit 0.
- `promtool check rules deploy/prometheus/nbu.rules.yml` → SUCCESS (if `promtool`
  available; otherwise YAML-lint the file).
- The two provisioning YAMLs parse (yaml-lint / `python3 -c yaml.safe_load`).
- `mkdocs build --strict` → succeeds (nav + new page resolve).
- Manual checklist (needs a reachable NBU appliance): `docker compose up`, confirm
  Grafana at `:3000` shows the NetBackup folder with the overview dashboard populated
  from the Prometheus datasource, and Prometheus `/rules` lists the nbu rules.

## Risks

- **Datasource UID matching:** dashboards use a `${datasource}` variable, not a hard
  UID, so a single default Prometheus datasource is sufficient; verified by the manual
  checklist.
- **Legacy dashboard on `main`:** until #27 merges, the provisioned legacy dashboard is
  the un-templatized version. Acceptable — it still loads; provisioning serves whatever
  JSON is on disk.
- **`:latest` image tags** in the demo composes are intentional (consistent with the
  existing compose files and the demo's purpose); not for production use.
