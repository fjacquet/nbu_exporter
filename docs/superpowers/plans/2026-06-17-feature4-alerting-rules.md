# Feature 4 — Alerting Rules (per-client + tape) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax. **One task per subagent (small), gate each on its promtool run, commit per task.** This feature is YAML rules + promtool unit tests + a Makefile target + docs (no Go code). TDD here = write the promtool unit test, watch it fail (rule absent), write the rule, watch it pass.

**Goal:** Two new optional Prometheus rules files — per-client backup-staleness and tape drive-health alerts — with promtool unit tests, completing Charles's request.

**Architecture:** Standalone rules files under `deploy/prometheus/` (generic `nbu.rules.yml` untouched), each with a sibling `*_test.yml` exercised by `promtool test rules`; a `make check-rules` target runs `promtool check rules` + `promtool test rules` over them. Annotations use only `$labels` (deterministic) so the unit tests assert exact rendered strings.

**Tech Stack:** Prometheus alerting rules (YAML), `promtool` (3.12 available locally). No Go. RTK: optional. Commit trailer: `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`.

**Spec:** `docs/superpowers/specs/2026-06-17-feature4-alerting-rules-design.md`

## Notes grounding the plan

- Existing generic rules: `deploy/prometheus/nbu.rules.yml` (group `nbu_exporter`, style: `alert`/`expr`/`for`/`labels.severity`/`annotations.summary,description`). Left unchanged.
- Metrics consumed: `nbu_client_last_successful_backup_timestamp_seconds{site,client}` (Feature 3, on main) and `nbu_tape_drives_count{site,state,drive_type,robot_type}` (Feature 2, on main).
- `promtool` resolves a test file's `rule_files` relative to the test file's own directory, so `rule_files: [rules-perclient.yml]` works when the test sits beside the rule in `deploy/prometheus/`.
- **Deviation from spec (intentional):** annotations are static (`$labels` only), dropping the `humanizeDuration`/`$value` "age" text — promtool requires `exp_annotations` to match exactly and the humanized format is brittle to hardcode. Operators can add `$value` back if they want it.

## File Structure

- `deploy/prometheus/rules-perclient.yml` *(new)* — group `nbu_perclient`.
- `deploy/prometheus/rules-perclient_test.yml` *(new)* — promtool unit test.
- `deploy/prometheus/rules-tape.yml` *(new)* — group `nbu_tape`.
- `deploy/prometheus/rules-tape_test.yml` *(new)* — promtool unit test.
- `Makefile` — add `check-rules` target + add it to `.PHONY`.
- `docs/metrics.md` — fix the stale `grafana/alerts.yml` reference + document the optional rules files.
- `CHANGELOG.md` — `[Unreleased]` entry.

---

## Task 1: Per-client staleness rules + unit test

**Files:** Create `deploy/prometheus/rules-perclient_test.yml`, `deploy/prometheus/rules-perclient.yml`.

- [ ] **Step 1: Write the failing test** — create `deploy/prometheus/rules-perclient_test.yml`:

```yaml
# promtool test rules deploy/prometheus/rules-perclient_test.yml
rule_files:
  - rules-perclient.yml
tests:
  - interval: 1h
    input_series:
      # Last successful backup at t=0; series present (value 0) through 60h.
      - series: 'nbu_client_last_successful_backup_timestamp_seconds{site="paris",client="clientA"}'
        values: '0x60'
    alert_rule_test:
      # 24h old: below the 25h warning threshold -> nothing.
      - eval_time: 24h
        alertname: NbuClientBackupStale
        exp_alerts: []
      # 27h old (true since 25h, > for:1h): warning fires.
      - eval_time: 27h
        alertname: NbuClientBackupStale
        exp_alerts:
          - exp_labels:
              severity: warning
              site: paris
              client: clientA
            exp_annotations:
              summary: "Client clientA has no successful backup in over 25h (site paris)"
              description: "Client clientA on site paris has not completed a successful backup in over 25 hours."
      # 27h old: still below 48h -> critical quiet.
      - eval_time: 27h
        alertname: NbuClientBackupCritical
        exp_alerts: []
      # 50h old (true since 48h, > for:1h): critical fires.
      - eval_time: 50h
        alertname: NbuClientBackupCritical
        exp_alerts:
          - exp_labels:
              severity: critical
              site: paris
              client: clientA
            exp_annotations:
              summary: "Client clientA has no successful backup in over 48h (site paris)"
              description: "Client clientA on site paris has not completed a successful backup in over 48 hours."
```

- [ ] **Step 2: Run — expect FAIL** (rule file absent)

Run: `promtool test rules deploy/prometheus/rules-perclient_test.yml`
Expected: error like `error parsing rule_files: ... rules-perclient.yml: no such file or directory`.

- [ ] **Step 3: Implement** — create `deploy/prometheus/rules-perclient.yml`:

```yaml
# Optional Prometheus alerting rules — per-client backup staleness.
#
# Requires the opt-in per-client collector (collectors.perClient), which produces
# nbu_client_last_successful_backup_timestamp_seconds. Load alongside nbu.rules.yml:
#
#   rule_files:
#     - /etc/prometheus/rules/nbu.rules.yml
#     - /etc/prometheus/rules/rules-perclient.yml
#
# Validate:  make check-rules
#
# Note: these fire for a client that HAD a success and went stale. A client that has
# never succeeded produces no series and cannot be caught by a time()-based rule;
# confirm the metric appears for each allowlisted client after enabling the collector.
groups:
  - name: nbu_perclient
    rules:
      - alert: NbuClientBackupStale
        expr: time() - nbu_client_last_successful_backup_timestamp_seconds > 25 * 3600
        for: 1h
        labels:
          severity: warning
        annotations:
          summary: "Client {{ $labels.client }} has no successful backup in over 25h (site {{ $labels.site }})"
          description: "Client {{ $labels.client }} on site {{ $labels.site }} has not completed a successful backup in over 25 hours."
      - alert: NbuClientBackupCritical
        expr: time() - nbu_client_last_successful_backup_timestamp_seconds > 48 * 3600
        for: 1h
        labels:
          severity: critical
        annotations:
          summary: "Client {{ $labels.client }} has no successful backup in over 48h (site {{ $labels.site }})"
          description: "Client {{ $labels.client }} on site {{ $labels.site }} has not completed a successful backup in over 48 hours."
```

- [ ] **Step 4: Run — expect PASS**

Run: `promtool test rules deploy/prometheus/rules-perclient_test.yml`
Expected: `SUCCESS`. Also `promtool check rules deploy/prometheus/rules-perclient.yml` → `SUCCESS`.

- [ ] **Step 5: Commit**

```bash
git add deploy/prometheus/rules-perclient.yml deploy/prometheus/rules-perclient_test.yml
git commit -m "feat(alerts): per-client backup-staleness rules (warning 25h / critical 48h)"
```

---

## Task 2: Tape drive-health rules + unit test

**Files:** Create `deploy/prometheus/rules-tape_test.yml`, `deploy/prometheus/rules-tape.yml`.

- [ ] **Step 1: Write the failing test** — create `deploy/prometheus/rules-tape_test.yml`:

```yaml
# promtool test rules deploy/prometheus/rules-tape_test.yml
rule_files:
  - rules-tape.yml
tests:
  - interval: 1m
    input_series:
      - series: 'nbu_tape_drives_count{site="paris",state="DOWN",drive_type="DT_HCART",robot_type="TLD"}'
        values: '1x60'
      - series: 'nbu_tape_drives_count{site="paris",state="UP",drive_type="DT_HCART",robot_type="TLD"}'
        values: '3x60'
    alert_rule_test:
      # One DOWN drive, true > for:15m -> warning fires (sum by site drops drive_type/robot_type).
      - eval_time: 20m
        alertname: NbuTapeDriveDown
        exp_alerts:
          - exp_labels:
              severity: warning
              site: paris
            exp_annotations:
              summary: "Tape drive(s) DOWN on site paris"
              description: "One or more tape drives are DOWN on site paris."
      # No DISABLED series -> no alert.
      - eval_time: 40m
        alertname: NbuTapeDriveDisabled
        exp_alerts: []
```

- [ ] **Step 2: Run — expect FAIL** (rule file absent)

Run: `promtool test rules deploy/prometheus/rules-tape_test.yml`
Expected: error `... rules-tape.yml: no such file or directory`.

- [ ] **Step 3: Implement** — create `deploy/prometheus/rules-tape.yml`:

```yaml
# Optional Prometheus alerting rules — tape drive health.
#
# Requires the opt-in tape collector (collectors.tape), which produces
# nbu_tape_drives_count. Load alongside nbu.rules.yml:
#
#   rule_files:
#     - /etc/prometheus/rules/nbu.rules.yml
#     - /etc/prometheus/rules/rules-tape.yml
#
# Validate:  make check-rules
groups:
  - name: nbu_tape
    rules:
      - alert: NbuTapeDriveDown
        expr: sum by (site) (nbu_tape_drives_count{state="DOWN"}) > 0
        for: 15m
        labels:
          severity: warning
        annotations:
          summary: "Tape drive(s) DOWN on site {{ $labels.site }}"
          description: "One or more tape drives are DOWN on site {{ $labels.site }}."
      - alert: NbuTapeDriveDisabled
        expr: sum by (site) (nbu_tape_drives_count{state="DISABLED"}) > 0
        for: 30m
        labels:
          severity: warning
        annotations:
          summary: "Tape drive(s) DISABLED on site {{ $labels.site }}"
          description: "One or more tape drives are DISABLED on site {{ $labels.site }}."
```

- [ ] **Step 4: Run — expect PASS**

Run: `promtool test rules deploy/prometheus/rules-tape_test.yml` → `SUCCESS`; `promtool check rules deploy/prometheus/rules-tape.yml` → `SUCCESS`.

- [ ] **Step 5: Commit**

```bash
git add deploy/prometheus/rules-tape.yml deploy/prometheus/rules-tape_test.yml
git commit -m "feat(alerts): tape drive DOWN/DISABLED rules (per site)"
```

---

## Task 3: `make check-rules` target

**Files:** Modify `Makefile`.

- [ ] **Step 1: Implement** — add this target to `Makefile` (place it after the `vuln:` target):

```makefile
check-rules:
	promtool check rules deploy/prometheus/nbu.rules.yml deploy/prometheus/rules-perclient.yml deploy/prometheus/rules-tape.yml
	promtool test rules deploy/prometheus/rules-perclient_test.yml deploy/prometheus/rules-tape_test.yml
```

Then add `check-rules` to the `.PHONY:` line (append it to the existing list).

- [ ] **Step 2: Run — expect PASS**

Run: `make check-rules`
Expected: `promtool check rules` prints `SUCCESS` for all three rule files; `promtool test rules` prints `SUCCESS` for both test files.

- [ ] **Step 3: Commit**

```bash
git add Makefile
git commit -m "build: add 'make check-rules' (promtool check + unit-test the alert rules)"
```

---

## Task 4: Docs + CHANGELOG

**Files:** Modify `docs/metrics.md`, `CHANGELOG.md`.

- [ ] **Step 1:** In `docs/metrics.md`, replace the stale line (currently `Alerting rules are provided in `grafana/alerts.yml` (load via `rule_files`).`) with:

```markdown
Alerting rules live in `deploy/prometheus/`: the generic `nbu.rules.yml` (availability,
staleness, backup-success-rate, storage) plus two **optional** files loaded only if you
enabled the matching opt-in collector — `rules-perclient.yml` (per-client backup staleness;
needs `collectors.perClient`) and `rules-tape.yml` (tape drive DOWN/DISABLED; needs
`collectors.tape`). Load via `rule_files` and validate with `make check-rules`.
```

- [ ] **Step 2:** In `CHANGELOG.md` `[Unreleased]` → Added:

```markdown
- **Alerting rules** (`deploy/prometheus/`): two optional, site-aware Prometheus rule files —
  `rules-perclient.yml` (`NbuClientBackupStale` >25h warning, `NbuClientBackupCritical` >48h
  critical) and `rules-tape.yml` (`NbuTapeDriveDown` / `NbuTapeDriveDisabled` per site) — each
  with promtool unit tests, plus a `make check-rules` target. The generic `nbu.rules.yml` is
  unchanged.
```

- [ ] **Step 3:** Run `make check-rules` once more (sanity) → `SUCCESS`.

- [ ] **Step 4: Commit**

```bash
git add docs/metrics.md CHANGELOG.md
git commit -m "docs: document optional per-client + tape alerting rules"
```

---

## Final

- [ ] Whole-implementation review (spec coverage: per-client stale/critical ✓, tape down/disabled ✓, site-aware ✓, promtool unit tests ✓, make check-rules ✓, generic file untouched ✓, metrics.md fixed ✓, never-succeeded limitation documented in the rules file ✓).
- [ ] `make check-rules` green; `make ci` green (no Go changed, but confirm nothing broke).
- [ ] superpowers:finishing-a-development-branch.

## Acceptance criteria (from spec)

- Two optional rules files under `deploy/prometheus/`; generic `nbu.rules.yml` unchanged.
- `NbuClientBackupStale` (warning, >25h) + `NbuClientBackupCritical` (critical, >48h) on
  `nbu_client_last_successful_backup_timestamp_seconds`; `NbuTapeDriveDown` + `NbuTapeDriveDisabled`
  (warning, `sum by (site)`) on `nbu_tape_drives_count`. All carry `site` (and `client`).
- promtool unit tests prove each fires past its threshold and stays quiet below it; `make check-rules`
  validates + tests all rules.
