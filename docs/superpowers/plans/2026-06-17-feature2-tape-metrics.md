# Feature 2 — Tape / Drive Metrics Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax. **One task per subagent (small), gate each on `make ci`, commit per task.** Background subagents in this environment stall on a 600s watchdog for large tasks — if a dispatch stalls, reconcile git state and finish that task inline. Never start the next task before the current one is green + committed.

**Goal:** Add an opt-in `tape` sub-collector exposing NetBackup tape/drive health (drives by state/type/robot, tape-volume status mix, robot device-host count) over REST, default-off, multi-site-aware.

**Architecture:** A new `tapeCollector` implementing the existing `subCollector` interface (alerts/malware/catalog/SLO pattern), built per target by `buildSubCollectorsFor`, so its metrics buffer into the per-site snapshot and carry the `site` label. It reads three `/storage/` endpoints with per-endpoint graceful degradation; tape-media is offset-paginated (loop until a short page).

**Tech Stack:** Go 1.26, prometheus/client_golang, testify; `make ci` gate (fmt/vet/lint/test-race/govulncheck + 70% coverage). RTK: `rtk proxy go test …` / `rtk proxy make ci`. Commit trailer: `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`.

**Spec:** `docs/superpowers/specs/2026-06-17-feature2-tape-metrics-design.md`

## Confirmed API facts (from `docs/veritas-10.5/storage.yaml`)

- `GET /storage/drives` — `data[].attributes`: `driveName`, `driveStatus` (`DRIVE_STATUS_UP|DOWN|MIXED|DISABLED`), `driveType` (`DT_HCART`…), `robotType` (`TLD|ACS|NOT_ROBOTIC|NA`), `robotNumber` (int), `deviceHost`. Exposes `page[offset]`/`page[limit]`. Present on NBU 10.0+.
- `GET /storage/tape-media` — `data[].attributes`: `barcode`, `mediaType` (`HCART|DLT|…`), `mediaStatus` (free-form string, e.g. `ACTIVE MULTIPLEXED`), `robotType`, `robotNumber`. `meta.pagination` like storage-units. NBU 10.5+.
- `GET /storage/robots-device-hosts` — `data[]`: `{id: <hostname>, type: "deviceHost"}`. NBU 10.5+.

## Metric set (every series carries `site` first)

| Metric | Labels (after `site`) |
|---|---|
| `nbu_tape_drives_count` (gauge) | `state`, `drive_type`, `robot_type` |
| `nbu_tape_drive_info` (gauge =1) | `drive_name`, `media_server`, `drive_type`, `robot_number`, `state` |
| `nbu_tape_media_count` (gauge) | `media_type`, `status` |
| `nbu_tape_robot_device_hosts` (gauge) | *(site only)* |

`state` label = `driveStatus` with the `DRIVE_STATUS_` prefix stripped (`UP`/`DOWN`/`MIXED`/`DISABLED`).

## File Structure

- `internal/models/Config.go` — add `Tape CollectorToggle` to the `Collectors` struct.
- `internal/models/Tape.go` *(new)* — `TapeDrives`, `TapeMedia`, `RobotDeviceHosts` JSON:API response structs.
- `internal/exporter/collector_tape.go` *(new)* — `tapeCollector` (implements `subCollector`); helpers for the three endpoints + offset loop.
- `internal/exporter/collector_tape_test.go` *(new)* — table/fixture tests + a path-routing mock client.
- `internal/exporter/subcollector.go` — register `tape` in `buildSubCollectorsFor`.
- `testdata/api-versions/` — new fixtures: `drives-response.json`, `tape-media-page1.json`, `tape-media-page2.json`, `robots-device-hosts-response.json`.
- Docs: `docs/metrics.md`, `docs/config-examples/`, `CHANGELOG.md`.

---

## Task 1: Config — `collectors.tape` toggle

**Files:** Modify `internal/models/Config.go`; Test `internal/models/Config_test.go`.

- [ ] **Step 1: Write the failing test**

Add to `internal/models/Config_test.go`:

```go
func TestCollectorsTapeToggleDefaultsDisabled(t *testing.T) {
	cfg := &Config{}
	if cfg.Collectors.Tape.Enabled {
		t.Error("collectors.tape should default to disabled")
	}
}
```

- [ ] **Step 2: Run — expect FAIL** (compile error: `Tape` undefined)

Run: `rtk proxy go test ./internal/models/ -run TestCollectorsTapeToggle -v`
Expected: build failure — `cfg.Collectors.Tape undefined`.

- [ ] **Step 3: Implement**

In `internal/models/Config.go`, add `Tape` to the `Collectors` struct (after `SLO`):

```go
	Collectors struct {
		Alerts  CollectorToggle `yaml:"alerts"`
		Malware CollectorToggle `yaml:"malware"`
		Catalog CollectorToggle `yaml:"catalog"`
		SLO     CollectorToggle `yaml:"slo"`
		Tape    CollectorToggle `yaml:"tape"`
	} `yaml:"collectors"`
```

- [ ] **Step 4: Run — expect PASS** (`rtk proxy go test ./internal/models/ -run TestCollectorsTapeToggle -v`).
- [ ] **Step 5: Commit**

```bash
git add internal/models/Config.go internal/models/Config_test.go
git commit -m "feat(config): add collectors.tape toggle (default off)"
```

---

## Task 2: Response models

**Files:** Create `internal/models/Tape.go`. (No standalone test — exercised by the collector tests in later tasks; this task only adds types so the package still builds.)

- [ ] **Step 1: Create `internal/models/Tape.go`**

```go
package models

// TapeDrives is the response from GET /storage/drives (JSON:API).
type TapeDrives struct {
	Data []struct {
		Attributes struct {
			DriveName   string `json:"driveName"`
			DriveStatus string `json:"driveStatus"` // DRIVE_STATUS_UP|DOWN|MIXED|DISABLED
			DriveType   string `json:"driveType"`   // DT_HCART, DT_DLT, ...
			RobotType   string `json:"robotType"`   // TLD, ACS, NOT_ROBOTIC, NA
			RobotNumber int    `json:"robotNumber"`
			DeviceHost  string `json:"deviceHost"`
		} `json:"attributes"`
	} `json:"data"`
}

// TapeMedia is the response from GET /storage/tape-media (JSON:API).
type TapeMedia struct {
	Data []struct {
		Attributes struct {
			Barcode     string `json:"barcode"`
			MediaType   string `json:"mediaType"`   // HCART, DLT, HC_CLN, ...
			MediaStatus string `json:"mediaStatus"` // free-form, e.g. "ACTIVE MULTIPLEXED"
			RobotType   string `json:"robotType"`
			RobotNumber int    `json:"robotNumber"`
		} `json:"attributes"`
	} `json:"data"`
}

// RobotDeviceHosts is the response from GET /storage/robots-device-hosts (JSON:API).
// Each entry is a configured robot device host; data[].id is the hostname.
type RobotDeviceHosts struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}
```

- [ ] **Step 2: Run — expect PASS** (`rtk proxy go build ./...`). Expected: builds clean.
- [ ] **Step 3: Commit**

```bash
git add internal/models/Tape.go
git commit -m "feat(models): tape/drive/robot-device-host JSON:API response structs"
```

---

## Task 3: Drive metrics (`nbu_tape_drives_count` + `nbu_tape_drive_info`)

**Files:** Create `internal/exporter/collector_tape.go`, `internal/exporter/collector_tape_test.go`; create `testdata/api-versions/drives-response.json`.

- [ ] **Step 1: Create the drives fixture** `testdata/api-versions/drives-response.json`

```json
{
  "data": [
    {"attributes": {"driveName": "drive0", "driveStatus": "DRIVE_STATUS_UP",   "driveType": "DT_HCART", "robotType": "TLD", "robotNumber": 0, "deviceHost": "ms1.example.com"}},
    {"attributes": {"driveName": "drive1", "driveStatus": "DRIVE_STATUS_UP",   "driveType": "DT_HCART", "robotType": "TLD", "robotNumber": 0, "deviceHost": "ms1.example.com"}},
    {"attributes": {"driveName": "drive2", "driveStatus": "DRIVE_STATUS_DOWN", "driveType": "DT_HCART", "robotType": "TLD", "robotNumber": 0, "deviceHost": "ms1.example.com"}}
  ]
}
```

- [ ] **Step 2: Write the failing test** `internal/exporter/collector_tape_test.go`

```go
package exporter

import (
	"context"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

// tapeRoutedClient is a NetBackupClient mock that serves a fixture per endpoint
// path (and per offset for paginated endpoints). A path mapped to "" yields an error
// (used to exercise per-endpoint graceful degradation).
type tapeRoutedClient struct {
	t       *testing.T
	byPath  map[string]string // path substring -> fixture file ("" => return error)
}

func (c *tapeRoutedClient) FetchData(_ context.Context, url string, target interface{}) error {
	c.t.Helper()
	for sub, fixture := range c.byPath {
		if strings.Contains(url, sub) {
			if fixture == "" {
				return errors.New("endpoint unavailable")
			}
			data, err := os.ReadFile(fixture)
			require.NoError(c.t, err)
			return json.Unmarshal(data, target)
		}
	}
	c.t.Fatalf("unexpected URL: %s", url)
	return nil
}
func (c *tapeRoutedClient) DetectAPIVersion(context.Context) (string, error) { return models.APIVersion140, nil }
func (c *tapeRoutedClient) Close() error                                     { return nil }

func TestTapeCollector_Drives(t *testing.T) {
	client := &tapeRoutedClient{t: t, byPath: map[string]string{
		"/storage/drives":              "../../testdata/api-versions/drives-response.json",
		"/storage/tape-media":          "",
		"/storage/robots-device-hosts": "",
	}}
	c := newTapeCollector(client, testConfig(), "site1")
	ch := make(chan prometheus.Metric, 64)
	require.NoError(t, c.Collect(context.Background(), ch))
	close(ch)

	driveCounts := map[string]float64{} // "state|drive_type|robot_type" -> value
	infoSites := 0
	for m := range ch {
		var d dto.Metric
		require.NoError(t, m.Write(&d))
		require.Equal(t, "site1", labelValue(&d, "site"))
		desc := m.Desc().String()
		switch {
		case strings.Contains(desc, "nbu_tape_drives_count"):
			key := labelValue(&d, "state") + "|" + labelValue(&d, "drive_type") + "|" + labelValue(&d, "robot_type")
			driveCounts[key] = d.GetGauge().GetValue()
		case strings.Contains(desc, "nbu_tape_drive_info"):
			require.Equal(t, float64(1), d.GetGauge().GetValue())
			infoSites++
		}
	}
	require.Equal(t, float64(2), driveCounts["UP|DT_HCART|TLD"])
	require.Equal(t, float64(1), driveCounts["DOWN|DT_HCART|TLD"])
	require.Equal(t, 3, infoSites, "one nbu_tape_drive_info per drive")
}
```

(Add imports `encoding/json`, `errors`, `os`, and `github.com/fjacquet/nbu_exporter/internal/models` to the test file.)

- [ ] **Step 3: Run — expect FAIL** (`newTapeCollector` undefined).

Run: `rtk proxy go test ./internal/exporter/ -run TestTapeCollector_Drives -v`
Expected: build failure — `undefined: newTapeCollector`.

- [ ] **Step 4: Implement** `internal/exporter/collector_tape.go`

```go
package exporter

import (
	"context"
	"strings"

	"github.com/fjacquet/nbu_exporter/internal/models"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

const (
	drivesPath    = "/storage/drives"
	tapeMediaPath = "/storage/tape-media"
	robotHostPath = "/storage/robots-device-hosts"
)

// tapeCollector is an opt-in sub-collector for tape/drive health. It reads
// /storage/drives, /storage/tape-media and /storage/robots-device-hosts, with
// per-endpoint graceful degradation (a missing endpoint on older NetBackup, or a
// permission error, is logged and skipped).
type tapeCollector struct {
	client NetBackupClient
	cfg    models.Config
	site   string

	drivesCount     *prometheus.Desc
	driveInfo       *prometheus.Desc
	mediaCount      *prometheus.Desc
	robotHostsCount *prometheus.Desc
}

func newTapeCollector(client NetBackupClient, cfg models.Config, site string) *tapeCollector {
	return &tapeCollector{
		client: client,
		cfg:    cfg,
		site:   site,
		drivesCount: prometheus.NewDesc(
			"nbu_tape_drives_count",
			"Number of tape drives by status, drive type and robot type",
			[]string{"site", "state", "drive_type", "robot_type"}, nil,
		),
		driveInfo: prometheus.NewDesc(
			"nbu_tape_drive_info",
			"Tape drive info (always 1; metadata in labels)",
			[]string{"site", "drive_name", "media_server", "drive_type", "robot_number", "state"}, nil,
		),
		mediaCount: prometheus.NewDesc(
			"nbu_tape_media_count",
			"Number of tape volumes by media type and status",
			[]string{"site", "media_type", "status"}, nil,
		),
		robotHostsCount: prometheus.NewDesc(
			"nbu_tape_robot_device_hosts",
			"Number of device hosts that have robots configured",
			[]string{"site"}, nil,
		),
	}
}

func (c *tapeCollector) Name() string { return "tape" }

// driveState strips the DRIVE_STATUS_ prefix so the label reads UP/DOWN/MIXED/DISABLED.
func driveState(s string) string { return strings.TrimPrefix(s, "DRIVE_STATUS_") }

func (c *tapeCollector) Collect(ctx context.Context, ch chan<- prometheus.Metric) error {
	c.collectDrives(ctx, ch)
	return nil
}

func (c *tapeCollector) collectDrives(ctx context.Context, ch chan<- prometheus.Metric) {
	url := c.cfg.BuildURL(drivesPath, map[string]string{QueryParamLimit: pageLimit, QueryParamOffset: "0"})
	var resp models.TapeDrives
	if err := c.client.FetchData(ctx, url, &resp); err != nil {
		log.WithError(err).WithField("site", c.site).Warn("tape: drives fetch failed; skipping")
		return
	}
	type key struct{ state, driveType, robotType string }
	counts := map[key]float64{}
	for _, d := range resp.Data {
		a := d.Attributes
		state := driveState(a.DriveStatus)
		counts[key{state, a.DriveType, a.RobotType}]++
		ch <- prometheus.MustNewConstMetric(c.driveInfo, prometheus.GaugeValue, 1,
			c.site, a.DriveName, a.DeviceHost, a.DriveType, strconv.Itoa(a.RobotNumber), state)
	}
	for k, v := range counts {
		ch <- prometheus.MustNewConstMetric(c.drivesCount, prometheus.GaugeValue, v,
			c.site, k.state, k.driveType, k.robotType)
	}
}
```

(Add `strconv` to the import list.)

- [ ] **Step 5: Run — expect PASS** (`rtk proxy go test ./internal/exporter/ -run TestTapeCollector_Drives -v`).
- [ ] **Step 6: `make ci` checkpoint** green.
- [ ] **Step 7: Commit**

```bash
git add internal/exporter/collector_tape.go internal/exporter/collector_tape_test.go testdata/api-versions/drives-response.json
git commit -m "feat(exporter): tape collector drive metrics (count + per-drive info)"
```

---

## Task 4: Tape-media metrics with offset pagination

**Files:** Modify `internal/exporter/collector_tape.go`, `internal/exporter/collector_tape_test.go`; create `testdata/api-versions/tape-media-page1.json`, `tape-media-page2.json`.

- [ ] **Step 1: Create two-page media fixtures**

`testdata/api-versions/tape-media-page1.json` (the loop terminates on an **empty** page — it advances `page[offset]` by the number of rows returned and stops when a page has zero rows — so small fixtures work: page1 has 2 rows, page2 has 1, and any further offset returns an empty page):

```json
{
  "data": [
    {"attributes": {"barcode": "ABC001", "mediaType": "HCART", "mediaStatus": "ACTIVE", "robotType": "TLD", "robotNumber": 0}},
    {"attributes": {"barcode": "ABC002", "mediaType": "HCART", "mediaStatus": "ACTIVE", "robotType": "TLD", "robotNumber": 0}}
  ]
}
```

`testdata/api-versions/tape-media-page2.json`:

```json
{
  "data": [
    {"attributes": {"barcode": "ABC003", "mediaType": "HCART", "mediaStatus": "FROZEN", "robotType": "TLD", "robotNumber": 0}}
  ]
}
```

- [ ] **Step 2: Write the failing test** — add to `collector_tape_test.go`. This needs the mock to return page1 then page2 by `page[offset]`. Extend `tapeRoutedClient.FetchData` to honor an optional per-path offset map (add field `byOffset map[string]map[string]string` keyed by path substring then offset value), and add this test:

```go
func TestTapeCollector_MediaPaginated(t *testing.T) {
	client := &tapeRoutedClient{
		t: t,
		byPath: map[string]string{
			"/storage/drives":              "",
			"/storage/robots-device-hosts": "",
		},
		byOffset: map[string]map[string]string{
			"/storage/tape-media": {
				"0": "../../testdata/api-versions/tape-media-page1.json",
				"2": "../../testdata/api-versions/tape-media-page2.json",
			},
		},
	}
	c := newTapeCollector(client, testConfig(), "site1")
	ch := make(chan prometheus.Metric, 64)
	require.NoError(t, c.Collect(context.Background(), ch))
	close(ch)

	media := map[string]float64{} // "media_type|status" -> value
	for m := range ch {
		var d dto.Metric
		require.NoError(t, m.Write(&d))
		if strings.Contains(m.Desc().String(), "nbu_tape_media_count") {
			require.Equal(t, "site1", labelValue(&d, "site"))
			media[labelValue(&d, "media_type")+"|"+labelValue(&d, "status")] = d.GetGauge().GetValue()
		}
	}
	require.Equal(t, float64(2), media["HCART|ACTIVE"], "both pages aggregated")
	require.Equal(t, float64(1), media["HCART|FROZEN"])
}
```

Update `tapeRoutedClient.FetchData` to check `byOffset` first (parse the `page[offset]=` value out of `url`; an offset not in the map returns an empty `{"data":[]}` page, which is what ends the collector's loop). The collector advances `page[offset]` by the number of rows returned and stops on the first empty page:

```go
func (c *tapeRoutedClient) FetchData(_ context.Context, url string, target interface{}) error {
	c.t.Helper()
	for sub, offsets := range c.byOffset {
		if strings.Contains(url, sub) {
			off := "0"
			if i := strings.Index(url, "page[offset]="); i >= 0 {
				rest := url[i+len("page[offset]="):]
				if j := strings.IndexByte(rest, '&'); j >= 0 {
					off = rest[:j]
				} else {
					off = rest
				}
			}
			fixture, ok := offsets[off]
			if !ok { // past the last page -> empty result
				return json.Unmarshal([]byte(`{"data":[]}`), target)
			}
			data, err := os.ReadFile(fixture)
			require.NoError(c.t, err)
			return json.Unmarshal(data, target)
		}
	}
	for sub, fixture := range c.byPath { /* unchanged from Task 3 */ }
	c.t.Fatalf("unexpected URL: %s", url)
	return nil
}
```

(Keep the `byPath` loop body from Task 3 after the `byOffset` loop. Add `byOffset map[string]map[string]string` to the struct.)

- [ ] **Step 3: Run — expect FAIL** (no media metrics emitted yet).

Run: `rtk proxy go test ./internal/exporter/ -run TestTapeCollector_MediaPaginated -v`
Expected: FAIL — `media["HCART|ACTIVE"]` is 0.

- [ ] **Step 4: Implement** — add media collection + offset loop to `collector_tape.go`, and call it from `Collect`:

```go
func (c *tapeCollector) Collect(ctx context.Context, ch chan<- prometheus.Metric) error {
	c.collectDrives(ctx, ch)
	c.collectMedia(ctx, ch)
	return nil
}

func (c *tapeCollector) collectMedia(ctx context.Context, ch chan<- prometheus.Metric) {
	type key struct{ mediaType, status string }
	counts := map[key]float64{}
	offset := 0
	for page := 0; page < maxMediaPages; page++ {
		url := c.cfg.BuildURL(tapeMediaPath, map[string]string{
			QueryParamLimit:  pageLimit,
			QueryParamOffset: strconv.Itoa(offset),
		})
		var resp models.TapeMedia
		if err := c.client.FetchData(ctx, url, &resp); err != nil {
			log.WithError(err).WithField("site", c.site).Warn("tape: tape-media fetch failed; skipping")
			return
		}
		if len(resp.Data) == 0 { // empty page: no more rows
			break
		}
		for _, d := range resp.Data {
			counts[key{d.Attributes.MediaType, d.Attributes.MediaStatus}]++
		}
		offset += len(resp.Data) // advance by rows returned; loop ends on the next empty page
		if ctx.Err() != nil {
			break
		}
		if page == maxMediaPages-1 {
			log.WithField("site", c.site).Warnf("tape: tape-media pagination hit the %d-page cap; counts may be truncated", maxMediaPages)
		}
	}
	for k, v := range counts {
		ch <- prometheus.MustNewConstMetric(c.mediaCount, prometheus.GaugeValue, v, c.site, k.mediaType, k.status)
	}
}
```

Add `const maxMediaPages = 1000` near the path constants in `collector_tape.go` — a safety cap (≈100k volumes at `pageLimit=100`) so a backend that ignores `page[offset]` cannot loop forever; hitting it is logged (no silent truncation).

- [ ] **Step 5: Run — expect PASS** (`rtk proxy go test ./internal/exporter/ -run TestTapeCollector -v`). Both drive + media tests pass.
- [ ] **Step 6: `make ci` checkpoint** green.
- [ ] **Step 7: Commit**

```bash
git add internal/exporter/collector_tape.go internal/exporter/collector_tape_test.go testdata/api-versions/tape-media-page1.json testdata/api-versions/tape-media-page2.json
git commit -m "feat(exporter): tape collector media metrics with offset pagination"
```

---

## Task 5: Robot device-host count, registration, and graceful degradation

**Files:** Modify `internal/exporter/collector_tape.go`, `internal/exporter/subcollector.go`, `internal/exporter/collector_tape_test.go`; create `testdata/api-versions/robots-device-hosts-response.json`.

- [ ] **Step 1: Create the robots fixture** `testdata/api-versions/robots-device-hosts-response.json`

```json
{
  "data": [
    {"id": "host1.example.com", "type": "deviceHost"},
    {"id": "host2.example.com", "type": "deviceHost"}
  ]
}
```

- [ ] **Step 2: Write the failing tests** — add to `collector_tape_test.go`:

```go
func TestTapeCollector_RobotHostsAndRegistration(t *testing.T) {
	client := &tapeRoutedClient{t: t, byPath: map[string]string{
		"/storage/drives":              "",
		"/storage/tape-media":          "",
		"/storage/robots-device-hosts": "../../testdata/api-versions/robots-device-hosts-response.json",
	}}
	c := newTapeCollector(client, testConfig(), "site1")
	ch := make(chan prometheus.Metric, 16)
	require.NoError(t, c.Collect(context.Background(), ch))
	close(ch)

	var got float64
	found := false
	for m := range ch {
		var d dto.Metric
		require.NoError(t, m.Write(&d))
		if strings.Contains(m.Desc().String(), "nbu_tape_robot_device_hosts") {
			require.Equal(t, "site1", labelValue(&d, "site"))
			got = d.GetGauge().GetValue()
			found = true
		}
	}
	require.True(t, found, "nbu_tape_robot_device_hosts must be emitted")
	require.Equal(t, float64(2), got)
}

// TestTapeCollector_GracefulDegradation: all endpoints fail -> Collect returns nil, emits nothing.
func TestTapeCollector_GracefulDegradation(t *testing.T) {
	client := &tapeRoutedClient{t: t, byPath: map[string]string{
		"/storage/drives":              "",
		"/storage/tape-media":          "",
		"/storage/robots-device-hosts": "",
	}}
	c := newTapeCollector(client, testConfig(), "site1")
	ch := make(chan prometheus.Metric, 4)
	require.NoError(t, c.Collect(context.Background(), ch))
	close(ch)
	require.Empty(t, ch)
}

// TestBuildSubCollectorsFor_Tape: the tape collector is built when enabled.
func TestBuildSubCollectorsFor_Tape(t *testing.T) {
	cfg := testConfig()
	cfg.Collectors.Tape.Enabled = true
	subs := buildSubCollectorsFor(&errClient{}, cfg, "site1")
	found := false
	for _, s := range subs {
		if s.Name() == "tape" {
			found = true
		}
	}
	require.True(t, found, "tape collector should be built when collectors.tape.enabled")
}
```

- [ ] **Step 3: Run — expect FAIL** (robots metric not emitted; tape not registered).

Run: `rtk proxy go test ./internal/exporter/ -run 'TestTapeCollector_RobotHostsAndRegistration|TestTapeCollector_GracefulDegradation|TestBuildSubCollectorsFor_Tape' -v`
Expected: FAIL.

- [ ] **Step 4: Implement** — add robots collection and call it from `Collect`:

```go
func (c *tapeCollector) Collect(ctx context.Context, ch chan<- prometheus.Metric) error {
	c.collectDrives(ctx, ch)
	c.collectMedia(ctx, ch)
	c.collectRobotHosts(ctx, ch)
	return nil
}

func (c *tapeCollector) collectRobotHosts(ctx context.Context, ch chan<- prometheus.Metric) {
	url := c.cfg.BuildURL(robotHostPath, map[string]string{QueryParamLimit: pageLimit})
	var resp models.RobotDeviceHosts
	if err := c.client.FetchData(ctx, url, &resp); err != nil {
		log.WithError(err).WithField("site", c.site).Warn("tape: robots-device-hosts fetch failed; skipping")
		return
	}
	ch <- prometheus.MustNewConstMetric(c.robotHostsCount, prometheus.GaugeValue, float64(len(resp.Data)), c.site)
}
```

Register in `internal/exporter/subcollector.go` `buildSubCollectorsFor`, after the SLO block:

```go
	if cfg.Collectors.Tape.Enabled {
		subs = append(subs, newTapeCollector(client, cfg, site))
	}
```

- [ ] **Step 5: Run — expect PASS** (the three tests above + the whole `TestTapeCollector*` set).
- [ ] **Step 6: `make ci` checkpoint** green (`rtk proxy make ci`).
- [ ] **Step 7: Commit**

```bash
git add internal/exporter/collector_tape.go internal/exporter/subcollector.go internal/exporter/collector_tape_test.go testdata/api-versions/robots-device-hosts-response.json
git commit -m "feat(exporter): tape collector robot-device-host count + register opt-in tape collector"
```

---

## Task 6: Docs + CHANGELOG

**Files:** `docs/metrics.md`, `docs/config-examples/README.md`, `CHANGELOG.md`.

- [ ] **Step 1:** In `docs/metrics.md`, under "## NetBackup 11.2 opt-in collectors", add the four tape metrics to the table and a note that the `tape` collector covers `/storage/drives` (NBU 10.0+), `/storage/tape-media` + `/storage/robots-device-hosts` (10.5+), with per-endpoint graceful degradation; and add `tape: { enabled: false }` to the `collectors:` example block.

```markdown
| `nbu_tape_drives_count` | Gauge | `state`, `drive_type`, `robot_type` | `GET /storage/drives` | Drives grouped by status/type/robot type |
| `nbu_tape_drive_info` | Gauge | `drive_name`, `media_server`, `drive_type`, `robot_number`, `state` | `GET /storage/drives` | One series per drive (value always 1) |
| `nbu_tape_media_count` | Gauge | `media_type`, `status` | `GET /storage/tape-media` | Tape volumes grouped by media type + `mediaStatus` |
| `nbu_tape_robot_device_hosts` | Gauge | — | `GET /storage/robots-device-hosts` | Count of device hosts with robots configured |
```

- [ ] **Step 2:** In `docs/config-examples/README.md`, add a row/sentence noting `collectors.tape` enables drive/tape/robot metrics (opt-in, 10.0+ for drives).

- [ ] **Step 3:** In `CHANGELOG.md` `[Unreleased]` → Added:

```markdown
- **Tape / drive metrics** (opt-in `collectors.tape`, default off): `nbu_tape_drives_count`,
  `nbu_tape_drive_info`, `nbu_tape_media_count`, `nbu_tape_robot_device_hosts` over REST
  (`/storage/drives` on NBU 10.0+; `/storage/tape-media` + `/storage/robots-device-hosts` on 10.5+),
  with per-endpoint graceful degradation and the `site` label. See the Feature 2 design spec.
```

- [ ] **Step 4:** `make ci` green (`rtk proxy make ci`).
- [ ] **Step 5: Commit**

```bash
git add docs/metrics.md docs/config-examples/README.md CHANGELOG.md
git commit -m "docs: document opt-in tape/drive metrics collector"
```

---

## Final

- [ ] Whole-implementation review (spec coverage: drives count+info ✓, media count w/ pagination ✓, robot-device-host count ✓, opt-in toggle ✓, per-endpoint graceful degradation ✓, `site` label ✓, version degradation ✓).
- [ ] `make ci` green; quick manual sanity: enable `collectors.tape` against a mock returning the three fixtures → `/metrics` shows the four `nbu_tape_*` metrics with `site`.
- [ ] superpowers:finishing-a-development-branch.

## Acceptance criteria (from spec)

- Opt-in `tape` collector, default off; drives report on NBU 10.0+, tape-media/robots on 10.5+, with a missing endpoint logged+skipped (never flips `nbu_up`).
- `nbu_tape_drives_count{state,drive_type,robot_type}`, `nbu_tape_drive_info` per drive, `nbu_tape_media_count{media_type,status}` (offset-paginated aggregation), `nbu_tape_robot_device_hosts` — all `site`-labelled.
- No CLI shell-out; REST only. Bounded label cardinality regardless of fleet size.
