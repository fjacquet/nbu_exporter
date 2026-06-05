# Sync nbu_exporter Tooling Baseline up to pflex_exporter

**Date:** 2026-06-04
**Status:** Approved — ready for implementation planning
**Type:** Tooling / CI / packaging (no Go source changes)

## Problem

`nbu_exporter` and `pflex_exporter` are sibling Prometheus exporters maintained in
parallel. They share the same architecture and core dependency versions, but the
**engineering-hygiene layer has drifted**: `pflex_exporter` is ahead on Go version,
Makefile quality targets, CI security scanning, SBOM generation, and Docker hardening.

This conflicts with the repo owner's global mandate that every project run Semgrep
scanning in CI — `pflex` honors it, `nbu` does not.

### Verified drift (as of 2026-06-04)

| Dimension | nbu_exporter | pflex_exporter | Gap |
|---|---|---|---|
| Go version | 1.25.0 | 1.26.4 | nbu behind |
| Makefile targets | 8 (minimal) | rich: `tools`, `vet`, `lint`, `test-race`, `vuln`, `ci`, `sbom`, `release` | nbu behind |
| Semgrep in CI | ❌ | ✅ | **nbu missing** |
| govulncheck | ❌ | ✅ | **nbu missing** |
| SBOM (CycloneDX) | ❌ | ✅ + provenance | nbu missing |
| Pinned lint version | ❌ (`golangci-lint run`) | ✅ v2.12.2 via `make tools` | nbu behind |
| Dockerfile | `golang:latest`, runs as root | `golang:1.26` pinned, non-root UID 10001 | nbu less hardened |

## Goal

Close the hygiene drift. Match pflex's Go version, Makefile, CI gates, security
scanning, SBOM, and Docker hardening — **without** copying pflex's domain features
and **without** dropping nbu's GoReleaser + Homebrew release pipeline.

## Non-goals (explicitly out of scope)

- pflex's OTLP **metrics push** (`otlpmetricgrpc`) — nbu intentionally exports tracing only.
- pflex's **Kubernetes enrichment** (`internal/k8s`).
- pflex's **HTTP retry/backoff** and **multi-cluster errgroup** collection.
- A `.golangci.yml` config file — neither repo has one; parity = pinned linter binary only.
- **Container-image SBOM/provenance attestation** — see Workstream 4 rationale. nbu builds
  images *through* GoReleaser, which does not natively attest image SBOMs the way pflex's
  `docker/build-push-action` does. Out of scope unless image builds move out of GoReleaser
  (they are not — owner chose to keep GoReleaser).

## Key decisions (locked with owner)

1. **Release tooling:** keep GoReleaser + Homebrew tap (more capable than pflex's
   hand-rolled Makefile release); add SBOM to it.
2. **CI shape:** consolidate into a single `ci.yml` mirroring pflex (jobs: `quality`,
   `sbom`, `semgrep`); keep `static.yml` and `release.yml` separate.
3. **Scanners:** keep **both** CodeQL **and** add Semgrep + govulncheck.
4. **Coverage threshold:** leave values as-is. Effective gate is **70% total** via
   `.testcoverage.yml` (`vladopajic/go-test-coverage`). The `COVERAGE_THRESHOLD: 80`
   env var in the old workflows is **vestigial** (never read) — it is dropped with the
   retired workflow, not preserved as a second gate.

## Design

### Workstream 1 — Go 1.25 → 1.26

- `go.mod`: `go 1.25.0` → `go 1.26.0`.
- `Dockerfile`: builder `golang:latest` → `golang:1.26` (pinned).
- `.github/workflows/release.yml`: replace hardcoded `go-version: "1.25"` with
  `go-version-file: go.mod` so the version is single-sourced.

**Risk:** new Go toolchain may surface new `go vet` / linter findings. Verified by
`make ci` passing before completion.

### Workstream 2 — Makefile expansion

Adopt pflex's quality targets; **preserve** nbu-specific targets.

Add:
- Pinned-version block: `GOLANGCI_LINT_VERSION ?= v2.12.2`,
  `CYCLONEDX_GOMOD_VERSION ?= latest`, `GOVULNCHECK_VERSION ?= latest`.
- Targets: `tools`, `fmt-check`, `vet`, `lint`, `test-race`, `vuln`, `sbom`,
  `ci` (= `fmt-check vet lint test-race vuln`).
- `VERSION`/`LDFLAGS` with `-X main.version=$(VERSION)` injected into `cli` and
  `build-release` (matches pflex; supports the existing `--version` flag).

Update:
- `sure` → `fmt vet test` + `go build ./...` + `golangci-lint run` (adds the missing `vet`).

Preserve unchanged: `cli`, `build-release`, `docker` (image-rm logic), `run-cli`,
`run-docker`, `clean`, `test-coverage`.

Update `.PHONY` accordingly.

### Workstream 3 — CI consolidation → `ci.yml`

Create `.github/workflows/ci.yml` mirroring pflex, triggers `push: [main]` + `pull_request`:

- **`quality`** job: checkout → `setup-go` with `go-version-file: go.mod` →
  `make tools` → `make ci` → coverage gate step
  (`vladopajic/go-test-coverage@v2` reading `.testcoverage.yml`, 70%) → upload `coverage.out`.
- **`sbom`** job: `make tools` → `make sbom` → upload `dist/sbom.cdx.json`.
- **`semgrep`** job: `semgrep/semgrep` container → `semgrep scan --config auto --error --skip-unknown-extensions`.

Workflow file changes:
- **Retire** `build.yml` (its fmt/vet/build/test/coverage responsibilities move to `ci.yml`).
- **Keep CodeQL**: rename `coverage.yml` → `codeql.yml`; strip its now-redundant
  `go fmt` / `go vet` steps, leaving pure CodeQL init/autobuild/analyze.
- **Untouched**: `static.yml` (docs). `release.yml` receives only WS1 + WS4 edits.

Net workflow set after this change: `ci.yml`, `codeql.yml`, `static.yml`, `release.yml`.

### Workstream 4 — GoReleaser SBOM (+ signing)

- `.goreleaser.yml`: add an `sboms:` block invoking **cyclonedx-gomod** (the same tool
  as `make sbom`, so one SBOM toolchain) producing a CycloneDX SBOM attached to the
  release. Add a `signs:` block for **cosign keyless** signing of checksums + SBOM.
- `release.yml`: add `id-token: write` permission; install cosign + run `make tools`
  (for cyclonedx-gomod) before the GoReleaser step. `fetch-depth: 0` already present.
- Optionally align action versions toward pflex (`checkout@v6`, `setup-go@v6`) — cosmetic.

**Scope boundary:** deliverable is archive/source CycloneDX SBOM + cosign signatures.
Container-image SBOM attestation is **not** included (see Non-goals).

### Workstream 5 — Dockerfile hardening

`Dockerfile` (and `Dockerfile.goreleaser`, used by the release image build):
- Builder pinned `golang:1.26` (WS1).
- Static build flags: `CGO_ENABLED=0 -ldflags="-s -w"`.
- Non-root runtime: `adduser -D -u 10001 nbu`, `mkdir -p /var/log/nbu_exporter`,
  `chown nbu:nbu /var/log/nbu_exporter`, `USER nbu`; drop `WORKDIR /root/`.

### Workstream 6 — Verification (gate before "done")

- `make tools && make ci` → green.
- `make sbom` → emits `dist/sbom.cdx.json`.
- `docker build` succeeds; container runs as non-root (UID 10001).
- `goreleaser check` (or `release --snapshot --clean`) validates `.goreleaser.yml`.
- `semgrep scan --config auto` clean on the new CI/Docker/config files
  (per owner's global Semgrep mandate).

## Files touched

- `go.mod`
- `Makefile`
- `Dockerfile`, `Dockerfile.goreleaser`
- `.goreleaser.yml`
- `.github/workflows/`: **new** `ci.yml`; **rename** `coverage.yml` → `codeql.yml`
  (trimmed); **delete** `build.yml`; **edit** `release.yml`. `static.yml` unchanged.

No `internal/` or `main.go` source changes.

## Open follow-ups (not blocking)

- Consider a later "define a shared canonical baseline" pass where pflex adopts nbu's
  strengths (GoReleaser/Homebrew, `.testcoverage.yml` gate, committed `CLAUDE.md`).
  Tracked separately; this spec is one-directional (nbu ← pflex hygiene).
