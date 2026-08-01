# ADR-0005: `/livez` `/readyz`, and `/health` always answering 200 with cached state

## Status

Accepted (2026-08-01)

## Context

Same argument as obs_exporter's ADR-0013 and ADR-0014, applied here in one
pass: an exporter is a probe. "Site unreachable" is data it reports, not a
failure of the exporter process. Coupling that fact to an HTTP status code
on any endpoint — the chart's `livenessProbe`/`readinessProbe`, or the
informational `/health` — risks something downstream (kubelet, a dashboard,
a script) treating a healthy, correctly-reporting exporter as down.

This repo had a second problem on top of that: `healthHandler` called
`s.collector.TestConnectivity(ctx)` live, on every single `/health` request,
with its own 5-second timeout — a network round-trip to NetBackup on a path
meant to be a cheap, instant status check. A status endpoint that can block
for up to 5 seconds, or that adds load to the monitored system on every
poll, defeats the purpose of a status endpoint.

`charts/nbu-exporter/values.yaml` wired both `livenessProbe` and
`readinessProbe` to `/health`. As a *liveness* check this was always wrong:
no restart makes an unreachable NetBackup API reachable, and a slow or
hanging `TestConnectivity` call could itself trigger a liveness failure
unrelated to the exporter's own health.

## Decision

Two new endpoints, `/livez` and `/readyz`, both `staticOKHandler` — always
`200 OK`, no state read, nothing that can make either fail once the process
is running. The chart's default probes now point at them.

`/health` no longer calls `TestConnectivity`. It reads the same
`SnapshotStore` the background collection loop publishes to (the same store
`/metrics` reads) and always answers `200 OK` with a JSON body:
`{"sites": [{"site": "...", "ok": true, "last_scrape": "...", "err": ""}]}`.
`ok` comes from `SiteSnapshot.Up`; `last_scrape` is the later of
`LastStorageScrape`/`LastJobsScrape`; `err` joins `StorageErr`/`JobsErr` when
either is set.

## Consequences

- **Breaking**: `/health`'s response body changes from plain text
  (`"OK"`/`"OK (starting)"`/`"UNHEALTHY: ..."`) to JSON, and its status code
  is always 200 (previously 503 when the API was unreachable). Anything
  parsing the old text format or gating on the old status code needs
  updating.
- `/health` no longer triggers a live NetBackup API call — it costs nothing
  beyond a map read, and it no longer adds request load to the monitored
  system just from being polled.
- Chart default probe wiring changes; a fresh `helm install` or an upgrade
  without pinned probe overrides gets the fix automatically.
- Alert on a per-site `_up` metric (or `/health`'s body), never on any
  probe's HTTP status.
