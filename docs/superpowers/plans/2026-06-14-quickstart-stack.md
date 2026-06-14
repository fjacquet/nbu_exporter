# Observability Quickstart Stack — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `docker compose up` bring up exporter + Prometheus + Grafana with auto-provisioned datasource, dashboards, and alerting rules — the family canonical quickstart stack.

**Architecture:** Reuse the existing dashboards (`grafana/nbu-overview.json`, legacy `NBU Statistics-*.json`) and alerting rules (`grafana/alerts.yml`); add Grafana provisioning, a Grafana service to the compose file, a ghcr-image compose variant, relocate the rules to `deploy/prometheus/`, wire `rule_files`, and document it.

**Tech Stack:** Docker Compose, Prometheus, Grafana (file-based provisioning), MkDocs.

**Spec:** `docs/superpowers/specs/2026-06-14-quickstart-stack-design.md`

**Conventions for every task:**
- This is configuration/docs, not Go — verification is `docker compose config -q`, `promtool check rules`, YAML parse, and `mkdocs build --strict`.
- Keep the existing hardening on every service (`security_opt: [no-new-privileges:true]`, healthcheck, `restart: unless-stopped`).
- Commit messages use Conventional Commits + trailer `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`.
- Branch: `feat/quickstart-stack` (already created off `main`).

---

## File Structure

| File | Responsibility | Action |
|---|---|---|
| `deploy/prometheus/nbu.rules.yml` | Prometheus alerting rules (moved from `grafana/alerts.yml`) | Create (git mv) |
| `prometheus.yml` | Add `rule_files` wiring | Modify |
| `grafana/provisioning/datasources/datasource.yml` | Auto-provision Prometheus datasource | Create |
| `grafana/provisioning/dashboards/dashboards.yml` | Auto-load dashboards from a folder | Create |
| `docker-compose.yml` | exporter + Prometheus + Grafana demo (build-based) | Modify |
| `docker-compose.ghcr.yml` | same stack, pulls ghcr image | Create |
| `docs/dashboards.md` | Document the dashboards | Create |
| `docs/deployment/docker.md` | Add compose demo walkthrough | Modify |
| `mkdocs.yml` | Add Dashboards to nav | Modify |

---

## Task 1: Relocate alerting rules and wire Prometheus

**Files:**
- Create (move): `deploy/prometheus/nbu.rules.yml` (from `grafana/alerts.yml`)
- Modify: `prometheus.yml`

- [ ] **Step 1: Move the rules file with git**

```bash
mkdir -p deploy/prometheus
git mv grafana/alerts.yml deploy/prometheus/nbu.rules.yml
```

- [ ] **Step 2: Update the header comment in the moved file**

In `deploy/prometheus/nbu.rules.yml`, the top comment shows a load path. Update the example to the new mount path. Replace the comment block lines that read:

```
#   rule_files:
#     - /etc/prometheus/rules/nbu_alerts.yml
#
# Validate:  promtool check rules grafana/alerts.yml
```

with:

```
#   rule_files:
#     - /etc/prometheus/rules/nbu.rules.yml
#
# Validate:  promtool check rules deploy/prometheus/nbu.rules.yml
```

- [ ] **Step 3: Add `rule_files` to `prometheus.yml`**

Insert this block immediately after the `global:` section (before `scrape_configs:`):

```yaml
rule_files:
  - /etc/prometheus/rules/nbu.rules.yml
```

- [ ] **Step 4: Validate the rules**

Run: `promtool check rules deploy/prometheus/nbu.rules.yml`
Expected: `SUCCESS` (N rules found).
If `promtool` is not installed, instead run:
`python3 -c "import yaml,sys; yaml.safe_load(open('deploy/prometheus/nbu.rules.yml')); print('YAML OK')"`
Expected: `YAML OK`.

- [ ] **Step 5: Validate prometheus.yml parses**

Run: `python3 -c "import yaml; yaml.safe_load(open('prometheus.yml')); print('OK')"`
Expected: `OK`.

- [ ] **Step 6: Commit**

```bash
git add deploy/prometheus/nbu.rules.yml prometheus.yml
git commit -m "feat(deploy): relocate alerting rules to deploy/prometheus and wire rule_files"
```

---

## Task 2: Grafana provisioning files

**Files:**
- Create: `grafana/provisioning/datasources/datasource.yml`
- Create: `grafana/provisioning/dashboards/dashboards.yml`

- [ ] **Step 1: Create the datasource provisioning file**

`grafana/provisioning/datasources/datasource.yml`:

```yaml
apiVersion: 1

datasources:
  - name: Prometheus
    type: prometheus
    uid: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true
    editable: true
```

- [ ] **Step 2: Create the dashboard provider file**

`grafana/provisioning/dashboards/dashboards.yml`:

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

- [ ] **Step 3: Validate both YAML files parse**

Run:
```bash
python3 -c "import yaml; yaml.safe_load(open('grafana/provisioning/datasources/datasource.yml')); yaml.safe_load(open('grafana/provisioning/dashboards/dashboards.yml')); print('OK')"
```
Expected: `OK`.

- [ ] **Step 4: Commit**

```bash
git add grafana/provisioning/datasources/datasource.yml grafana/provisioning/dashboards/dashboards.yml
git commit -m "feat(grafana): add datasource and dashboard provisioning"
```

---

## Task 3: Add Grafana to `docker-compose.yml`

**Files:**
- Modify: `docker-compose.yml`

The current file runs `nbu_exporter` + `prometheus` with `version: "3.8"`. Make the three edits below.

- [ ] **Step 1: Remove the obsolete `version` key**

Delete the first line `version: "3.8"` and the blank line after it. The file should start directly with `services:`.

- [ ] **Step 2: Mount the rules into the existing `prometheus` service**

In the `prometheus` service's `volumes:` list (it currently mounts `./prometheus.yml` and a `prometheus-data` volume), add the rules mount so the list reads:

```yaml
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml:ro
      - ./deploy/prometheus/nbu.rules.yml:/etc/prometheus/rules/nbu.rules.yml:ro
      - prometheus-data:/prometheus
```

(If the current `docker-compose.yml` has no `prometheus` service — verify first — STOP and report: the spec assumed Prometheus is already present. The repo's `docker-compose.yml` currently has only `nbu_exporter` + Prometheus is in `docker-compose-otel.yaml`. In that case, ADD a `prometheus` service mirroring the one in `docker-compose-otel.yaml`, with the two read-only mounts above plus `prometheus-data:/prometheus`, `ports: ["9090:9090"]`, the standard `command:` flags, `security_opt`, healthcheck, `restart: unless-stopped`, `depends_on: [nbu_exporter]`, `networks: [monitoring]`.)

- [ ] **Step 3: Add the `grafana` service**

Add this service after `prometheus` (before the top-level `networks:` block):

```yaml
  grafana: # nosemgrep
    image: grafana/grafana:latest
    container_name: grafana
    security_opt:
      - no-new-privileges:true
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_USER=${GF_ADMIN_USER:-admin}
      - GF_SECURITY_ADMIN_PASSWORD=${GF_ADMIN_PASSWORD:-admin}
      - GF_USERS_ALLOW_SIGN_UP=false
    volumes:
      - ./grafana/provisioning:/etc/grafana/provisioning:ro
      - ./grafana/nbu-overview.json:/var/lib/grafana/dashboards/nbu-overview.json:ro
      - ./grafana/NBU Statistics-1629904585394.json:/var/lib/grafana/dashboards/nbu-statistics.json:ro
      - grafana-storage:/var/lib/grafana
    restart: unless-stopped
    depends_on:
      - prometheus
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:3000/api/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 20s
    networks:
      - monitoring
```

- [ ] **Step 4: Add the `grafana-storage` volume**

Ensure a top-level `volumes:` block exists with both volumes:

```yaml
volumes:
  prometheus-data:
  grafana-storage:
```

(If the file's bottom currently has only `networks:` and no `volumes:` block, add the `volumes:` block.)

- [ ] **Step 5: Validate the compose file**

Run: `docker compose -f docker-compose.yml config -q && echo "compose OK"`
Expected: `compose OK` (no errors). If `docker compose` is unavailable, run
`python3 -c "import yaml; yaml.safe_load(open('docker-compose.yml')); print('YAML OK')"` → `YAML OK`.

- [ ] **Step 6: Commit**

```bash
git add docker-compose.yml
git commit -m "feat(compose): add Grafana with provisioning to the demo stack"
```

---

## Task 4: Create `docker-compose.ghcr.yml`

**Files:**
- Create: `docker-compose.ghcr.yml`

- [ ] **Step 1: Create the ghcr variant**

Create `docker-compose.ghcr.yml` — identical to the final `docker-compose.yml` from Task 3, except the exporter service uses the published image and has no `build:` key. Write the complete file:

```yaml
services:
  nbu_exporter: # nosemgrep
    image: ghcr.io/fjacquet/nbu_exporter:latest
    container_name: nbu_exporter
    security_opt:
      - no-new-privileges:true
    ports:
      - "2112:2112"
    volumes:
      - ./config.yaml:/etc/nbu_exporter/config.yaml:ro
      - ./log:/var/log/nbu_exporter
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:2112/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
    environment:
      - NBU1_HOSTNAME=${NBU1_HOSTNAME:-master.my.domain}
      - NBU1_APIKEY=${NBU1_APIKEY:-}
    networks:
      - monitoring

  prometheus: # nosemgrep
    image: prom/prometheus:latest
    container_name: prometheus
    security_opt:
      - no-new-privileges:true
    command:
      - "--config.file=/etc/prometheus/prometheus.yml"
      - "--storage.tsdb.path=/prometheus"
      - "--web.console.libraries=/usr/share/prometheus/console_libraries"
      - "--web.console.templates=/usr/share/prometheus/consoles"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml:ro
      - ./deploy/prometheus/nbu.rules.yml:/etc/prometheus/rules/nbu.rules.yml:ro
      - prometheus-data:/prometheus
    ports:
      - "9090:9090"
    restart: unless-stopped
    depends_on:
      - nbu_exporter
    networks:
      - monitoring

  grafana: # nosemgrep
    image: grafana/grafana:latest
    container_name: grafana
    security_opt:
      - no-new-privileges:true
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_USER=${GF_ADMIN_USER:-admin}
      - GF_SECURITY_ADMIN_PASSWORD=${GF_ADMIN_PASSWORD:-admin}
      - GF_USERS_ALLOW_SIGN_UP=false
    volumes:
      - ./grafana/provisioning:/etc/grafana/provisioning:ro
      - ./grafana/nbu-overview.json:/var/lib/grafana/dashboards/nbu-overview.json:ro
      - ./grafana/NBU Statistics-1629904585394.json:/var/lib/grafana/dashboards/nbu-statistics.json:ro
      - grafana-storage:/var/lib/grafana
    restart: unless-stopped
    depends_on:
      - prometheus
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:3000/api/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 20s
    networks:
      - monitoring

networks:
  monitoring:
    driver: bridge

volumes:
  prometheus-data:
  grafana-storage:
```

NOTE: keep this file's `prometheus`/`grafana` services byte-identical to the ones produced in Task 3 (same mounts, hardening, healthchecks). If Task 3's final services differ from the above (e.g. different `command:` flags), reconcile so the only difference between the two compose files is the exporter's `image:` vs `build:`.

- [ ] **Step 2: Validate**

Run: `docker compose -f docker-compose.ghcr.yml config -q && echo "compose OK"`
Expected: `compose OK`. Fallback if no docker: `python3 -c "import yaml; yaml.safe_load(open('docker-compose.ghcr.yml')); print('YAML OK')"`.

- [ ] **Step 3: Commit**

```bash
git add docker-compose.ghcr.yml
git commit -m "feat(compose): add ghcr-image quickstart variant"
```

---

## Task 5: Documentation

**Files:**
- Create: `docs/dashboards.md`
- Modify: `docs/deployment/docker.md`
- Modify: `mkdocs.yml`

- [ ] **Step 1: Create `docs/dashboards.md`**

Write a page covering: the auto-provisioned Prometheus datasource and the "NetBackup" Grafana folder; the **NetBackup Overview** dashboard (`nbu-overview.json`) — its `${datasource}`, `$storage_unit`, `$policy_type` template variables and the panel groups (health/availability, storage capacity, jobs, and the 11.2 opt-in collector panels for alerts/malware/catalog/SLO); and the **legacy NBU Statistics** dashboard. State that dashboards are loaded from `grafana/*.json` via file provisioning and that editing the JSON and restarting Grafana updates them. Keep headings consistent with other files in `docs/` (sentence-case `#`/`##`, lines ≤120 chars per `.markdownlint.json`).

- [ ] **Step 2: Add a compose-demo walkthrough to `docs/deployment/docker.md`**

Read the existing `docs/deployment/docker.md` first and append a "## Quickstart demo stack (Docker Compose)" section that documents:
- `docker compose up -d` (build variant) vs `docker compose -f docker-compose.ghcr.yml up -d` (pull the published image).
- Service URLs: exporter metrics `http://localhost:2112/metrics`, Prometheus `http://localhost:9090`, Grafana `http://localhost:3000` (default login `admin` / `admin`, overridable via `GF_ADMIN_USER`/`GF_ADMIN_PASSWORD`).
- That the Grafana datasource and the NetBackup dashboards are auto-provisioned, and Prometheus loads alerting rules from `deploy/prometheus/nbu.rules.yml`.
- That `config.yaml` is the source of truth; set `NBU1_HOSTNAME`/`NBU1_APIKEY` (e.g. in a gitignored `.env`) for the single-target quickstart.
- `docker compose down -v` to tear down.

- [ ] **Step 3: Add Dashboards to the mkdocs nav**

In `mkdocs.yml`, under the `Deployment:` nav section (which currently lists Verification, Quick Reference, Docker), add a Dashboards entry:

```yaml
      - Dashboards: dashboards.md
```

- [ ] **Step 4: Validate the docs build**

Run: `mkdocs build --strict 2>&1 | tail -5`
Expected: build succeeds with no warnings about missing pages/nav. If `mkdocs` is not installed, instead confirm the new file is referenced and parses:
`python3 -c "import yaml; yaml.safe_load(open('mkdocs.yml')); print('mkdocs.yml OK')"` and `test -f docs/dashboards.md && echo "page OK"`.

- [ ] **Step 5: Commit**

```bash
git add docs/dashboards.md docs/deployment/docker.md mkdocs.yml
git commit -m "docs: document the quickstart demo stack and dashboards"
```

---

## Task 6: Final verification gate

**Files:** none (verification)

- [ ] **Step 1: Validate both compose files**

Run:
```bash
docker compose -f docker-compose.yml config -q && echo "main OK"
docker compose -f docker-compose.ghcr.yml config -q && echo "ghcr OK"
```
Expected: `main OK` and `ghcr OK`. (Fallback to YAML parse if docker unavailable, per earlier tasks.)

- [ ] **Step 2: Confirm the only diff between the two composes is the exporter image/build**

Run: `diff <(grep -v 'image: ghcr' docker-compose.ghcr.yml) docker-compose.yml | head -40`
Review: the differences should be limited to the exporter's `build: .` / `image:` line and any ordering. If `prometheus`/`grafana` services differ in substance, reconcile.

- [ ] **Step 3: Validate the rules once more**

Run: `promtool check rules deploy/prometheus/nbu.rules.yml` (or the YAML fallback).
Expected: SUCCESS.

- [ ] **Step 4: Confirm no stray `grafana/alerts.yml` remains and the move is clean**

Run: `test ! -f grafana/alerts.yml && test -f deploy/prometheus/nbu.rules.yml && echo "move OK"`
Expected: `move OK`.

- [ ] **Step 5: Final commit if any fixups were needed**

```bash
git add -A
git commit -m "chore(compose): reconcile demo stack verification fixups"
```

---

## Self-Review notes (for the executor)

- The repo's current `docker-compose.yml` may contain only the exporter + Prometheus (Prometheus is definitely present in `docker-compose-otel.yaml`). Verify what's in `docker-compose.yml` BEFORE Task 3; if Prometheus is missing there, add it per Task 3 Step 2's parenthetical, mirroring `docker-compose-otel.yaml`.
- Do NOT edit any dashboard JSON (`grafana/nbu-overview.json`, `NBU Statistics-*.json`) — this work only provisions them.
- The legacy dashboard volume key uses the file's real name with a space: `./grafana/NBU Statistics-1629904585394.json`. Keep it quoted/exact.
- `# nosemgrep` on compose services matches the existing repo convention for the compose files (already present on the current services) — keep it consistent; this is compose YAML, not Go (the no-suppression rule in the stack standard targets Go source).
- Leave `.serena/project.yml` and untracked `docs/veritas-11.2/` unstaged throughout.
