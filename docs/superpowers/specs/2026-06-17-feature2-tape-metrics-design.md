# Feature 2 — Tape / Drive / Robot Metrics (opt-in sub-collector)

- **Date:** 2026-06-17
- **Status:** Design (approved — pending spec review)
- **Deciders:** Frederic Jacquet
- **Related:** [ADR-0002 (opt-in sub-collector framework)](../../adr/0002-opt-in-sub-collector-framework.md),
  [feature deferrals rationale](2026-06-16-nbu-feature-deferrals.md),
  [ADR-0004 (multi-site snapshot model)](../../adr/0004-multisite-snapshot-collection-model.md)

## Context & Goal

Charles's request (Feature 2) asks for tape library visibility: drives, tape media (volumes),
and robots. The deferral note settled the approach: **REST, not CLI shell-out** (shelling out to
`bpmedialist`/`vmquery` would force the exporter onto a master/media server, breaking the
remote-HTTP-only model), shipped as an **opt-in sub-collector** per ADR-0002.

Goal: a single opt-in `tape` sub-collector exposing bounded-cardinality drive/media/robot health
metrics, default-off, that works across NBU versions via per-endpoint graceful degradation and
carries the `site` label automatically (multi-site, ADR-0004).

## Scope

Three `/storage/` endpoints (validated against the checked-in OpenAPI bundles):

| Endpoint | NBU availability | Notes |
|----------|------------------|-------|
| `GET /storage/drives` | **10.0+** (`docs/veritas-10.3/storage.yaml`) | present on all supported NBU |
| `GET /storage/tape-media` | **10.5+** (`docs/veritas-10.5/storage.yaml`) | absent on NBU 10.0–10.4 |
| `GET /storage/robots-device-hosts` | **10.5+** | returns a flat **list of device-host names** only |

The collector must **degrade per endpoint**: on NBU 10.0–10.4 the tape-media and robots endpoints
are absent (404/406) — those are logged and skipped while drives still report. The same applies
to any per-endpoint permission error. A failing endpoint never flips `nbu_up` to 0.

**Robot correction (validated against 10.5 + 11.0 specs):** there is **no bulk robot-listing
endpoint** — only `/storage/robots-device-hosts` (hostnames: `data[].id`, `type: deviceHost`) and
a per-robot `GET /storage/robots/{robotId}`. So a `robots_count{robot_type}` / per-robot `info`
metric is **not** buildable without N+1 per-robot calls (rejected). Robot context is instead taken
from the **`robotType` / `robotNumber` attributes already present on each drive** (and tape
volume). 11.0/API-13.0 is identical here, so this is version-robust.

Out of scope (separate features): alerting rules for tape (**Feature 4**), per-client metrics
(**Feature 3**), and any tape *write* operations (move/rescan/erase) — this collector is read-only.

## Metrics

Every series carries `site` as its first label (shared label builder; ADR-0004 invariant). The
metric-specific labels:

| Metric | Type | Labels (after `site`) | Meaning |
|--------|------|-----------------------|---------|
| `nbu_tape_drives_count` | Gauge | `state`, `drive_type`, `robot_type` | Number of drives grouped by status + type + robot type |
| `nbu_tape_media_count` | Gauge | `media_type`, `status` | Number of tape volumes grouped by media type + status |
| `nbu_tape_robot_device_hosts` | Gauge | *(none beyond `site`)* | Number of device hosts that have robots configured |
| `nbu_tape_drive_info` | Gauge (=1) | `drive_name`, `media_server`, `drive_type`, `robot_number`, `state` | One series per drive (tens) — identifies *which* drive is down |

**State vocabularies (confirmed enums in `storage.yaml`):**

- drive `state` ← `driveStatus` field (`driveStatusEnum`): `UP`, `DOWN`, `MIXED`, `DISABLED`
- drive `drive_type` ← `driveType` field (`driveTypeEnum`): `DT_HCART`, `DT_DLT`, `DT_HCART2`, … (low cardinality)
- `robot_type` ← `robotType` field **on the drive** (`robotTypeEnum`): `NA`, `NOT_ROBOTIC`, `ACS`, `TLD`
- media `media_type` ← `mediaType` field (`tapeMediaTypeEnum`): `HCART`, `DLT`, …, plus cleaning media (`HC_CLN`, `DLT_CLN`, …)
- media `status` ← **`mediaStatus`** field (free-form string, e.g. `ACTIVE MULTIPLEXED`)

**Cardinality:** `count` metrics are bounded by enum cross-products regardless of fleet size.
The per-entity info metric is emitted **only** for drives (tens), **never** for tape media (can be
thousands of volumes). Per-entity values like media barcode/serial and drive serial are kept out
of all labels. Including `state` as a `drive_info` label is the
kube-state-metrics idiom; the only cost is mild series churn when a drive's state changes, which
is infrequent and acceptable.

## Collection, pagination & degradation

- The collector implements the existing `subCollector` interface (`Name() string`,
  `Collect(ctx, ch) error`) and is built per target by `buildSubCollectorsFor(client, cfg, site)`,
  so its metrics are buffered into the `SiteSnapshot` and re-emitted per scrape (no live fetch on
  scrape).
- **Aggregate from `data[]`** (discover states from the data, like the alerts/malware collectors)
  rather than count-only filtered queries — `mediaStatus` is free-form so it cannot be enumerated
  up front, and drives must be enumerated anyway for their info metric.
  - **drives** → aggregate `driveStatus × driveType × robotType` into `nbu_tape_drives_count`, and
    emit one `nbu_tape_drive_info` per drive.
  - **tape-media** → aggregate `mediaType × mediaStatus` into `nbu_tape_media_count`.
  - **robots-device-hosts** → `nbu_tape_robot_device_hosts` = `len(data)` (count of device hosts).
- **Pagination:** drives are tens → a single page (`page[limit]`) suffices. robots-device-hosts is
  a short host list. **Tape media can be thousands → offset-paginate** with `page[offset]`/
  `page[limit]` (drives confirmed to expose `page[offset]`; tape-media returns the same
  `meta.pagination` shape as `/storage/storage-units`, so reuse the existing `/storage/` offset
  loop — worst case a non-paginating endpoint returns a single page and the loop terminates). Runs
  on the 5m collection cycle, not per scrape.
- **Graceful degradation:** each endpoint fetch is independent; an error (absent endpoint on 10.x,
  permission denied, transient failure) is logged with `site` + endpoint and skipped. Successful
  endpoints still emit. Mirrors the per-combination degradation already in the catalog collector.

## Configuration

One opt-in toggle (default disabled), consistent with the other sub-collectors:

```yaml
collectors:
  tape:
    enabled: false   # drives + tape-media + robot-device-host count
```

## Testing

- JSON fixtures per endpoint under `testdata/`: `drives` (mixed `driveStatus`/`driveType`/
  `robotType`), `tape-media` (**multi-page**, to exercise offset pagination + aggregation),
  `robots-device-hosts` (a couple of host entries).
- An "absent endpoint" / error fixture proving per-endpoint graceful degradation (drives succeed,
  tape-media 404 → only drive metrics emitted, `Collect` returns nil, `nbu_up` unaffected).
- Assertions mirror existing sub-collector tests: bounded labels, `site` first label, correct
  counts, `nbu_tape_drive_info` present per drive (drives only).

## Confirmed field names (validated against `docs/veritas-10.5/storage.yaml`)

- **drives** `data[].attributes`: `driveName`, `driveStatus`, `driveType`, `robotType`,
  `robotNumber`, `deviceHost` (used for `media_server`). GET exposes `page[offset]`/`page[limit]`.
- **tape-media** `data[].attributes`: `barcode`, `mediaType`, `mediaStatus`, `robotType`,
  `robotNumber`, `robotHost`. Response carries `meta.pagination` (same shape as storage-units).
- **robots-device-hosts** `data[]`: `{ id: <hostname>, type: "deviceHost" }` — host list only.

## Open items (verify at implementation time — do not change the shape above)

1. The exact `mediaStatus` value space in the wild (free-form string; the collector groups on
   whatever the API returns — no fixed enum assumed).
2. Confirm the tape-media offset loop terminates correctly against the live `meta.pagination`
   (the existing `/storage/storage-units` loop is the reference).

## Consequences

- **Positive:** tape-library health (drives down by state/type/robot, tape-volume status mix,
  robot device-host count) becomes observable over the remote HTTP model; bounded cardinality;
  works on 10.x (drives) through 11.x; per-site via the snapshot model; unlocks Feature 4 tape
  alerting.
- **Trade-offs:** tape-media collection pulls all volumes each cycle (mitigated by opt-in + 5m
  cycle); `drive_info{state}` churns on state change (accepted). 11.2 `storage.yaml` is not in the
  repo bundle — the endpoints are backward-compatible, but a follow-up should add the 11.2 bundle
  to confirm (mirrors the existing 11.2 `admin.yaml` gap).
