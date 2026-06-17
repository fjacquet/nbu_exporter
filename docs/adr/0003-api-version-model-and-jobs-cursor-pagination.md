# ADR-0003: NetBackup API version model and jobs cursor pagination

- **Status:** Accepted
- **Date:** 2026-06-17
- **Deciders:** Frederic Jacquet
- **Related:** [#34](https://github.com/fjacquet/nbu_exporter/pull/34),
  `docs/superpowers/specs/2026-06-16-nbu-10x-support-design.md`, ADR-0002,
  `exporter-standards` (version/generation detection; "absent, never zero" defensive parsing)

## Context

Two latent defects surfaced when validating the exporter against the **real NetBackup
OpenAPI specs** (the 10.3 / 10.5 / 11.0 / 11.2 bundles under `docs/veritas-*`) rather than
against assumptions:

1. **Wrong API version for NBU 10.x.** The exporter mapped NetBackup 10.0–10.4 to API
   media-type `version=3.0`. No NetBackup REST API has ever used `3.0` — the 10.3 bundle
   declares `version=10.0` for `/admin/jobs` and `/storage/storage-units`. Requesting `3.0`
   against a 10.x appliance returns a legacy/reduced representation (missing fields), which
   looked like "NBU 10.x lacks these features" but was really "we asked for the wrong
   version." (A reporter's "API v3.0" turned out to be the exporter's *Helm* release version,
   not a NetBackup API version.)

2. **Jobs pagination was offset-based, but the API is cursor-based.** `GET /admin/jobs` has
   been **cursor-paginated since API version 9.0** (`page[after]`/`page[before]` request
   params; `meta.pagination.next`/`prev` as opaque **string** cursors; `rangeTruncated`) —
   confirmed identical across the 10.3, 10.5, 11.0 **and** 11.2 specs. The exporter sent
   `page[offset]` and declared `Next int`, so the response failed to unmarshal (string cursor
   into an int) and **every `nbu_jobs_*` metric was silently dropped** on all modern
   NetBackup — the root cause of issue #13 (storage metrics present, jobs empty on NBU 11.0).
   `/storage/storage-units` remains **offset**-paginated.

## Decision

- **Version model:** support `10.0` (NBU 10.0–10.4), `12.0` (10.5), `13.0` (11.0), `14.0`
  (11.2); **drop `3.0`**. Auto-detection probes `models.SupportedAPIVersions` in descending
  order (`14.0 → 13.0 → 12.0 → 10.0`) and the first to respond wins. A single
  `Accept: application/vnd.netbackup+json;version=X` header is used for all endpoints — verified
  that `/admin/jobs` and `/storage/storage-units` answer at the same version per release.
- **Jobs pagination is cursor-based.** `Jobs.Meta.Pagination` carries
  `{Limit int, Next string, Prev string, RangeTruncated bool}`; `FetchJobDetails` omits
  `page[after]` on the first page and then sends `page[after]=<next>`, following
  `meta.pagination.next` until it is empty. `HandlePagination` is a generic cursor loop.
- **Storage pagination is left offset-based** — the two endpoints genuinely differ; they are
  deliberately *not* unified.
- **Validation is grounded in the checked-in OpenAPI bundles**, not in third-party
  assumptions; the 10.3 and 11.2 bundles were added to the repo as that source of truth.

## Consequences

**Positive**

- `nbu_jobs_*` metrics work across **all modern NetBackup** (10.x / 10.5 / 11.0 / 11.2),
  fixing the silent data-loss behind issue #13.
- NBU 10.x auto-detects with no manual `apiVersion`; the shipped config example for 10.x uses
  `10.0`.
- Architectural claims are backed by authoritative vendor specs rather than report wording.

**Negative / trade-offs**

- **Two pagination styles coexist** (jobs = cursor, storage = offset). This is intentional and
  matches the API; the code and this ADR guard against a well-meaning "unify pagination"
  refactor that would re-break jobs.
- The single `Accept`-version header assumes jobs and storage share one version per release
  (true across 10.0–14.0); revisit if NetBackup ever versions those endpoints independently.
- The OpenAPI bundles add ~10 MB to the repository, accepted as the validation source of
  truth (mirrors keeping vendor specs for the other family exporters).
