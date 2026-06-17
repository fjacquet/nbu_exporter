# NBU 10.x Support & Multi-Site Roadmap — Design

**Date:** 2026-06-16
**Status:** Approved (pending implementation plan)
**Scope:** Settle the direction on a five-feature request from an external operator
(Charles Bijon, NBU 10.3 / API v3.0, two sites), and specify the first PR:
restore API-version auto-detection (Feature 5).

## Background

An external operator running **NetBackup 10.3.0.1 (API v3.0 only)** across **two sites
(one master per site)** sent a feature request covering five items plus a batch of
"already done" v3.0 validation fixes held in their fork. The request asks the maintainer
for direction before they open PRs.

The feature premises were audited against the codebase and are accurate:

- Config exposes a **single** `nbuserver:` block (`internal/models/Config.go`), not an array.
- `SetDefaults()` forces `APIVersion = "14.0"` when omitted, and detection only runs when
  the version is empty — so omitting `apiVersion` silently disables the documented
  auto-detect and hard-fails on NBU < 11.2.
- `ClientName` already exists in `JobAttributes` but is not emitted as a label.
- Tape storage units are excluded from metrics today.

Two corrections to their email: the cited `acceleratorOptimization` bug (issue #17) is
**already fixed** (`Jobs.go` is `float64` with a regression test), and their stated
baseline ("post v3.0.0 Helm chart release") does not match this repo — there is **no Helm
chart** and the latest tag is **v2.9.0**. Confirm their actual tree before promising PR
compatibility.

## Direction decided

The per-feature decisions and the rationale for deferring everything except Feature 5 are
recorded separately in
[`2026-06-16-nbu-feature-deferrals.md`](2026-06-16-nbu-feature-deferrals.md). Summary:

| # | Feature | Decision |
|---|---|---|
| done | v3.0 validation fixes | PR welcome; its version guard depends on Feature 5 (must key off the *detected* version). |
| 5 | Auto-detect fix | **This PR.** |
| 1 | Multi-master / multi-site | Accept **option (b)**: `nbuservers:[]` array, one process, family **snapshot model**, required `site` identity label (1:1 with master). Keystone; deferred. |
| 2 | Tape / robot metrics | REST under `/storage/*` (not `/media/*`); opt-in sub-collector; confirm v3.0 support on appliance. Deferred. |
| 3 | Per-client metrics | Opt-in + allowlist for cardinality; long lookback. Deferred. |
| 4 | Alerting rules | Separate optional rules file; coupled to F2/F3. Deferred. |

## Goals (this PR — Feature 5)

- Restore the documented behavior: **omitting `apiVersion` triggers auto-detection**
  (`14.0 → 13.0 → 12.0 → 3.0`) instead of hard-defaulting to `14.0`.
- Make a default (version-omitted) config work out-of-the-box against **NBU 10.x**.
- Leave the v3.0 graceful-degradation work to the operator's own PR — Feature 5 is the
  prerequisite that makes their version guard meaningful.

## Non-goals (this PR)

- No multi-master / `nbuservers:[]` changes, no snapshot-model adoption (Feature 1).
- No version-guarding of the opt-in sub-collectors, no `dedup_ratio` / queued-reason
  suppression (the operator's v3.0 PR owns these).
- No new metrics, no metric-name or label changes.

## Design — Feature 5

### Root cause

The auto-detect machinery is **fully built and never reached**:

- `validateAPIVersion()` already returns `nil` for an empty version
  (`internal/models/Config.go`).
- `shouldPerformVersionDetection()` already returns `true` for an empty version, and
  `performVersionDetectionIfNeeded()` runs the detector and writes the detected version
  back to config (`internal/exporter/client.go`).

The single line defeating both is the default in `SetDefaults()`:

```go
// internal/models/Config.go
if c.NbuServer.APIVersion == "" {
    c.NbuServer.APIVersion = APIVersion140 // "14.0"
}
```

Because `Validate()` calls `SetDefaults()` first, the version is never empty by the time
detection is consulted, so detection never runs.

### The fix (delete, don't add)

1. **Remove** the `APIVersion` default block from `SetDefaults()`. An empty version then
   flows correctly: passes validation → triggers detection → detector negotiates the
   highest supported version and writes it back.
2. **Update** the `SetDefaults()` doc comment, which currently states it defaults the API
   version to 14.0.

No other change is required:

- `buildSubCollectors()` does not read `APIVersion` today.
- Detection runs inside `NewNbuCollector` / `NewNbuClientWithVersionDetection` **before**
  any code that needs the version, and the detector probes explicit versions itself, so a
  transiently-empty version on the base client during detection is safe.

### Files touched

- `internal/models/Config.go` — remove default block; fix doc comment.
- `internal/models/Config_test.go` (and any test asserting `SetDefaults`→`14.0`) — update
  expectations; add a regression test that an empty version stays empty after
  `SetDefaults()`/`Validate()` and that `shouldPerformVersionDetection` reports `true`.
- `README.md` / config examples / `CLAUDE.md` — confirm "auto-detect if omitted" is now
  literally true and note the fail-fast consequence below.

### Consequences (honest)

- **No regression for 11.2 users.** Detection tries `14.0` first, so an omitted version on
  an 11.2 appliance still resolves to `14.0`.
- **Behavior change:** omitting `apiVersion` against an **unreachable** appliance now
  **fails fast at startup** (the detection probe errors) instead of starting and failing
  per-scrape. This is preferable but is a change; document it. Operators who want
  start-anyway behavior set `apiVersion` explicitly.
- The detection path issues a few extra startup requests when the version is omitted; this
  is the intended, documented cost of auto-detect.

## Testing

- Unit: `SetDefaults()` leaves an empty `APIVersion` empty; sets URI/CacheTTL defaults as
  before.
- Unit: `shouldPerformVersionDetection` is `true` for empty, `false` for a set version.
- Regression: a config with no `apiVersion` validates successfully (no "unsupported
  version" error) and is routed to detection.
- Existing version-detection integration tests continue to pass for the
  `14.0 → 13.0 → 12.0 → 3.0` ladder.
- `make ci` (fmt/vet/lint/test-race/govulncheck + 70% coverage) green.

## Acceptance criteria

- A `config.yaml` with `apiVersion` omitted, pointed at an NBU 10.x appliance, auto-detects
  `3.0` and collects storage/jobs metrics — no manual `apiVersion: "3.0"` workaround.
- The same config against an 11.2 appliance auto-detects `14.0`.
- No change to emitted metric names or labels.

## References

- Deferral / direction rationale: `docs/superpowers/specs/2026-06-16-nbu-feature-deferrals.md`
- Family standard: `exporter-standards` skill — `architecture.md` (snapshot model, "one
  process, many backends" identity label), `decisions.md`.
- Prior art: ADR-0002 (opt-in sub-collector framework), spec
  `2026-06-14-nbu-11.2-validation-design.md`.
