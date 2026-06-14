# ADR-0002: Pluggable opt-in sub-collector framework

- **Status:** Accepted
- **Date:** 2026-06-14
- **Deciders:** Frederic Jacquet
- **Related:** `docs/superpowers/specs/2026-06-14-nbu-11.2-validation-design.md`, `docs/superpowers/plans/2026-06-14-nbu-11.2-validation.md`

## Context

Until now the exporter collected exactly two metric areas — storage and jobs —
hardcoded directly in `NbuCollector.Collect` and fetched concurrently via an
`errgroup`. Validating the implementation against the NetBackup 11.2 API
(`version=14.0`) surfaced several additional, valuable signals the exporter did
not expose: alerts (`/manage/alerts`), malware scan results
(`/malware/latest-scan-results`), catalog posture (`/catalog/images`), and SLOs
(`/servicecatalog/slos`).

Adding these inline would have:

- grown `prometheus.go` past the point where it can be reasoned about as one unit;
- made each new area untestable in isolation;
- forced every deployment (and every NetBackup version) to call endpoints it may
  not need or support, increasing API load and failure surface.

These endpoints are also not universally available or relevant — malware and SLO
features depend on appliance configuration, and `/catalog/images` is heavier than
the core collectors.

## Decision

Introduce a small `subCollector` interface (`Name() string`,
`Collect(ctx, ch) error`) in `internal/exporter/subcollector.go`. Each optional
metric area is its own focused file (`collector_alerts.go`, `collector_malware.go`,
`collector_catalog.go`, `collector_slo.go`) implementing that interface, taking
the `NetBackupClient` interface for testability.

`NbuCollector` holds an `[]subCollector` built once at construction from config
(`buildSubCollectors`) and runs them via `runSubCollectors` alongside the existing
storage/jobs collection. Three properties are intentional:

1. **Opt-in, default off.** A new `collectors` config section
   (`alerts`/`malware`/`catalog`/`slo`, each `{enabled: bool}`) gates each
   collector; the zero value is disabled. Existing deployments and older NetBackup
   versions are unaffected unless an operator explicitly enables a collector.
2. **Per-collector graceful degradation.** `runSubCollectors` runs each collector
   concurrently and logs-and-skips on error; a failing endpoint never propagates
   or suppresses the others. This extends the existing graceful-degradation
   principle already used for storage/jobs.
3. **`nbu_up` stays tied to storage/jobs only.** The optional collectors never
   flip the core reachability signal, so enabling an unsupported endpoint cannot
   make the exporter look "down".

The core storage and jobs collection paths are left unchanged — they are not
retrofitted onto the interface in this change.

## Consequences

**Positive**

- New metric areas are added as small, independently testable files that follow a
  single established pattern.
- Operators pay only for the endpoints they enable; older appliances are safe by
  default.
- `prometheus.go` stays focused on the core collector and orchestration.

**Negative / trade-offs**

- Two parallel mechanisms now exist: the original storage/jobs code in
  `prometheus.go` and the `subCollector` framework. Storage/jobs were not migrated
  to keep this change focused; a future ADR could unify them if the duplication
  becomes a burden.
- Sub-collector metric descriptors are created per-collector and emitted as const
  metrics rather than registered through `NbuCollector.Describe` (the exporter
  already uses the unchecked-collector pattern).
- Curated label subsets (e.g. catalog malware/anomaly status) are required to keep
  cardinality bounded; this is a manual guard, not an automatic one.
