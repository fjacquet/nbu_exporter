# Feature 3 — Per-Client "Last Successful Backup" Metric (opt-in + allowlist)

- **Date:** 2026-06-17
- **Status:** Design (approved — pending spec review)
- **Deciders:** Frederic Jacquet
- **Related:** [feature deferrals rationale](2026-06-16-nbu-feature-deferrals.md),
  [ADR-0002 (opt-in sub-collector framework)](../../adr/0002-opt-in-sub-collector-framework.md),
  [ADR-0004 (multi-site snapshot model)](../../adr/0004-multisite-snapshot-collection-model.md)

## Context & Goal

Charles asked for per-client visibility — specifically being able to alert "client X has had no
successful backup in 25h". `JobAttributes` already carries `clientName`, `endTime`, `status`,
`jobType`, `policyType`. The `client` label is high-cardinality (hundreds of clients), so the
feature is **opt-in (default off)** and bounded by an **exact-name allowlist**.

Goal: an opt-in collector exposing, per allowlisted client, the timestamp of that client's most
recent successful backup — the primitive that Feature 4's alert rule consumes — multi-site-aware
(`site` label) and safe-by-default.

## Metric

| Metric | Type | Labels (after `site`) | Meaning |
|--------|------|-----------------------|---------|
| `nbu_client_last_successful_backup_timestamp_seconds` | Gauge | `client` | Unix time (seconds) of the client's most recent successful backup |

- One series per allowlisted client. `policy_type` is intentionally **not** a label (it would be
  just the latest success's policy and would churn the series when a client's most-recent backup
  switches policy; the per-client timestamp is the alerting unit).
- Enables Feature 4: `time() - nbu_client_last_successful_backup_timestamp_seconds{client="X"} > 25*3600`
  alerts on staleness; `absent(...{client="X"})` catches a client that has *never* succeeded (or is
  unreachable).

## Collection (grounded in `docs/veritas-10.5/admin.yaml`)

For **each allowlisted client**, issue one targeted jobs query and read the single most-recent
successful backup — no lookback window is needed:

```
GET /admin/jobs?filter=clientName eq '<client>' and jobType eq 'BACKUP' and status eq 0
                &sort=-endTime&page[limit]=1
```

- `sort` supports descending via a `-` prefix (admin.yaml: "Prefix with `-` for descending order");
  `clientName`, `jobType`, `status`, `endTime` are all filter/sort attributes. `status eq 0` =
  fully successful. `sort=-endTime` + `page[limit]=1` returns the latest success regardless of age,
  so there is **no lookback gap** (a client last backed up 30 days ago still reports correctly).
- Reuse the existing `models.Jobs` response struct; read `data[0].Attributes.EndTime` and emit
  `float64(endTime.Unix())`. Empty `data` (client never succeeded / out of retention) → emit no
  series for that client. Skip a row whose `EndTime` is zero.
- The jobs endpoint is cursor-paginated, but `page[limit]=1` means a single fetch — we never follow
  the cursor (only the top row is wanted).
- **Cost:** N queries per collection cycle per site, where N = allowlist size; each returns ≤1 tiny
  row. Bounded by the (deliberately small) allowlist the operator chooses. Runs on the background
  collection cycle (default 5m), not per Prometheus scrape.

## Architecture

- A new opt-in `perClientCollector` implementing the `subCollector` interface, built per target by
  `buildSubCollectorsFor(client, cfg, site)` when `cfg.Collectors.PerClient.Enabled` — so its
  metrics buffer into the `SiteSnapshot` and carry the `site` label (ADR-0004 snapshot model).
- It iterates the allowlist, issuing the targeted query per client, and emits one timestamp gauge
  per client that has a successful backup.

## Configuration

The toggle needs more than a bool (it carries the allowlist), so a dedicated config struct:

```yaml
collectors:
  perClient:
    enabled: false
    allowlist: ["clientA.example.com", "clientB.example.com"]   # exact names
```

- **Empty/unset `allowlist` while `enabled` ⇒ emit no per-client series**, and log a one-time
  notice ("perClient enabled but allowlist empty; add client names to emit metrics"). Safe-by-default:
  enabling the flag can never accidentally explode cardinality.
- Exact-match names (no globs/regex — YAGNI).

## Error handling / degradation

- Each per-client query is independent: a failure (API error, timeout) is logged with `site` +
  `client` and skipped — that client gets no series this cycle; other clients, the core metrics,
  and `nbu_up` are unaffected. `Collect` always returns nil.

## Testing

- Fixture: a `/admin/jobs` response with one successful backup → assert
  `nbu_client_last_successful_backup_timestamp_seconds{client=…}` equals that `endTime` (Unix).
- Empty `data` fixture for a client → no series emitted for it.
- Allowlist gating: a non-allowlisted client is never queried and never emitted; an **empty**
  allowlist emits nothing.
- The mock asserts the request carries the expected `filter` (clientName/jobType/status), `sort=-endTime`,
  and `page[limit]=1`.
- `site` is the first label on the emitted series.

## Open items (verify at implementation time — do not change the metric shape)

1. Confirm the exact `jobType` filter value for backups (`BACKUP`) and that `status eq 0` is the
   right "fully successful" code (partial success = `1` is intentionally excluded).
2. Confirm OData filter quoting for `clientName eq '<name>'` after `BuildURL` URL-encoding; skip or
   sanitize allowlist entries containing a single quote (they would break the filter).

## Consequences

- **Positive:** the long-requested "last successful backup per client" signal, safe-by-default
  (opt-in + allowlist + empty=nothing), bounded cardinality, multi-site, no lookback gap; unlocks
  Feature 4's per-client alert. Reuses the validated jobs endpoint + `models.Jobs` (no new model).
- **Trade-offs:** N small queries per cycle proportional to allowlist size (acceptable: opt-in,
  small lists, 5m cycle). A windowed single-scan alternative was rejected — it needs a lookback
  (coverage gap for clients backed up just outside the window) and scans far more rows.
