# Feature 3 — Per-Client Last-Successful-Backup Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax. **One task per subagent (small), gate each on `make ci`, commit per task.** If a background dispatch stalls on the 600s watchdog, reconcile git state and finish that task inline. Never start the next task before the current one is green + committed.

**Goal:** Add an opt-in per-client metric — `nbu_client_last_successful_backup_timestamp_seconds{site,client}` — for each allowlisted client, the alerting primitive for "no backup in N hours".

**Architecture:** A new `perClientCollector` implementing the `subCollector` interface (alerts/malware/tape pattern), built per target by `buildSubCollectorsFor`, so its metrics buffer into the per-site snapshot and carry `site`. For each allowlisted client it issues one targeted `/admin/jobs` query (`filter clientName + jobType=BACKUP + status=0`, `sort=-endTime`, `page[limit]=1`) and emits that job's `endTime`. No lookback window.

**Tech Stack:** Go 1.26, prometheus/client_golang, testify; `make ci` gate. RTK: `rtk proxy …`. Commit trailer: `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`.

**Spec:** `docs/superpowers/specs/2026-06-17-feature3-per-client-design.md`

## Confirmed facts (codebase + `docs/veritas-10.5/admin.yaml`)

- `models.Jobs.Data[].Attributes` has `ClientName`, `JobType` (value `"BACKUP"`), `PolicyType`, `Status` (int), `EndTime` (`time.Time`).
- `jobsPath = "/admin/jobs"` (netbackup.go). `QueryParamFilter="filter"`, `QueryParamSort="sort"`, `QueryParamLimit="page[limit]"` (client.go). `sort` supports `-` prefix for descending. `status eq 0` = fully successful.
- Sub-collectors implement `subCollector` (`Name() string`, `Collect(ctx, ch) error`) and are built in `buildSubCollectorsFor(client NetBackupClient, cfg models.Config, site string)` (subcollector.go).

## File Structure

- `internal/models/Config.go` — add `PerClientConfig{Enabled bool, Allowlist []string}` + `PerClient PerClientConfig` to the `Collectors` struct.
- `internal/exporter/collector_perclient.go` *(new)* — `perClientCollector`.
- `internal/exporter/collector_perclient_test.go` *(new)* — tests + a URL-recording mock.
- `internal/exporter/subcollector.go` — register `perClient` in `buildSubCollectorsFor`.
- Docs: `docs/metrics.md`, `CHANGELOG.md`.

---

## Task 1: Config — `collectors.perClient` (enabled + allowlist)

**Files:** Modify `internal/models/Config.go`; Test `internal/models/Config_test.go`.

- [ ] **Step 1: Write the failing test**

```go
func TestCollectorsPerClientConfigDefaults(t *testing.T) {
	cfg := &Config{}
	if cfg.Collectors.PerClient.Enabled {
		t.Error("collectors.perClient should default to disabled")
	}
	if len(cfg.Collectors.PerClient.Allowlist) != 0 {
		t.Error("collectors.perClient.allowlist should default to empty")
	}
}
```

- [ ] **Step 2: Run — expect FAIL** (`PerClient` undefined). `rtk proxy go test ./internal/models/ -run TestCollectorsPerClient -v`.

- [ ] **Step 3: Implement** — in `internal/models/Config.go`, add the type (near `CollectorToggle`):

```go
// PerClientConfig is the opt-in per-client metrics toggle. Unlike CollectorToggle it
// carries an allowlist that bounds the high-cardinality client label; an empty
// allowlist emits no per-client series.
type PerClientConfig struct {
	Enabled   bool     `yaml:"enabled"`
	Allowlist []string `yaml:"allowlist"`
}
```

and add the field to the `Collectors` struct (after `Tape`):

```go
		PerClient PerClientConfig `yaml:"perClient"`
```

- [ ] **Step 4: Run — expect PASS.** `make ci` (`rtk proxy make ci`) green.
- [ ] **Step 5: Commit**

```bash
git add internal/models/Config.go internal/models/Config_test.go
git commit -m "feat(config): add collectors.perClient (enabled + allowlist)"
```

---

## Task 2: `perClientCollector` core — query + emit last success

**Files:** Create `internal/exporter/collector_perclient.go`, `internal/exporter/collector_perclient_test.go`.

- [ ] **Step 1: Write the failing test** (`collector_perclient_test.go`)

```go
package exporter

import (
	"context"
	"encoding/json"
	"errors"
	neturl "net/url"
	"testing"
	"time"

	"github.com/fjacquet/nbu_exporter/internal/models"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

// perClientMock records request URLs and returns a fixed JSON body (or an error
// when response == "").
type perClientMock struct {
	urls     []string
	response string
}

func (m *perClientMock) FetchData(_ context.Context, url string, target interface{}) error {
	m.urls = append(m.urls, url)
	if m.response == "" {
		return errors.New("jobs query failed")
	}
	return json.Unmarshal([]byte(m.response), target)
}
func (m *perClientMock) DetectAPIVersion(context.Context) (string, error) {
	return models.APIVersion140, nil
}
func (m *perClientMock) Close() error { return nil }

func perClientConfig(allowlist ...string) models.Config {
	cfg := testConfig()
	cfg.Collectors.PerClient.Enabled = true
	cfg.Collectors.PerClient.Allowlist = allowlist
	return cfg
}

func TestPerClient_EmitsLastSuccess(t *testing.T) {
	const endStr = "2026-06-16T10:00:00Z"
	mock := &perClientMock{response: `{"data":[{"attributes":{"clientName":"clientA","jobType":"BACKUP","policyType":"Standard","status":0,"endTime":"` + endStr + `"}}]}`}
	c := newPerClientCollector(mock, perClientConfig("clientA"), "site1")
	ch := make(chan prometheus.Metric, 8)
	require.NoError(t, c.Collect(context.Background(), ch))
	close(ch)

	var d dto.Metric
	got := 0
	var value float64
	for m := range ch {
		require.NoError(t, m.Write(&d))
		require.Equal(t, "site1", labelValue(&d, "site"))
		require.Equal(t, "clientA", labelValue(&d, "client"))
		value = d.GetGauge().GetValue()
		got++
	}
	require.Equal(t, 1, got, "one series for the one allowlisted client")
	want, _ := time.Parse(time.RFC3339, endStr)
	require.Equal(t, float64(want.Unix()), value)

	// The request carries the expected filter/sort/limit (decoded via net/url).
	require.Len(t, mock.urls, 1)
	q, err := neturl.Parse(mock.urls[0])
	require.NoError(t, err)
	require.Equal(t, "-endTime", q.Query().Get("sort"))
	require.Equal(t, "1", q.Query().Get("page[limit]"))
	filter := q.Query().Get("filter")
	require.Contains(t, filter, "clientName eq 'clientA'")
	require.Contains(t, filter, "jobType eq 'BACKUP'")
	require.Contains(t, filter, "status eq 0")
}

func TestPerClient_NoSuccessNoSeries(t *testing.T) {
	mock := &perClientMock{response: `{"data":[]}`}
	c := newPerClientCollector(mock, perClientConfig("clientA"), "site1")
	ch := make(chan prometheus.Metric, 4)
	require.NoError(t, c.Collect(context.Background(), ch))
	close(ch)
	require.Empty(t, ch, "a client with no successful backup emits no series")
}
```

- [ ] **Step 2: Run — expect FAIL** (`newPerClientCollector` undefined). `rtk proxy go test ./internal/exporter/ -run TestPerClient -v`.

- [ ] **Step 3: Implement** `internal/exporter/collector_perclient.go`

```go
package exporter

import (
	"context"
	"fmt"

	"github.com/fjacquet/nbu_exporter/internal/models"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

// perClientCollector is an opt-in sub-collector that emits, per allowlisted client,
// the timestamp of that client's most recent successful backup.
type perClientCollector struct {
	client    NetBackupClient
	cfg       models.Config
	site      string
	allowlist []string
	desc      *prometheus.Desc
}

func newPerClientCollector(client NetBackupClient, cfg models.Config, site string) *perClientCollector {
	return &perClientCollector{
		client:    client,
		cfg:       cfg,
		site:      site,
		allowlist: cfg.Collectors.PerClient.Allowlist,
		desc: prometheus.NewDesc(
			"nbu_client_last_successful_backup_timestamp_seconds",
			"Unix timestamp of the client's most recent successful backup",
			[]string{"site", "client"}, nil,
		),
	}
}

func (c *perClientCollector) Name() string { return "perclient" }

func (c *perClientCollector) Collect(ctx context.Context, ch chan<- prometheus.Metric) error {
	for _, name := range c.allowlist {
		c.collectClient(ctx, ch, name)
	}
	return nil
}

// collectClient queries the single most recent successful backup for one client and
// emits its endTime. A fetch error / no result is logged-and-skipped for that client.
func (c *perClientCollector) collectClient(ctx context.Context, ch chan<- prometheus.Metric, name string) {
	filter := fmt.Sprintf("clientName eq '%s' and jobType eq 'BACKUP' and status eq 0", name)
	url := c.cfg.BuildURL(jobsPath, map[string]string{
		QueryParamFilter: filter,
		QueryParamSort:   "-endTime",
		QueryParamLimit:  "1",
	})
	var resp models.Jobs
	if err := c.client.FetchData(ctx, url, &resp); err != nil {
		log.WithError(err).WithField("site", c.site).WithField("client", name).
			Warn("perClient: jobs query failed; skipping client")
		return
	}
	if len(resp.Data) == 0 {
		return // no successful backup on record for this client
	}
	end := resp.Data[0].Attributes.EndTime
	if end.IsZero() {
		return
	}
	ch <- prometheus.MustNewConstMetric(c.desc, prometheus.GaugeValue, float64(end.Unix()), c.site, name)
}
```

- [ ] **Step 4: Run — expect PASS.** `rtk proxy go test ./internal/exporter/ -run TestPerClient -v`.
- [ ] **Step 5: `make ci`** green.
- [ ] **Step 6: Commit**

```bash
git add internal/exporter/collector_perclient.go internal/exporter/collector_perclient_test.go
git commit -m "feat(exporter): per-client collector emits last-successful-backup timestamp"
```

---

## Task 3: Guards — empty allowlist, quoted names, error degradation

**Files:** Modify `internal/exporter/collector_perclient.go`, `internal/exporter/collector_perclient_test.go`.

- [ ] **Step 1: Write the failing tests** (append to `collector_perclient_test.go`)

```go
func TestPerClient_EmptyAllowlistEmitsNothing(t *testing.T) {
	mock := &perClientMock{response: `{"data":[]}`}
	c := newPerClientCollector(mock, perClientConfig(), "site1") // no clients
	ch := make(chan prometheus.Metric, 4)
	require.NoError(t, c.Collect(context.Background(), ch))
	close(ch)
	require.Empty(t, ch)
	require.Empty(t, mock.urls, "no queries when the allowlist is empty")
}

func TestPerClient_QuotedNameSkipped(t *testing.T) {
	mock := &perClientMock{response: `{"data":[]}`}
	c := newPerClientCollector(mock, perClientConfig("bad'name"), "site1")
	ch := make(chan prometheus.Metric, 4)
	require.NoError(t, c.Collect(context.Background(), ch))
	close(ch)
	require.Empty(t, mock.urls, "a name with a single quote is skipped (no unsafe filter)")
}

func TestPerClient_FetchErrorDegrades(t *testing.T) {
	mock := &perClientMock{response: ""} // FetchData returns an error
	c := newPerClientCollector(mock, perClientConfig("clientA"), "site1")
	ch := make(chan prometheus.Metric, 4)
	require.NoError(t, c.Collect(context.Background(), ch), "Collect never propagates a per-client error")
	close(ch)
	require.Empty(t, ch)
}
```

- [ ] **Step 2: Run — expect FAIL** (`TestPerClient_QuotedNameSkipped` fails — the quoted name is currently queried). `rtk proxy go test ./internal/exporter/ -run TestPerClient -v`.

- [ ] **Step 3: Implement** — add an empty-allowlist notice in `Collect` and a quote guard in `collectClient`:

```go
func (c *perClientCollector) Collect(ctx context.Context, ch chan<- prometheus.Metric) error {
	if len(c.allowlist) == 0 {
		log.WithField("site", c.site).Info("perClient enabled but allowlist empty; no per-client series emitted")
		return nil
	}
	for _, name := range c.allowlist {
		c.collectClient(ctx, ch, name)
	}
	return nil
}
```

and at the top of `collectClient`, before building the filter:

```go
	if strings.ContainsRune(name, '\'') {
		log.WithField("site", c.site).WithField("client", name).
			Warn("perClient: client name contains a single quote; skipping (cannot build a safe filter)")
		return
	}
```

Add `"strings"` to the imports of `collector_perclient.go`.

- [ ] **Step 4: Run — expect PASS** (all `TestPerClient*`).
- [ ] **Step 5: `make ci`** green.
- [ ] **Step 6: Commit**

```bash
git add internal/exporter/collector_perclient.go internal/exporter/collector_perclient_test.go
git commit -m "feat(exporter): per-client guards (empty allowlist, quoted-name skip, error degradation)"
```

---

## Task 4: Register the collector

**Files:** Modify `internal/exporter/subcollector.go`; Test `internal/exporter/collector_perclient_test.go`.

- [ ] **Step 1: Write the failing test** (append)

```go
func TestBuildSubCollectorsFor_PerClient(t *testing.T) {
	cfg := perClientConfig("clientA")
	subs := buildSubCollectorsFor(&errClient{}, cfg, "site1")
	found := false
	for _, s := range subs {
		if s.Name() == "perclient" {
			found = true
		}
	}
	require.True(t, found, "perClient collector should be built when collectors.perClient.enabled")
}
```

- [ ] **Step 2: Run — expect FAIL** (`perclient` not registered). `rtk proxy go test ./internal/exporter/ -run TestBuildSubCollectorsFor_PerClient -v`.

- [ ] **Step 3: Implement** — in `internal/exporter/subcollector.go`, add to `buildSubCollectorsFor` (after the `Tape` block):

```go
	if cfg.Collectors.PerClient.Enabled {
		subs = append(subs, newPerClientCollector(client, cfg, site))
	}
```

- [ ] **Step 4: Run — expect PASS.** `make ci` green.
- [ ] **Step 5: Commit**

```bash
git add internal/exporter/subcollector.go internal/exporter/collector_perclient_test.go
git commit -m "feat(exporter): register opt-in per-client collector"
```

---

## Task 5: Docs + CHANGELOG

**Files:** `docs/metrics.md`, `CHANGELOG.md`.

- [ ] **Step 1:** In `docs/metrics.md`, add a row to the opt-in collectors table:

```markdown
| `nbu_client_last_successful_backup_timestamp_seconds` | Gauge | `client` | `GET /admin/jobs` (per allowlisted client) | `endTime` of the latest `status=0` BACKUP for the client |
```

and add the `perClient` block to the `collectors:` example, plus a sentence: the `perClient` collector is opt-in and **requires an explicit `allowlist`** (empty ⇒ no series) because the `client` label is high-cardinality; it issues one targeted `/admin/jobs` query per allowlisted client.

```yaml
collectors:
  alerts:  { enabled: false }
  malware: { enabled: false }
  catalog: { enabled: false }
  slo:     { enabled: false }
  tape:    { enabled: false }
  perClient:
    enabled: false
    allowlist: []   # exact client names; empty => no per-client series
```

- [ ] **Step 2:** In `CHANGELOG.md` `[Unreleased]` → Added:

```markdown
- **Per-client last-successful-backup metric** (opt-in `collectors.perClient`, default off):
  `nbu_client_last_successful_backup_timestamp_seconds{site,client}` for each allowlisted client,
  from a targeted `/admin/jobs` query (latest `status=0` BACKUP). Exact-name allowlist bounds the
  `client` cardinality; an empty allowlist emits nothing. Enables a "no backup in N hours" alert.
```

- [ ] **Step 3:** `make ci` green.
- [ ] **Step 4: Commit**

```bash
git add docs/metrics.md CHANGELOG.md
git commit -m "docs: document opt-in per-client last-successful-backup metric"
```

---

## Final

- [ ] Whole-implementation review (spec coverage: metric ✓, targeted query filter/sort/limit ✓, allowlist gating ✓, empty=nothing ✓, quoted-name safety ✓, per-client error degradation ✓, `site` label ✓, opt-in registration ✓, docs ✓).
- [ ] `make ci` green.
- [ ] superpowers:finishing-a-development-branch.

## Acceptance criteria (from spec)

- Opt-in `perClient` collector, default off; emits `nbu_client_last_successful_backup_timestamp_seconds{site,client}` only for allowlisted clients with a successful backup on record.
- Empty allowlist ⇒ no series (logged). Names with `'` skipped. Per-client query failure logged + skipped, never affecting other clients / core metrics / `nbu_up`.
- One targeted `/admin/jobs` query per allowlisted client (`filter clientName+jobType=BACKUP+status=0`, `sort=-endTime`, `page[limit]=1`); no lookback window.
