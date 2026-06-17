package exporter

import (
	"context"
	"encoding/json"
	"errors"
	neturl "net/url"
	"os"
	"strings"
	"testing"

	"github.com/fjacquet/nbu_exporter/internal/models"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

// tapeRoutedClient is a NetBackupClient mock. byPath serves a fixture per endpoint
// path substring ("" => return an error, to exercise graceful degradation). byOffset
// serves paginated fixtures keyed by path substring then page[offset] value; an offset
// not in the map returns an empty page (which ends the collector's pagination loop).
type tapeRoutedClient struct {
	t        *testing.T
	byPath   map[string]string
	byOffset map[string]map[string]string
}

func (c *tapeRoutedClient) FetchData(_ context.Context, rawURL string, target interface{}) error {
	c.t.Helper()
	for sub, offsets := range c.byOffset {
		if strings.Contains(rawURL, sub) {
			u, err := neturl.Parse(rawURL)
			require.NoError(c.t, err)
			off := u.Query().Get("page[offset]")
			if off == "" {
				off = "0"
			}
			fixture, ok := offsets[off]
			if !ok { // past the last page -> empty result
				return json.Unmarshal([]byte(`{"data":[]}`), target)
			}
			if fixture == "" { // simulate a fetch error at this offset (mid-pagination)
				return errors.New("endpoint unavailable")
			}
			data, err := os.ReadFile(fixture)
			require.NoError(c.t, err)
			return json.Unmarshal(data, target)
		}
	}
	for sub, fixture := range c.byPath {
		if strings.Contains(rawURL, sub) {
			if fixture == "" {
				return errors.New("endpoint unavailable")
			}
			data, err := os.ReadFile(fixture)
			require.NoError(c.t, err)
			return json.Unmarshal(data, target)
		}
	}
	c.t.Fatalf("unexpected URL: %s", rawURL)
	return nil
}
func (c *tapeRoutedClient) DetectAPIVersion(context.Context) (string, error) {
	return models.APIVersion140, nil
}
func (c *tapeRoutedClient) Close() error { return nil }

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
	infoCount := 0
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
			infoCount++
		}
	}
	require.Equal(t, float64(2), driveCounts["UP|DT_HCART|TLD"])
	require.Equal(t, float64(1), driveCounts["DOWN|DT_HCART|TLD"])
	require.Equal(t, 3, infoCount, "one nbu_tape_drive_info per drive")
}

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

// All endpoints fail -> Collect returns nil and emits nothing (graceful degradation).
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

// The tape collector is built only when collectors.tape is enabled.
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

// TestTapeCollector_MediaPartialOnError verifies that a mid-pagination fetch error
// still emits the counts accumulated from earlier pages (degraded-but-partial).
func TestTapeCollector_MediaPartialOnError(t *testing.T) {
	client := &tapeRoutedClient{
		t: t,
		byPath: map[string]string{
			"/storage/drives":              "",
			"/storage/robots-device-hosts": "",
		},
		byOffset: map[string]map[string]string{
			"/storage/tape-media": {
				"0": "../../testdata/api-versions/tape-media-page1.json",
				"2": "", // page 2 errors mid-pagination
			},
		},
	}
	c := newTapeCollector(client, testConfig(), "site1")
	ch := make(chan prometheus.Metric, 32)
	require.NoError(t, c.Collect(context.Background(), ch))
	close(ch)

	media := map[string]float64{}
	for m := range ch {
		var d dto.Metric
		require.NoError(t, m.Write(&d))
		if strings.Contains(m.Desc().String(), "nbu_tape_media_count") {
			media[labelValue(&d, "media_type")+"|"+labelValue(&d, "status")] = d.GetGauge().GetValue()
		}
	}
	require.Equal(t, float64(2), media["HCART|ACTIVE"], "page-1 counts emitted despite the page-2 error")
}
