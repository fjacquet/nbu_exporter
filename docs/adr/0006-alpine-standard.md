# Standardize container base image on Alpine, add HEALTHCHECK to the published image

## Status

Accepted (2026-08-01)

## Context

The exporter family had two published-image patterns — Alpine (5 repos, this one
included) and `gcr.io/distroless/static:nonroot` (3 repos) — as undocumented
per-repo author choice, with no written criterion. Alpine has a shell and `wget`,
so it can carry a Docker `HEALTHCHECK`; distroless cannot. nbu_exporter's local
`./Dockerfile` and compose files already got a `/livez`-based `HEALTHCHECK` /
`healthcheck:` on 2026-08-01; `Dockerfile.goreleaser` — the file that builds the
image this repo actually publishes to GHCR — did not.

## Decision

`Dockerfile.goreleaser` gains the same `HEALTHCHECK`, using `127.0.0.1` (not
`localhost` — Alpine's busybox `wget` resolves `localhost` via `::1` first, and
the exporter only binds IPv4) against `/livez`, which never depends on target
reachability or the collection cycle.

## Consequences

- Non-breaking: purely additive. No base-image or UID change (already Alpine,
  already uid 10001).
- The full family standard, including the three distroless repos' conversion, is
  recorded in `obs_exporter`'s
  `docs/superpowers/specs/2026-08-01-alpine-standard-design.md`.
