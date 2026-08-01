# Alpine Standard — Dockerfile.goreleaser HEALTHCHECK Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `HEALTHCHECK` to `Dockerfile.goreleaser` — the file that builds nbu_exporter's actually-published image — matching the `/livez` check already added to the local `./Dockerfile` and both compose files on 2026-08-01.

**Architecture:** No base-image change needed here — `Dockerfile.goreleaser` is already `alpine:latest`. This is purely additive: one `HEALTHCHECK` instruction using the same verified pattern.

**Tech Stack:** Docker, Alpine (`wget`/busybox).

**Spec:** `docs/superpowers/specs/2026-08-01-alpine-standard-design.md` in `obs_exporter` (family-wide design; this repo's own copy is not required — this plan is self-contained).

## Global Constraints

- `HEALTHCHECK` targets `http://127.0.0.1:9440/livez`, never `localhost` — Alpine's busybox `wget` resolves `localhost` via `::1` first, and the exporter only binds IPv4.
- Timing: `--interval=30s --timeout=5s --start-period=10s --retries=3`.
- Verify by building and running the image, not just by reading the Dockerfile — a `localhost`-based check on this exact family passed hadolint and `docker compose config` while still failing at runtime (caught on `nbu_exporter` itself, ironically, in the local-Dockerfile round).
- No inline `nosemgrep`/`//nolint` suppressions.

## File Structure

| File | Responsibility |
| --- | --- |
| `Dockerfile.goreleaser` | Adds `HEALTHCHECK` before `USER nbu` |
| `docs/adr/000N-alpine-standard.md` | Records the family decision as it applies to this repo |
| `CHANGELOG.md` | `Added` entry (non-breaking) |

---

### Task 1: Add HEALTHCHECK to Dockerfile.goreleaser

**Files:**
- Modify: `Dockerfile.goreleaser`

**Interfaces:** none — single-file, no code.

- [ ] **Step 1: Edit the file**

Insert before `USER nbu`:

```dockerfile
EXPOSE 9440

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget --quiet --tries=1 --spider http://127.0.0.1:9440/livez || exit 1

USER nbu
```

(The `EXPOSE 9440` line already exists — only the `HEALTHCHECK` block and the blank lines around it are new.)

- [ ] **Step 2: Lint**

Run: `hadolint Dockerfile.goreleaser`
Expected: no new findings versus the current baseline (pre-existing `DL3007`/`DL3025`/`DL3066` on this file, if any, are out of scope).

- [ ] **Step 3: Build and verify at runtime**

`Dockerfile.goreleaser` expects a pre-built binary staged into the build context (normally done by GoReleaser's buildx step), so build it directly against a local binary:

```bash
CGO_ENABLED=0 go build -o nbu_exporter .
mkdir -p linux/amd64 && cp nbu_exporter linux/amd64/nbu_exporter
docker build -f Dockerfile.goreleaser --build-arg TARGETPLATFORM=linux/amd64 -t nbu_exporter:healthcheck-test .
docker run -d --name nbu-hc-test -p 19440:9440 \
  -e NBU1_HOSTNAME=master.my.domain -e NBU1_APIKEY=dummy \
  nbu_exporter:healthcheck-test
sleep 15
docker inspect --format='{{.State.Health.Status}}' nbu-hc-test
```

Expected: `healthy`.

- [ ] **Step 4: Clean up test artifacts**

```bash
docker rm -f nbu-hc-test
docker rmi nbu_exporter:healthcheck-test
rm -rf linux nbu_exporter
```

- [ ] **Step 5: Commit**

```bash
git add Dockerfile.goreleaser
git commit -m "feat(docker): add HEALTHCHECK to the published image (Dockerfile.goreleaser)"
```

---

### Task 2: ADR + CHANGELOG

**Files:**
- Create: `docs/adr/000N-alpine-standard.md` (N = next free number in this repo's `docs/adr/` sequence — check before writing)
- Modify: `CHANGELOG.md`

**Interfaces:** none.

- [ ] **Step 1: Find the next ADR number**

Run: `ls docs/adr/ | sort -V | tail -3`

- [ ] **Step 2: Write the ADR**

```markdown
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
```

- [ ] **Step 3: Add the CHANGELOG entry**

Under `## [Unreleased]` (create it above the most recent version heading if absent), `### Added`:

```markdown
- `HEALTHCHECK` added to the published Docker image, checking `/livez`. See
  ADR-000N.
```

- [ ] **Step 4: Commit**

```bash
git add docs/adr/000N-alpine-standard.md CHANGELOG.md
git commit -m "docs: record ADR-000N (HEALTHCHECK on the published image)"
```

## Self-Review

- Spec coverage: this repo's row in the family table (`Dockerfile.goreleaser` HEALTHCHECK only, no base-image change) — covered by Task 1. Documentation — covered by Task 2.
- No placeholders: ADR number and CHANGELOG heading location require a one-command check (Step 1) before writing, not a guess.
- Scope: single repo, two tasks, matches the family plan's per-repo row exactly.
