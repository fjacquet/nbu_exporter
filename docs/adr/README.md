# Architecture Decision Records

This directory records significant architectural and engineering decisions for
`nbu_exporter`, using lightweight [ADRs](https://adr.github.io/).

Each ADR captures the **context**, the **decision**, the **alternatives
considered**, and the **consequences**. ADRs are immutable once accepted; a
superseding decision gets a new ADR that references the old one.

## Index

| ADR | Title | Status | Date |
|-----|-------|--------|------|
| [0001](0001-tooling-baseline-sync-with-pflex.md) | Sync the tooling/CI/security baseline with pflex_exporter | Accepted | 2026-06-05 |
| [0002](0002-opt-in-sub-collector-framework.md) | Pluggable opt-in sub-collector framework | Accepted | 2026-06-14 |
| [0003](0003-api-version-model-and-jobs-cursor-pagination.md) | NetBackup API version model and jobs cursor pagination | Accepted | 2026-06-17 |
| [0004](0004-multisite-snapshot-collection-model.md) | Multi-site support via the snapshot collection model and a `site` identity label | Accepted | 2026-06-17 |
