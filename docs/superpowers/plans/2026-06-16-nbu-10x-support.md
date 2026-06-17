# NetBackup 10.x Support Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the exporter work correctly against modern NetBackup by negotiating API media-type `version=10.0` for NBU 10.x and switching jobs collection to cursor pagination (storage untouched).

**Architecture:** Two cohesive parts in one PR. Part A is a mechanical version-model change driven by one slice (`models.SupportedAPIVersions`), which the detector already iterates. Part B replaces the jobs offset-pagination model + loop with cursor semantics (`page[after]` + string `next` cursor), since `GET /admin/jobs` has been cursor-paginated since API 9.0 and the current `Next int` model fails to unmarshal the string cursor — silently zeroing all `nbu_jobs_*`. Storage uses a single `offset=0` request and is not touched.

**Tech Stack:** Go 1.26, resty/v2, testify, Prometheus client; `make ci` gate (fmt/vet/lint/test-race/govulncheck + 70% coverage via `.testcoverage.yml`).

**Spec:** `docs/superpowers/specs/2026-06-16-nbu-10x-support-design.md`

**Key facts (verified against `docs/veritas-10.3/`):**
- `/admin/jobs` and `/storage/storage-units` both answer at `version=10.0` — one `Accept` header covers both.
- `/admin/jobs` request params: `page[limit]`, `page[after]` (string cursor), `page[before]`; response `meta.pagination` = `{limit int, next string, prev string, rangeTruncated bool}`.
- `RTK note:` commands are auto-prefixed with `rtk`; to see raw `go test` output use `rtk proxy go test ...`.

---

## File Structure

- `internal/models/Config.go` — version constants + `SupportedAPIVersions` (replace `APIVersion30` with `APIVersion100`).
- `internal/models/Jobs.go` — jobs `Meta.Pagination` struct → cursor shape.
- `internal/exporter/client.go` — add `QueryParamAfter` constant.
- `internal/exporter/netbackup.go` — `FetchJobDetails` (cursor) + `HandlePagination` (cursor loop) + `FetchAllJobsFull` caller.
- `internal/exporter/version_detector.go`, `internal/exporter/prometheus.go` — doc-comment ladders only.
- `internal/testutil/constants.go` — `APIVersion30` → `APIVersion100`.
- Tests: `internal/models/Config_test.go`, `internal/models/Jobs_test.go`, `internal/exporter/netbackup_test.go`, `version_detection_integration_test.go`, `api_compatibility_test.go`, `end_to_end_test.go`, `integration_test.go`, `metrics_consistency_test.go`, `performance_test.go`.
- Fixtures: `internal/testdata/api-versions/jobs-response-v3.json` → `*-v10.json` (cursor meta).
- Docs: `README.md`, `CLAUDE.md`, `docs/config-examples/README.md`, `docs/config-examples/config-netbackup-10.0.yaml`.

---

## PART A — API version 10.0 (drop the bogus 3.0)

### Task A1: Version constants + supported list

**Files:**
- Modify: `internal/models/Config.go:17-29`
- Test: `internal/models/Config_test.go`

- [ ] **Step 1: Update the failing tests first**

In `internal/models/Config_test.go`, change the supported-versions expectation (around line 392) and the constants test (around line 412-414):

```go
// TestSupportedAPIVersions
expectedVersions := []string{"14.0", "13.0", "12.0", "10.0"}
```

```go
// TestAPIVersionConstants — replace the "APIVersion30 constant" case with:
{
    name:     "APIVersion100 constant",
    constant: APIVersion100,
    expected: "10.0",
},
```

In `TestConfigValidateAPIVersion` (around lines 168-169 and 987-988), change the two `apiVersion: "3.0"` cases to `apiVersion: "10.0"` and rename them to `"valid API version 10.0"` / `"supported version 10.0"`. Update the `SetDefaults` doc-comment reference at `Config_test.go:72` from `3.0` to `10.0`.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `rtk proxy go test ./internal/models/ -run 'TestSupportedAPIVersions|TestAPIVersionConstants|TestConfigValidateAPIVersion' -v`
Expected: FAIL — `APIVersion100` undefined / `"3.0"` still in `SupportedAPIVersions`.

- [ ] **Step 3: Make the version-model change**

In `internal/models/Config.go`, replace lines 17-18 and 29:

```go
	// APIVersion100 represents the NetBackup 10.0-10.4 API version (media-type version=10.0).
	APIVersion100 = "10.0"
```

```go
var SupportedAPIVersions = []string{APIVersion140, APIVersion130, APIVersion120, APIVersion100}
```

Also fix the `SetDefaults` doc-comment ladder at `Config.go:84`: `(14.0 -> 13.0 -> 12.0 -> 10.0)`.

- [ ] **Step 4: Run the tests to verify they pass**

Run: `rtk proxy go test ./internal/models/ -run 'TestSupportedAPIVersions|TestAPIVersionConstants|TestConfigValidateAPIVersion' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
rtk git add internal/models/Config.go internal/models/Config_test.go
rtk git commit -m "feat(version): use API version 10.0 for NBU 10.x, drop bogus 3.0"
```

### Task A2: testutil constant + version-detection tests

**Files:**
- Modify: `internal/testutil/constants.go:58`
- Modify: `internal/exporter/test_common.go:21`
- Modify: `internal/exporter/version_detection_integration_test.go` (3.0 → 10.0 throughout)

- [ ] **Step 1: Rename the testutil constant**

In `internal/testutil/constants.go`, replace line 58:

```go
	APIVersion100 = "version=10.0"
```

- [ ] **Step 2: Update the exporter alias**

In `internal/exporter/test_common.go:21`, replace:

```go
	apiVersion100 = testutil.APIVersion100
```

(If `apiVersion30` is referenced elsewhere in non-test code it would already have failed to compile — it is test-only.)

- [ ] **Step 3: Update version-detection integration test**

In `internal/exporter/version_detection_integration_test.go`, replace every `"3.0"` literal and `testutil.APIVersion30` reference with `"10.0"` / `testutil.APIVersion100`, including the supported-version cases (lines ~89-102), the mock-server branch (lines ~21-22, 145-146, 152), the expected detected version (lines ~181-182), and the fallback-order array (line ~186 → `[]string{"14.0", "13.0", "12.0", "10.0"}`).

- [ ] **Step 4: Run the package build + detection tests**

Run: `rtk proxy go test ./internal/exporter/ -run 'VersionDetection' -v`
Expected: PASS (and the package compiles — no `apiVersion30`/`APIVersion30` left).

- [ ] **Step 5: Commit**

```bash
rtk git add internal/testutil/constants.go internal/exporter/test_common.go internal/exporter/version_detection_integration_test.go
rtk git commit -m "test(version): switch detection tests/constants from 3.0 to 10.0"
```

### Task A3: Doc-comment ladders + shipped docs

**Files:**
- Modify: `internal/exporter/version_detector.go` (lines 2-3, 42-43, 84-91)
- Modify: `internal/exporter/prometheus.go:116`
- Modify: `README.md`, `CLAUDE.md`, `docs/config-examples/README.md`, `docs/config-examples/config-netbackup-10.0.yaml`

- [ ] **Step 1: Update Go doc comments**

In `version_detector.go` replace the three `14.0 → 13.0 → 12.0 → 3.0` ladders (and the "NetBackup 10.0-10.4" step that says `3.0`) with `14.0 → 13.0 → 12.0 → 10.0`. Same one-line fix in `prometheus.go:116`.

- [ ] **Step 2: Update docs**

- `README.md`: detection feature line → `(14.0, 13.0, 12.0, 10.0)`.
- `CLAUDE.md`: version_detector line → `(14.0 → 13.0 → 12.0 → 10.0 fallback)`.
- `docs/config-examples/README.md`: the "Detection Process" list — replace the `3.0` step with the `10.0` step (NetBackup 10.0-10.4).
- `docs/config-examples/config-netbackup-10.0.yaml`: change `apiVersion: "3.0"` → `apiVersion: "10.0"` and its comment.

- [ ] **Step 3: Verify no stray 3.0 remains as an API version**

Run: `rtk proxy grep -rn '"3.0"\|version=3.0\|→ 3.0\|-> 3.0' internal README.md CLAUDE.md docs/config-examples`
Expected: no matches (Helm/chart "v3.0.0" references elsewhere are unrelated and out of scope).

- [ ] **Step 4: Build**

Run: `rtk proxy go build ./...`
Expected: success.

- [ ] **Step 5: Commit**

```bash
rtk git add internal/exporter/version_detector.go internal/exporter/prometheus.go README.md CLAUDE.md docs/config-examples
rtk git commit -m "docs(version): document the 14.0->13.0->12.0->10.0 detection ladder"
```

---

## PART B — Jobs cursor pagination

### Task B1: Cursor pagination model

**Files:**
- Modify: `internal/models/Jobs.go:113-123`
- Test: `internal/models/Jobs_test.go`

- [ ] **Step 1: Write the failing unmarshal test**

Add to `internal/models/Jobs_test.go`:

```go
// TestJobsCursorPaginationUnmarshal is a regression test: NBU 10.x (version=10.0)
// returns meta.pagination.next as a STRING cursor, not an int. The old int model
// failed to unmarshal it, silently zeroing all job metrics.
func TestJobsCursorPaginationUnmarshal(t *testing.T) {
	payload := `{
      "data": [],
      "meta": {"pagination": {"limit": 100, "next": "eyJqb2JJZCI6NX0", "prev": "", "rangeTruncated": false}}
    }`
	var jobs Jobs
	if err := json.Unmarshal([]byte(payload), &jobs); err != nil {
		t.Fatalf("unmarshal cursor pagination: %v", err)
	}
	if jobs.Meta.Pagination.Next != "eyJqb2JJZCI6NX0" {
		t.Errorf("Next = %q, want the string cursor", jobs.Meta.Pagination.Next)
	}
	if jobs.Meta.Pagination.Limit != 100 {
		t.Errorf("Limit = %d, want 100", jobs.Meta.Pagination.Limit)
	}
}
```

Ensure `encoding/json` is imported in the test file.

- [ ] **Step 2: Run to verify it fails**

Run: `rtk proxy go test ./internal/models/ -run TestJobsCursorPaginationUnmarshal -v`
Expected: FAIL — `json: cannot unmarshal string into Go struct field ... .next of type int`.

- [ ] **Step 3: Replace the jobs pagination struct**

In `internal/models/Jobs.go`, replace the `Meta.Pagination` block (lines 113-123) with the cursor shape:

```go
	Meta struct {
		Pagination struct {
			Limit          int    `json:"limit"`
			Next           string `json:"next"`
			Prev           string `json:"prev"`
			RangeTruncated bool   `json:"rangeTruncated"`
		} `json:"pagination"`
	} `json:"meta"`
```

Leave the `Links` struct (lines 125-138) unchanged.

- [ ] **Step 4: Run to verify it passes**

Run: `rtk proxy go test ./internal/models/ -run TestJobsCursorPaginationUnmarshal -v`
Expected: PASS. (The exporter package will not yet compile — fixed in B3.)

- [ ] **Step 5: Commit**

```bash
rtk git add internal/models/Jobs.go internal/models/Jobs_test.go
rtk git commit -m "feat(jobs): model cursor-based pagination (string next/prev)"
```

### Task B2: page[after] query-param constant

**Files:**
- Modify: `internal/exporter/client.go:102-105`

- [ ] **Step 1: Add the constant**

In the query-param const block, add after `QueryParamOffset`:

```go
	QueryParamAfter  = "page[after]"  // Cursor for the next page (jobs endpoint, NBU API >= 9.0)
```

- [ ] **Step 2: Build (will still fail in netbackup.go — expected)**

Run: `rtk proxy go vet ./internal/exporter/ 2>&1 | head`
Expected: errors only about `FetchJobDetails`/`Meta.Pagination` (fixed in B3), not about `QueryParamAfter`.

- [ ] **Step 3: Commit**

```bash
rtk git add internal/exporter/client.go
rtk git commit -m "feat(client): add page[after] cursor query param constant"
```

### Task B3: Cursor-based FetchJobDetails + HandlePagination loop

**Files:**
- Modify: `internal/exporter/netbackup.go` — `FetchJobDetails` (185-244), `HandlePagination` (268-283), `FetchAllJobsFull` caller (364-366)
- Test: `internal/exporter/netbackup_test.go`

- [ ] **Step 1: Write the failing multi-page cursor test**

Add to `internal/exporter/netbackup_test.go` (uses the existing `createTestConfig` + httptest pattern; serves page 1 with a `next` cursor and page 2 with empty `next`):

```go
func TestFetchAllJobsFollowsCursor(t *testing.T) {
	var secondPageCursor string
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(contentTypeHeader, contentTypeJSON)
		after := r.URL.Query().Get("page[after]")
		if after == "" {
			// Page 1: two jobs + a next cursor.
			_, _ = w.Write([]byte(`{"data":[
			  {"attributes":{"jobType":"BACKUP","policyType":"Standard","status":0,"state":"DONE","kilobytesTransferred":10}},
			  {"attributes":{"jobType":"BACKUP","policyType":"Standard","status":0,"state":"DONE","kilobytesTransferred":20}}
			],"meta":{"pagination":{"limit":100,"next":"CURSOR2","prev":""}}}`))
			return
		}
		secondPageCursor = after
		// Page 2: one job + empty next (last page).
		_, _ = w.Write([]byte(`{"data":[
		  {"attributes":{"jobType":"BACKUP","policyType":"Standard","status":0,"state":"DONE","kilobytesTransferred":30}}
		],"meta":{"pagination":{"limit":100,"next":"","prev":"CURSOR1"}}}`))
	}))
	defer server.Close()

	cfg := createTestConfig(strings.TrimPrefix(server.URL, testSchemeHTTPS), "10.0")
	cfg.NbuServer.Scheme = "https"
	client := NewNbuClient(cfg)

	agg, err := FetchAllJobsFull(context.Background(), client, "1h")
	require.NoError(t, err)
	require.NotNil(t, agg)

	// Three jobs aggregated across both pages (offset code would stop after page 1).
	total := 0
	for _, v := range agg.Count {
		total += int(v)
	}
	assert.Equal(t, 3, total, "should aggregate jobs from both pages")
	assert.Equal(t, "CURSOR2", secondPageCursor, "second request must carry page[after]=CURSOR2")
}
```

Confirm the test file imports `context`, `net/http`, `net/http/httptest`, `strings`, and testify `assert`/`require` (match the other tests in this package).

- [ ] **Step 2: Run to verify it fails**

Run: `rtk proxy go test ./internal/exporter/ -run TestFetchAllJobsFollowsCursor -v`
Expected: FAIL — currently the package won't even compile (B1 changed the model); after fixing compile it would stop after page 1. This drives the implementation.

- [ ] **Step 3: Convert `FetchJobDetails` to cursor**

In `internal/exporter/netbackup.go`, change the signature and body of `FetchJobDetails` (185-244):

```go
func FetchJobDetails(
	ctx context.Context,
	client *NbuClient,
	agg *JobAggregator,
	cursor string,
	startTime time.Time,
) (string, error) {
	ctx, span := client.tracing.StartSpan(ctx, "netbackup.fetch_job_page", trace.SpanKindClient)
	defer span.End()

	var jobs models.Jobs

	queryParams := map[string]string{
		QueryParamLimit:  jobPageLimit,
		QueryParamSort:   "jobId",
		QueryParamFilter: fmt.Sprintf("endTime gt %s", utils.ConvertTimeToNBUDate(startTime)),
	}
	if cursor != "" {
		queryParams[QueryParamAfter] = cursor
	}

	url := client.cfg.BuildURL(jobsPath, queryParams)

	if err := client.FetchData(ctx, url, &jobs); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", fmt.Errorf("failed to fetch job details: %w", err)
	}

	span.SetAttributes(attribute.Int(telemetry.AttrNetBackupJobsInPage, len(jobs.Data)))

	if len(jobs.Data) == 0 {
		span.SetStatus(codes.Ok, "No more jobs to process")
		return "", nil
	}

	for _, job := range jobs.Data {
		aggregateJob(agg, job.Attributes)
	}

	span.SetStatus(codes.Ok, "Page fetched successfully")
	return jobs.Meta.Pagination.Next, nil
}
```

Note: the `AttrNetBackupPageOffset` span attribute is dropped (offset is meaningless for cursors). Run `rtk proxy grep -rn AttrNetBackupPageOffset internal` — if any test asserts it, remove that assertion in the same commit. `strconv` is still used by `aggregateJob` (queue reason), so leave the import.

- [ ] **Step 4: Convert `HandlePagination` to a cursor loop**

Replace `HandlePagination` (268-283) and its doc comment:

```go
// HandlePagination drives cursor-based pagination: it calls fetchFunc with the
// current cursor (empty for the first page) and follows the returned next cursor
// until it is empty. It honours context cancellation between pages.
func HandlePagination(ctx context.Context, fetchFunc func(ctx context.Context, cursor string) (string, error)) error {
	cursor := ""
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			next, err := fetchFunc(ctx, cursor)
			if err != nil {
				return err
			}
			if next == "" {
				return nil
			}
			cursor = next
		}
	}
}
```

- [ ] **Step 5: Update the caller in `FetchAllJobsFull`**

At `netbackup.go:364-366`, change the closure to pass the cursor:

```go
	err := HandlePagination(ctx, func(ctx context.Context, cursor string) (string, error) {
		return FetchJobDetails(ctx, client, agg, cursor, startTime)
	})
```

- [ ] **Step 6: Run the cursor test (and the package)**

Run: `rtk proxy go test ./internal/exporter/ -run TestFetchAllJobsFollowsCursor -v`
Expected: PASS — 3 jobs aggregated, `page[after]=CURSOR2` on the second request.

- [ ] **Step 7: Commit**

```bash
rtk git add internal/exporter/netbackup.go internal/exporter/netbackup_test.go
rtk git commit -m "feat(jobs): follow NBU cursor pagination (page[after]) instead of offset"
```

### Task B4: Update remaining jobs tests + fixtures to cursor

**Files:**
- Rename: `internal/testdata/api-versions/jobs-response-v3.json` → `jobs-response-v10.json`
- Modify: `jobs-response-v10.json`, `jobs-response-v12.json`, `jobs-response-v13.json`, **`jobs-response-v14.json`** — convert each `meta.pagination` to the cursor shape (jobs are cursor-paginated on **all** versions ≥ API 9.0, **including 14.0**). Storage fixtures (`storage-response-v*.json`) stay offset — untouched.
- Modify: `api_compatibility_test.go`, `end_to_end_test.go`, `integration_test.go`, `metrics_consistency_test.go`, `performance_test.go`

The jobs endpoint is cursor-paginated on **all** versions (breaking change since API 9.0), so every jobs fixture/mock must present cursor `meta.pagination` and stop on empty `next`. Apply this canonical single-page (last-page) meta to job-list fixtures/mocks:

```json
"meta": {"pagination": {"limit": 100, "next": "", "prev": "", "rangeTruncated": false}}
```

- [ ] **Step 1: Rename v3→v10 and convert ALL jobs version fixtures to cursor**

```bash
rtk git mv internal/testdata/api-versions/jobs-response-v3.json internal/testdata/api-versions/jobs-response-v10.json
```

Edit the `meta.pagination` of **all four** jobs fixtures — `jobs-response-v10.json`, `jobs-response-v12.json`, `jobs-response-v13.json`, `jobs-response-v14.json` — to the cursor shape above (single/last page). Update the fixture map in `api_compatibility_test.go:32` from `{"3.0", ".../jobs-response-v3.json"}` to `{"10.0", ".../jobs-response-v10.json"}`, and ensure the map **includes a `14.0` → `jobs-response-v14.json` entry** alongside 12.0/13.0.

- [ ] **Step 2: Add 14.0 to the cross-version matrices + cursor mocks**

In `end_to_end_test.go`, `integration_test.go`, `metrics_consistency_test.go`, `performance_test.go`: replace `versions := []string{"3.0", "12.0", "13.0"}` with **`{"10.0", "12.0", "13.0", "14.0"}`** (this is what proves 14.0 works under cursor pagination). Ensure any inline `/admin/jobs` mock returns cursor `meta.pagination` (empty `next` for single-page mocks; branch on `page[after]` for multi-page mocks). Remove offset assertions (`page[offset]`, `Meta.Pagination.Offset/Last`) on the jobs path.

- [ ] **Step 3: Run the full exporter package**

Run: `rtk proxy go test ./internal/exporter/ 2>/dev/null | tail -20`
Expected: `ok  github.com/fjacquet/nbu_exporter/internal/exporter`. Fix any remaining offset-based job assertions until green.

- [ ] **Step 4: Commit**

```bash
rtk git add internal/testdata internal/exporter
rtk git commit -m "test(jobs): convert job fixtures/mocks to cursor pagination across versions"
```

### Task B5: Full CI gate

- [ ] **Step 1: Run the whole suite**

Run: `rtk proxy go test ./... 2>/dev/null | tail -15`
Expected: all packages `ok`.

- [ ] **Step 2: Run the CI gate**

Run: `rtk proxy make ci`
Expected: vet clean, golangci-lint `0 issues`, all packages pass under `-race`, govulncheck `No vulnerabilities found`, coverage ≥ 70%.

- [ ] **Step 3: Manual sanity (optional, no appliance needed)**

Run: `rtk proxy go build -o /tmp/nbu_exporter . && /tmp/nbu_exporter --help | head`
Expected: builds and runs.

- [ ] **Step 4: Final commit (if any lint/coverage tidy-ups were needed)**

```bash
rtk git add -A
rtk git commit -m "chore(nbu10x): make ci green (lint/coverage tidy-ups)"
```

---

## Acceptance criteria (from spec)

- Against a mock cursor-paginated `/admin/jobs`, the exporter follows `next` across pages and emits `nbu_jobs_*` for all jobs (Task B3 test proves this).
- `apiVersion` omitted on NBU 10.x → detection resolves `10.0`; jobs via cursor, storage via offset; full metrics.
- No `3.0` remains in `SupportedAPIVersions`, the detection ladder, or shipped config examples.
- **14.0 (NBU 11.2) still works — spec-confirmed:** `docs/veritas-11.2/admin.yaml` (API `version=14.0`) shows `/admin/jobs` uses the identical cursor contract (`page[after]`/`page[before]`, string `next`/`prev`, `rangeTruncated`). 14.0 is in the cross-version test matrix with a cursor jobs fixture, proving the fix repairs (not breaks) 14.0 — which was already offset-broken before.
- No change to emitted metric names or labels; storage pagination untouched.

## Out of scope (do not do here)

- Storage pagination (single `offset=0` request; >100 storage units truncation is a separate latent issue).
- Multi-site, tape, per-client metrics, alerting (separate items in the deferrals doc).
