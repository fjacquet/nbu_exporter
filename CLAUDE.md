# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A Prometheus exporter for Veritas NetBackup (NBU), written in Go 1.25. Exposes backup job statistics and storage metrics via HTTP for Prometheus scraping, with optional OpenTelemetry distributed tracing.

## Build & Development Commands

```bash
# Build the binary
make cli                    # Outputs to bin/nbu_exporter

# Run tests
make test                   # Run all tests
go test ./internal/exporter -run TestVersionDetection  # Run specific test
go test ./... -cover        # With coverage
go test -race ./...         # With race detection

# Code quality (format, test, build, lint)
make sure

# Run the exporter
./bin/nbu_exporter --config config.yaml
./bin/nbu_exporter -c config.yaml -d    # with debug mode

# Docker
make docker                 # Build image
make run-docker             # Run container on port 2112

# Clean
make clean
```

## Architecture

### Entry Point

- `main.go` - Cobra CLI, HTTP server with `/metrics` and `/health` endpoints, Prometheus registry, OpenTelemetry initialization
- `Server` struct manages lifecycle: HTTP server, Prometheus registry, telemetry manager

### Internal Packages (`internal/`)

**exporter/** - Core Prometheus collector

- `prometheus.go` - `NbuCollector` implementing `prometheus.Collector`. Metrics: `nbu_disk_bytes`, `nbu_jobs_bytes`, `nbu_jobs_count`, `nbu_status_count`, `nbu_api_version`
- `netbackup.go` - `FetchStorage()` and `FetchAllJobs()` for NBU API calls with pagination via `handlePagination()`
- `client.go` - Reusable HTTP client with connection pooling, TLS config, 2-minute timeout
- `version_detector.go` - Auto-detects API version (13.0 → 12.0 → 3.0 fallback)
- `tracing.go` - OpenTelemetry span creation helpers

**telemetry/** - OpenTelemetry integration

- `manager.go` - TracerProvider lifecycle, OTLP gRPC exporter, sampling configuration
- `attributes.go` - Span attribute constants

**models/** - Data structures

- `Config.go` - YAML config with `Validate()` method, `BuildURL()` helper
- `Jobs.go`, `Storage.go`, `Storages.go` - NBU API response structs

**testutil/** - Shared test helpers and constants

### Metrics Labels Pattern

Metrics use pipe-delimited keys split into labels:

- Storage: `name|type|size` (e.g., "pool1|AdvancedDisk|free")
- Jobs: `action|policy_type|status` (e.g., "BACKUP|Standard|0")

### Configuration

Requires `config.yaml` with sections: `server`, `nbuserver`, optional `opentelemetry`. API key obtained from NetBackup UI.

## Key Patterns

- **Graceful degradation**: Collector continues with partial metrics if storage or jobs fetch fails
- **Context propagation**: All API calls use context for cancellation/timeout
- **Version detection**: Auto-detects highest supported NBU API version at startup
- **Span hierarchy**: `prometheus.scrape` → `netbackup.fetch_storage` / `netbackup.fetch_jobs` → `netbackup.fetch_job_page`

## Key Dependencies

- `github.com/prometheus/client_golang` - Prometheus client
- `github.com/go-resty/resty/v2` - HTTP client
- `github.com/spf13/cobra` - CLI framework
- `github.com/sirupsen/logrus` - Logging
- `go.opentelemetry.io/otel` - OpenTelemetry tracing

<!-- rtk-instructions v2 -->
# RTK (Rust Token Killer) - Token-Optimized Commands

## Golden Rule

**Always prefix commands with `rtk`**. If RTK has a dedicated filter, it uses it. If not, it passes through unchanged. This means RTK is always safe to use.

**Important**: Even in command chains with `&&`, use `rtk`:
```bash
# ❌ Wrong
git add . && git commit -m "msg" && git push

# ✅ Correct
rtk git add . && rtk git commit -m "msg" && rtk git push
```

## RTK Commands by Workflow

### Build & Compile (80-90% savings)
```bash
rtk cargo build         # Cargo build output
rtk cargo check         # Cargo check output
rtk cargo clippy        # Clippy warnings grouped by file (80%)
rtk tsc                 # TypeScript errors grouped by file/code (83%)
rtk lint                # ESLint/Biome violations grouped (84%)
rtk prettier --check    # Files needing format only (70%)
rtk next build          # Next.js build with route metrics (87%)
```

### Test (90-99% savings)
```bash
rtk cargo test          # Cargo test failures only (90%)
rtk vitest run          # Vitest failures only (99.5%)
rtk playwright test     # Playwright failures only (94%)
rtk test <cmd>          # Generic test wrapper - failures only
```

### Git (59-80% savings)
```bash
rtk git status          # Compact status
rtk git log             # Compact log (works with all git flags)
rtk git diff            # Compact diff (80%)
rtk git show            # Compact show (80%)
rtk git add             # Ultra-compact confirmations (59%)
rtk git commit          # Ultra-compact confirmations (59%)
rtk git push            # Ultra-compact confirmations
rtk git pull            # Ultra-compact confirmations
rtk git branch          # Compact branch list
rtk git fetch           # Compact fetch
rtk git stash           # Compact stash
rtk git worktree        # Compact worktree
```

Note: Git passthrough works for ALL subcommands, even those not explicitly listed.

### GitHub (26-87% savings)
```bash
rtk gh pr view <num>    # Compact PR view (87%)
rtk gh pr checks        # Compact PR checks (79%)
rtk gh run list         # Compact workflow runs (82%)
rtk gh issue list       # Compact issue list (80%)
rtk gh api              # Compact API responses (26%)
```

### JavaScript/TypeScript Tooling (70-90% savings)
```bash
rtk pnpm list           # Compact dependency tree (70%)
rtk pnpm outdated       # Compact outdated packages (80%)
rtk pnpm install        # Compact install output (90%)
rtk npm run <script>    # Compact npm script output
rtk npx <cmd>           # Compact npx command output
rtk prisma              # Prisma without ASCII art (88%)
```

### Files & Search (60-75% savings)
```bash
rtk ls <path>           # Tree format, compact (65%)
rtk read <file>         # Code reading with filtering (60%)
rtk grep <pattern>      # Search grouped by file (75%)
rtk find <pattern>      # Find grouped by directory (70%)
```

### Analysis & Debug (70-90% savings)
```bash
rtk err <cmd>           # Filter errors only from any command
rtk log <file>          # Deduplicated logs with counts
rtk json <file>         # JSON structure without values
rtk deps                # Dependency overview
rtk env                 # Environment variables compact
rtk summary <cmd>       # Smart summary of command output
rtk diff                # Ultra-compact diffs
```

### Infrastructure (85% savings)
```bash
rtk docker ps           # Compact container list
rtk docker images       # Compact image list
rtk docker logs <c>     # Deduplicated logs
rtk kubectl get         # Compact resource list
rtk kubectl logs        # Deduplicated pod logs
```

### Network (65-70% savings)
```bash
rtk curl <url>          # Compact HTTP responses (70%)
rtk wget <url>          # Compact download output (65%)
```

### Meta Commands
```bash
rtk gain                # View token savings statistics
rtk gain --history      # View command history with savings
rtk discover            # Analyze Claude Code sessions for missed RTK usage
rtk proxy <cmd>         # Run command without filtering (for debugging)
rtk init                # Add RTK instructions to CLAUDE.md
rtk init --global       # Add RTK to ~/.claude/CLAUDE.md
```

## Token Savings Overview

| Category | Commands | Typical Savings |
|----------|----------|-----------------|
| Tests | vitest, playwright, cargo test | 90-99% |
| Build | next, tsc, lint, prettier | 70-87% |
| Git | status, log, diff, add, commit | 59-80% |
| GitHub | gh pr, gh run, gh issue | 26-87% |
| Package Managers | pnpm, npm, npx | 70-90% |
| Files | ls, read, grep, find | 60-75% |
| Infrastructure | docker, kubectl | 85% |
| Network | curl, wget | 65-70% |

Overall average: **60-90% token reduction** on common development operations.
<!-- /rtk-instructions -->