# Sync nbu_exporter Tooling Baseline up to pflex_exporter — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bring nbu_exporter's Go version, Makefile, CI, security scanning, SBOM, and Docker hardening up to pflex_exporter's level — without touching Go source or the GoReleaser/Homebrew release.

**Architecture:** Pure tooling/CI/packaging change. Bump Go 1.25→1.26, expand the Makefile with pflex's quality targets, consolidate CI into one `ci.yml` (quality + sbom + semgrep) while keeping CodeQL, add an SBOM + cosign signing to GoReleaser, and harden both Dockerfiles to run non-root.

**Tech Stack:** Go 1.26, GNU Make, GitHub Actions, golangci-lint v2.12.2, govulncheck, cyclonedx-gomod, syft, cosign, Semgrep, GoReleaser v2, Docker.

**Spec:** `docs/superpowers/specs/2026-06-04-sync-nbu-tooling-baseline-design.md`

---

## Conventions for this repo

- This repo prefixes git with `rtk` (e.g. `rtk git add ...`, `rtk git commit ...`). Plain `git` in code blocks below works too; use `rtk` to match project convention.
- `vendor/` is gitignored. After any dependency change, run `go mod vendor` **or** build with `-mod=mod`.
- End every commit message with the footer:
  `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`

## Deviations from spec (read before starting)

1. **No `-X main.version` ldflag.** `main.go` has no `version` package var (verified 2026-06-04). Adding the ldflag would target a nonexistent symbol and emit a linker warning. The spec mandates no Go source changes, so the Makefile keeps `LDFLAGS = -s -w` only. The Homebrew formula's `--version` test is a pre-existing latent issue — **out of scope** for this plan.
2. **GoReleaser SBOM uses `syft` (not `cyclonedx-gomod`).** The spec preferred a single SBOM tool, but GoReleaser's per-artifact SBOM model integrates cleanly with syft (its documented default), whereas cyclonedx-gomod is awkward to wire in. We configure syft to emit **CycloneDX JSON**, so the output *format* still matches `make sbom`. The CI `sbom` job keeps cyclonedx-gomod. This is a conscious format-consistent compromise; see Task 5 note.

## Checkpoints requiring owner input (do not silently expand scope)

- **Task 2 / `make lint`:** golangci-lint v2.12.2 is newer/pinned and may surface pre-existing findings. If `make lint` fails on existing code, STOP and report — fixing lint debt is a scope decision (spec forbids adding `.golangci.yml`).
- **Task 2 / `make vuln`:** govulncheck may report a vulnerable dependency requiring a bump. If it fails, STOP and report.

---

## Task 1: Bump Go to 1.26

**Files:**
- Modify: `go.mod:3`

- [ ] **Step 1: Edit the Go directive**

In `go.mod`, change the version line:

```
go 1.26.0
```

(from `go 1.25.0`)

- [ ] **Step 2: Tidy and re-vendor**

Run:

```bash
go mod tidy
go mod vendor
```

Expected: no errors; `go.mod`/`go.sum`/`vendor/` updated consistently.

- [ ] **Step 3: Verify build and tests pass on Go 1.26**

Run:

```bash
go build ./...
go test ./...
```

Expected: build succeeds; all tests PASS. (If your local Go is <1.26, the toolchain directive will auto-download 1.26 — allow it.)

- [ ] **Step 4: Commit**

```bash
git add go.mod go.sum
git commit -m "build: bump Go to 1.26

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 2: Expand the Makefile with quality/security targets

**Files:**
- Modify: `Makefile` (full replacement)

- [ ] **Step 1: Replace the Makefile**

Replace the entire contents of `Makefile` with:

```makefile
# Define the output binary
CLI_BIN = nbu_exporter
DIST    = dist
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS = -s -w

# Pinned tool versions (installed by `make tools`).
GOLANGCI_LINT_VERSION   ?= v2.12.2
CYCLONEDX_GOMOD_VERSION ?= latest
GOVULNCHECK_VERSION     ?= latest

# Default target: build, test, docker
all: cli test docker

# Install pinned dev/CI tooling into $(GOBIN)/$GOPATH/bin.
tools:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	go install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@$(CYCLONEDX_GOMOD_VERSION)
	go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)

# --- quality gates (used by CI) ---

fmt-check:
	@test -z "$$(gofmt -l .)" || (echo "gofmt needed in:"; gofmt -l .; exit 1)

fmt:
	go fmt ./...

vet:
	go vet ./...

lint:
	golangci-lint run ./...

test:
	go test ./...

test-race:
	go test -race -coverprofile=coverage.out -covermode=atomic ./...

vuln:
	govulncheck ./...

# Aggregate gate run by CI.
ci: fmt-check vet lint test-race vuln

# Local convenience: format, vet, test, build, lint.
sure: fmt vet test
	go build ./...
	golangci-lint run

# --- artifacts ---

# Build the CLI binary
cli:
	go build -ldflags="$(LDFLAGS)" -o bin/$(CLI_BIN) .

# Build with stripped symbols (alias kept for compatibility)
build-release:
	go build -ldflags="$(LDFLAGS)" -o bin/$(CLI_BIN) .

# CycloneDX SBOM for the Go module (source/dependency SBOM).
sbom:
	@mkdir -p $(DIST)
	cyclonedx-gomod mod -licenses -json -output $(DIST)/sbom.cdx.json
	@echo "wrote $(DIST)/sbom.cdx.json"

# Tests with HTML coverage report
test-coverage: test-race
	go tool cover -html=coverage.out -o coverage.html

# Build the Docker image
docker:
	@if [ -n "$(shell docker images -q $(CLI_BIN) 2> /dev/null)" ]; then \
		docker image rm -f $(CLI_BIN); \
	fi
	docker build -t $(CLI_BIN) .

# Run the CLI binary
run-cli: cli
	./bin/$(CLI_BIN) --config config.yaml

# Run the Docker container
run-docker: docker
	docker run -d -p 2112:2112 --name $(CLI_BIN) $(CLI_BIN)

# Clean up build artifacts
clean:
	rm -f bin/$(CLI_BIN) coverage.out coverage.html
	rm -rf $(DIST)
	@if [ -n "$(shell docker images -q $(CLI_BIN) 2> /dev/null)" ]; then \
		docker image rm -f $(CLI_BIN); \
	fi

.PHONY: all tools fmt-check fmt vet lint test test-race vuln ci sure \
        cli build-release sbom test-coverage docker run-cli run-docker clean
```

- [ ] **Step 2: Install the pinned tooling**

Run:

```bash
make tools
```

Expected: golangci-lint v2.12.2, cyclonedx-gomod, and govulncheck install without error. Ensure `$(go env GOPATH)/bin` is on your `PATH`.

- [ ] **Step 3: Verify the formatting/vet/lint gates**

Run:

```bash
make fmt-check
make vet
make lint
```

Expected: all pass. **CHECKPOINT:** if `make lint` reports pre-existing findings, STOP and report to owner (scope decision).

- [ ] **Step 4: Verify the test-race + vuln gates**

Run:

```bash
make test-race
make vuln
```

Expected: tests PASS with race detector; `coverage.out` produced; govulncheck reports no vulnerabilities. **CHECKPOINT:** if `make vuln` fails, STOP and report.

- [ ] **Step 5: Verify the SBOM target**

Run:

```bash
make sbom
test -s dist/sbom.cdx.json && echo "SBOM OK"
```

Expected: prints `wrote dist/sbom.cdx.json` then `SBOM OK`.

- [ ] **Step 6: Verify the aggregate gate**

Run:

```bash
make ci
```

Expected: runs fmt-check, vet, lint, test-race, vuln — all PASS.

- [ ] **Step 7: Commit**

```bash
git add Makefile
git commit -m "build: add quality/security Make targets (tools, lint, test-race, vuln, sbom, ci)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 3: Harden both Dockerfiles (pin builder, non-root runtime)

**Files:**
- Modify: `Dockerfile` (full replacement)
- Modify: `Dockerfile.goreleaser` (full replacement)

- [ ] **Step 1: Replace `Dockerfile`**

Replace the entire contents of `Dockerfile` with:

```dockerfile
# Stage 1: Build
FROM golang:1.26 AS builder

WORKDIR /app

# Copy and download dependencies using go mod
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Static build
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o nbu_exporter .

# Stage 2: Runtime
FROM alpine:latest

# ca-certificates for HTTPS, plus a non-root user and writable log dir
RUN apk --no-cache add ca-certificates && \
    adduser -D -u 10001 nbu && \
    mkdir -p /var/log/nbu_exporter && \
    chown nbu:nbu /var/log/nbu_exporter

# Copy the binary and default config
COPY --from=builder /app/nbu_exporter /usr/bin/nbu_exporter
COPY config.yaml /etc/nbu_exporter/config.yaml

# Expose the default port (configurable via config.yaml)
EXPOSE 2112

USER nbu

ENTRYPOINT ["/usr/bin/nbu_exporter"]
CMD ["--config", "/etc/nbu_exporter/config.yaml"]
```

- [ ] **Step 2: Replace `Dockerfile.goreleaser`**

Replace the entire contents of `Dockerfile.goreleaser` with:

```dockerfile
# Dockerfile for GoReleaser
# Expects the binary to be pre-built by GoReleaser.
FROM alpine:latest

# ca-certificates for HTTPS, plus a non-root user and writable log dir
RUN apk --no-cache add ca-certificates && \
    adduser -D -u 10001 nbu && \
    mkdir -p /var/log/nbu_exporter && \
    chown nbu:nbu /var/log/nbu_exporter

# Copy the pre-built binary from GoReleaser and the default config
COPY nbu_exporter /usr/bin/nbu_exporter
COPY config.yaml /etc/nbu_exporter/config.yaml

EXPOSE 2112

USER nbu

ENTRYPOINT ["/usr/bin/nbu_exporter"]
CMD ["--config", "/etc/nbu_exporter/config.yaml"]
```

- [ ] **Step 3: Build the image and verify it runs as non-root**

Run:

```bash
docker build -t nbu_exporter:harden-test .
docker run --rm --entrypoint id nbu_exporter:harden-test
```

Expected: build succeeds; `id` prints `uid=10001(nbu) gid=10001(nbu) ...` (not uid=0/root).

- [ ] **Step 4: Clean up the test image**

Run:

```bash
docker image rm -f nbu_exporter:harden-test
```

- [ ] **Step 5: Commit**

```bash
git add Dockerfile Dockerfile.goreleaser
git commit -m "build(docker): pin builder to golang:1.26 and run as non-root (uid 10001)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 4: Consolidate CI workflows

**Files:**
- Create: `.github/workflows/ci.yml`
- Rename + trim: `.github/workflows/coverage.yml` → `.github/workflows/codeql.yml`
- Delete: `.github/workflows/build.yml`
- Unchanged: `.github/workflows/static.yml`

- [ ] **Step 1: Create `.github/workflows/ci.yml`**

Create `.github/workflows/ci.yml` with:

```yaml
name: CI

on:
  push:
    branches: [main]
  pull_request:

permissions:
  contents: read

jobs:
  quality:
    name: Lint, vet, test, vuln
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
      - uses: actions/setup-go@v6
        with:
          go-version-file: go.mod
          cache: true
      - name: Install tooling
        run: make tools
      - name: Run CI gate (fmt, vet, lint, test -race, govulncheck)
        run: make ci
      - name: Check coverage threshold
        uses: vladopajic/go-test-coverage@v2
        with:
          config: ./.testcoverage.yml
      - name: Upload coverage
        if: always()
        uses: actions/upload-artifact@v7
        with:
          name: coverage
          path: coverage.out
          if-no-files-found: ignore

  sbom:
    name: Generate SBOM
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
      - uses: actions/setup-go@v6
        with:
          go-version-file: go.mod
          cache: true
      - name: Install tooling
        run: make tools
      - name: Generate CycloneDX SBOM
        run: make sbom
      - name: Upload SBOM
        uses: actions/upload-artifact@v7
        with:
          name: sbom
          path: dist/sbom.cdx.json

  semgrep:
    name: Semgrep
    runs-on: ubuntu-latest
    container:
      image: semgrep/semgrep
    steps:
      - uses: actions/checkout@v6
      - name: Semgrep scan
        run: semgrep scan --config auto --error --skip-unknown-extensions
```

- [ ] **Step 2: Recreate the CodeQL workflow under its real name**

Create `.github/workflows/codeql.yml` with (CodeQL-only; the fmt/vet steps that used to live in `coverage.yml` are now covered by `ci.yml`):

```yaml
name: CodeQL

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]
  schedule:
    - cron: "0 0 * * 1" # Run every Monday at midnight

jobs:
  analyze:
    name: CodeQL Analysis
    runs-on: ubuntu-latest
    permissions:
      actions: read
      contents: read
      security-events: write

    strategy:
      fail-fast: false
      matrix:
        language: ["go"]

    steps:
      - name: Checkout code
        uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
          cache: true

      - name: Initialize CodeQL
        uses: github/codeql-action/init@v4
        with:
          languages: ${{ matrix.language }}

      - name: Autobuild
        uses: github/codeql-action/autobuild@v4

      - name: Perform CodeQL Analysis
        uses: github/codeql-action/analyze@v4
```

- [ ] **Step 3: Remove the old `coverage.yml` and `build.yml`**

Run:

```bash
git rm .github/workflows/coverage.yml .github/workflows/build.yml
```

Expected: both files staged for deletion. (`coverage.yml` is superseded by `codeql.yml`; `build.yml` is superseded by `ci.yml`.)

- [ ] **Step 4: Validate YAML syntax of the new workflows**

Run:

```bash
python3 -c "import yaml,sys; [yaml.safe_load(open(f)) for f in ['.github/workflows/ci.yml','.github/workflows/codeql.yml']]; print('YAML OK')"
```

Expected: prints `YAML OK`.

- [ ] **Step 5: Run Semgrep locally on the changed files (owner's global mandate)**

Run:

```bash
semgrep scan --config auto --error .github/workflows/ Dockerfile Dockerfile.goreleaser Makefile
```

Expected: completes with no blocking (`--error`) findings. If Semgrep is not installed locally, run via Docker: `docker run --rm -v "$PWD:/src" semgrep/semgrep semgrep scan --config auto --error /src/.github /src/Dockerfile /src/Dockerfile.goreleaser /src/Makefile`.

- [ ] **Step 6: Commit**

```bash
git add .github/workflows/ci.yml .github/workflows/codeql.yml
git commit -m "ci: consolidate into ci.yml (quality+sbom+semgrep), rename coverage.yml to codeql.yml, drop build.yml

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 5: Add SBOM + signing to GoReleaser and update the release workflow

**Files:**
- Modify: `.goreleaser.yml` (add `sboms:` and `signs:` blocks)
- Modify: `.github/workflows/release.yml` (single-source Go version, add cosign/syft, id-token permission)

> **Note (deviation):** GoReleaser's SBOM is generated with **syft** configured to output **CycloneDX JSON** (format-consistent with `make sbom`). See "Deviations from spec" at the top. The `signs:` block uses cosign **keyless** signing of the checksums file and requires `id-token: write` in the workflow.

- [ ] **Step 1: Add the `sboms:` and `signs:` blocks to `.goreleaser.yml`**

Append these two top-level blocks to `.goreleaser.yml` (after the `archives:` block; order among top-level keys does not matter):

```yaml
sboms:
  - id: archive
    artifacts: archive
    cmd: syft
    args:
      - "$artifact"
      - "--output"
      - "cyclonedx-json=$document"
    documents:
      - "{{ .ArtifactName }}.sbom.cdx.json"

signs:
  - cmd: cosign
    artifacts: checksum
    output: true
    certificate: "${artifact}.pem"
    args:
      - "sign-blob"
      - "--output-certificate=${certificate}"
      - "--output-signature=${signature}"
      - "${artifact}"
      - "--yes"
```

- [ ] **Step 2: Update `.github/workflows/release.yml`**

Replace the entire contents of `.github/workflows/release.yml` with:

```yaml
# .github/workflows/release.yml

name: 🚀 Release Go Exporter

on:
  push:
    tags:
      - "v*.*.*"

permissions:
  contents: write # create the GitHub Release
  packages: write # push the image to GHCR
  id-token: write # cosign keyless signing / provenance

jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - name: ⬇️ Checkout code
        uses: actions/checkout@v4
        with:
          fetch-depth: 0 # needed for changelog generation

      - name: 🏗️ Setup Go
        uses: actions/setup-go@v5
        with:
          go-version-file: go.mod

      - name: 🧰 Install SBOM tooling (syft)
        uses: anchore/sbom-action/download-syft@v0

      - name: 🔏 Install cosign
        uses: sigstore/cosign-installer@v3

      - name: 🐋 Set up QEMU
        uses: docker/setup-qemu-action@v3

      - name: 🛠️ Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: 🔑 Log in to GitHub Container Registry (GHCR)
        uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: 🚀 Run GoReleaser
        uses: goreleaser/goreleaser-action@v6
        with:
          version: "~> v2"
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          TAP_GITHUB_TOKEN: ${{ secrets.TAP_GITHUB_TOKEN }}
```

- [ ] **Step 3: Validate the GoReleaser config schema**

Run:

```bash
go install github.com/goreleaser/goreleaser/v2@latest
goreleaser check
```

Expected: `1 configuration file(s) validated` / no schema errors.

- [ ] **Step 4: Dry-run the SBOM-producing pipeline (no publish/sign/docker)**

Run:

```bash
goreleaser release --snapshot --clean --skip=publish,sign,docker
ls dist/*.sbom.cdx.json
```

Expected: snapshot build completes; at least one `dist/*.sbom.cdx.json` is produced by syft. (We skip `sign` and `docker` because cosign keyless needs CI OIDC and image push needs registry auth — those run only in the tagged CI release.)

- [ ] **Step 5: Validate the release workflow YAML**

Run:

```bash
python3 -c "import yaml; yaml.safe_load(open('.github/workflows/release.yml')); print('YAML OK')"
```

Expected: prints `YAML OK`.

- [ ] **Step 6: Commit**

```bash
git add .goreleaser.yml .github/workflows/release.yml
git commit -m "release: emit CycloneDX SBOM (syft) and cosign-sign checksums; single-source Go version

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 6: Update docs currency and run the final verification sweep

**Files:**
- Modify: `CLAUDE.md` (Go version references)

- [ ] **Step 1: Update the Go version in `CLAUDE.md` Project Overview**

In `CLAUDE.md`, change the Project Overview line:

```
A Prometheus exporter for Veritas NetBackup (NBU), written in Go 1.26. Exposes backup job statistics and storage metrics via HTTP for Prometheus scraping, with optional OpenTelemetry distributed tracing.
```

(from `written in Go 1.25`)

- [ ] **Step 2: Update the Prerequisites in `CLAUDE.md`**

In `CLAUDE.md` Prerequisites, change:

```
- Go 1.26+
```

(from `- Go 1.25+`)

- [ ] **Step 3: Final full verification sweep**

Run:

```bash
make ci && make sbom && go build ./...
```

Expected: the aggregate gate passes, SBOM is written, and the binary builds.

- [ ] **Step 4: Confirm the final workflow set**

Run:

```bash
ls -1 .github/workflows/
```

Expected exactly: `ci.yml`, `codeql.yml`, `release.yml`, `static.yml` (no `build.yml`, no `coverage.yml`).

- [ ] **Step 5: Commit**

```bash
git add CLAUDE.md
git commit -m "docs: bump documented Go version to 1.26

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Definition of Done

- `go.mod` is on Go 1.26; `go build ./...` and `go test ./...` pass.
- `make ci` (fmt-check, vet, lint, test-race, vuln) passes with pinned golangci-lint v2.12.2.
- `make sbom` produces `dist/sbom.cdx.json`.
- `.github/workflows/` contains exactly `ci.yml`, `codeql.yml`, `release.yml`, `static.yml`.
- Both Dockerfiles pin `golang:1.26` and run as non-root uid 10001 (`docker run --entrypoint id` confirms).
- `goreleaser check` passes; `goreleaser release --snapshot --clean --skip=publish,sign,docker` emits a CycloneDX SBOM.
- Semgrep scan of the changed CI/Docker/Make files reports no blocking findings.
- `CLAUDE.md` reflects Go 1.26.
- Out of scope and untouched: all `internal/` and `main.go` source, `static.yml`, OTLP metrics push, k8s enrichment, HTTP retry, multi-cluster, container-image SBOM attestation.
