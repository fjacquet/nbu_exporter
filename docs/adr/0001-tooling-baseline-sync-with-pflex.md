# ADR-0001: Sync the tooling/CI/security baseline with pflex_exporter

- **Status:** Accepted
- **Date:** 2026-06-05
- **Deciders:** Frederic Jacquet
- **Related:** `docs/superpowers/specs/2026-06-04-sync-nbu-tooling-baseline-design.md`, `docs/superpowers/plans/2026-06-04-sync-nbu-tooling-baseline.md`, PR #19, release v2.5.0

## Context

`nbu_exporter` and `pflex_exporter` are sibling Prometheus exporters maintained in
parallel. Their application architecture and core dependency versions were already
in sync, but the **engineering-hygiene layer had drifted**: `pflex_exporter` was
ahead on Go version, Makefile quality targets, CI security scanning, SBOM
generation, and Docker hardening.

This drift also violated a standing project rule that every repository must run
Semgrep scanning in CI — `pflex` honored it, `nbu` did not.

We treat **`pflex_exporter` as the reference baseline** for the tooling layer and
bring `nbu_exporter` up to it, one-directionally, without copying pflex's
domain-specific features.

## Decision

Adopt the following baseline in `nbu_exporter`:

1. **Go 1.26** — bumped from 1.25; the `go` directive is pinned to a patch release
   (`go 1.26.4`) so CI (`go-version-file: go.mod`) installs a toolchain that
   includes stdlib security fixes.
2. **Makefile quality/security targets** — `tools` (pinned `golangci-lint v2.12.2`,
   `cyclonedx-gomod`, `govulncheck`), `fmt-check`, `vet`, `lint`, `test-race`,
   `vuln`, `sbom`, and an aggregate `ci` target. `fmt-check` excludes `vendor/`.
3. **Consolidated CI** — a single `ci.yml` with `quality` + `sbom` + `semgrep`
   jobs. `coverage.yml` was renamed to `codeql.yml` (it always ran CodeQL); the
   old `build.yml` was retired. **Both CodeQL and Semgrep** run, plus govulncheck.
   The 70 % coverage gate (`.testcoverage.yml`) is carried into `ci.yml`.
4. **Supply-chain hardening** — all GitHub Actions are **SHA-pinned** to their
   latest releases (with version comments), and the Semgrep container image is
   **digest-pinned**.
5. **Release SBOM + signing** — GoReleaser emits per-archive CycloneDX SBOMs
   (syft) and **cosign keyless** signatures (bundle format) for the checksums file.
6. **Docker hardening** — builder pinned to `golang:1.26`, static build, and the
   runtime image runs as a **non-root user (uid 10001)** in both `Dockerfile` and
   `Dockerfile.goreleaser`.

We **keep GoReleaser + the Homebrew tap** (more capable than pflex's hand-rolled
Makefile release) rather than converging on pflex's release mechanism.

## Alternatives considered

- **Replace GoReleaser with pflex's Makefile-based release** — rejected; GoReleaser
  + Homebrew tap is strictly more capable and already working.
- **Match pflex exactly (Semgrep only, drop CodeQL)** — rejected; running both
  scanners catches complementary issue classes at acceptable CI cost.
- **Make Semgrep non-blocking to avoid pre-existing findings** — rejected; it
  contradicts the project's Semgrep mandate. Pre-existing findings were instead
  triaged (false positives annotated, real hardening applied).
- **Container-image SBOM/provenance attestation (as pflex does via
  `docker/build-push-action`)** — deferred; `nbu` builds images through
  GoReleaser, which does not natively attest image SBOMs. Out of scope.

## Consequences

### Positive
- Semgrep + CodeQL + govulncheck now gate every PR; releases are signed and ship SBOMs.
- Reproducible CI via SHA-pinned actions; hardened, non-root container image.
- Stricter gates immediately surfaced and fixed real pre-existing debt: an
  un-gofmt'd test file, 16 stdlib CVEs (from pinning `go 1.26.0`), and a committed
  `.vscode/` editor config.

### Negative / trade-offs
- SHA-pinned actions require periodic maintenance (e.g. Dependabot) to stay current.
- `nbu` is now **ahead of `pflex`** on SHA-pinning and SBOM-in-GoReleaser, so a
  reverse-sync pass on `pflex` is warranted.
- `cosign-installer v4` defaults to the new bundle format; the signing config had
  to migrate from `--output-signature/--output-certificate` to `--bundle`.

### Follow-ups
- Migrate `.goreleaser.yml` off the deprecated `dockers`/`docker_manifests`
  (→ `dockers_v2`) and `brews` properties.
- Reverse-sync the SHA-pinning + release SBOM/signing improvements into `pflex_exporter`.
