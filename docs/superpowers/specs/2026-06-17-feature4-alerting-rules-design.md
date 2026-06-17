# Feature 4 — Alerting Rules (per-client + tape, separate optional files)

- **Date:** 2026-06-17
- **Status:** Design (approved — pending spec review)
- **Deciders:** Frederic Jacquet
- **Related:** [feature deferrals rationale](2026-06-16-nbu-feature-deferrals.md),
  Feature 3 (per-client metric), Feature 2 (tape metrics), existing
  `deploy/prometheus/nbu.rules.yml`

## Context & Goal

Charles asked for alerting on the new signals — notably "alert when a client has had no
successful backup in ~25h", plus tape-library health. The deferrals spec settled the shape: ship
**environment-specific rules in separate optional files**, keeping the generic
`deploy/prometheus/nbu.rules.yml` clean. Both dependencies are now on `main`: Feature 3
(`nbu_client_last_successful_backup_timestamp_seconds`) and Feature 2 (`nbu_tape_drives_count`).

Goal: two new optional Prometheus rules files (per-client, tape) with promtool unit tests, all
multi-site-aware (`site` label), loaded only by operators who enabled the corresponding opt-in
collectors.

## Files

- `deploy/prometheus/rules-perclient.yml` *(new)* — per-client backup-staleness alerts.
- `deploy/prometheus/rules-tape.yml` *(new)* — tape drive-health alerts.
- `deploy/prometheus/rules-perclient_test.yml`, `deploy/prometheus/rules-tape_test.yml` *(new)* —
  `promtool test rules` unit tests.
- `Makefile` — add a `check-rules` target running `promtool check rules` + `promtool test rules`
  over `deploy/prometheus/`.
- `docs/metrics.md` — fix the stale "rules are in `grafana/alerts.yml`" line (that file does not
  exist; the rules live in `deploy/prometheus/`), and document loading the optional files.
- `CHANGELOG.md` — `[Unreleased]` entry.

## Alerts

Style follows the existing `nbu.rules.yml`: `groups → rules`, `severity` label (`warning` /
`critical`), templated `summary`/`description`. Thresholds and `for:` durations are sensible
defaults operators can tune.

### `rules-perclient.yml` (group `nbu_perclient`) — requires `collectors.perClient`

| Alert | Expr | for | severity |
|-------|------|-----|----------|
| `NbuClientBackupStale` | `time() - nbu_client_last_successful_backup_timestamp_seconds > 25 * 3600` | 1h | warning |
| `NbuClientBackupCritical` | `time() - nbu_client_last_successful_backup_timestamp_seconds > 48 * 3600` | 1h | critical |

- Per-series (per `site` + `client`); no aggregation. Annotations name `{{ $labels.site }}` and
  `{{ $labels.client }}` and the age via `{{ $value | humanizeDuration }}`.
- **Known limitation (documented):** these fire for a client that *had* a success and then went
  stale. A client that has **never** succeeded produces no series, so it can't be caught by a
  `time() - metric` rule (would need per-client `absent()` with hard-coded names). Operators should
  confirm the metric appears for each allowlisted client after enabling.

### `rules-tape.yml` (group `nbu_tape`) — requires `collectors.tape`

| Alert | Expr | for | severity |
|-------|------|-----|----------|
| `NbuTapeDriveDown` | `sum by (site) (nbu_tape_drives_count{state="DOWN"}) > 0` | 15m | warning |
| `NbuTapeDriveDisabled` | `sum by (site) (nbu_tape_drives_count{state="DISABLED"}) > 0` | 30m | warning |

- Aggregated `by (site)` so one alert per site (not per drive_type/robot_type). Annotation:
  `{{ $value }} drive(s) DOWN on site {{ $labels.site }}`.

## Loading

Each file's header comment shows the `rule_files:` snippet, e.g.:

```yaml
rule_files:
  - /etc/prometheus/rules/nbu.rules.yml
  - /etc/prometheus/rules/rules-perclient.yml   # only if collectors.perClient is enabled
  - /etc/prometheus/rules/rules-tape.yml         # only if collectors.tape is enabled
```

They are **optional** because their metrics are opt-in. A rules file referencing a metric that is
never produced simply never fires — harmless — but they are kept separate so the generic rules
stay meaningful for every deployment.

## Testing

- `promtool check rules` on all three rules files (syntax/lint) — also covers the existing
  `nbu.rules.yml`.
- `promtool test rules` unit tests proving alert logic:
  - `NbuClientBackupStale`: a client whose last-success timestamp is ~26h old fires it; ~24h old
    does **not**. `NbuClientBackupCritical`: fires past 48h, quiet at 30h.
  - `NbuTapeDriveDown`: a `nbu_tape_drives_count{state="DOWN"}` series of 1 fires it; 0 (or only
    `state="UP"`) does not. Same shape for `NbuTapeDriveDisabled`.
- `make check-rules` runs both over `deploy/prometheus/` (promtool 3.12 confirmed available
  locally). Wiring promtool into CI is noted as an optional follow-up (CI has no promtool today,
  and the existing rules file is likewise validated only manually).

## Out of scope (noted)

- Updating the generic `nbu.rules.yml` annotations to include `{{ $labels.site }}` — a valid
  multi-site improvement, but an independent change per the deferrals spec; left for a follow-up.
- Free-form `mediaStatus` regex alerts (e.g. frozen media) — fragile against a free-form string;
  excluded deliberately. Tape alerts use the clean `driveStatus` enum only.
- Adding promtool to the CI workflow — optional follow-up.

## Consequences

- **Positive:** completes Charles's request (the "no backup in 25h" alert and tape drive-down
  alerts now exist), multi-site-aware, with executable unit tests for the alert logic; the generic
  rules stay clean; optional files load only where the metrics exist.
- **Trade-offs:** the "never succeeded" per-client case isn't generically alertable (documented);
  rules are not yet CI-gated (matches the existing file; `make check-rules` provides local
  validation).
