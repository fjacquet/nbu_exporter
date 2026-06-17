package exporter

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/fjacquet/nbu_exporter/internal/models"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

// tapeRoutedClient is a NetBackupClient mock that serves a fixture per endpoint
// path. A path mapped to "" yields an error (to exercise graceful degradation).
type tapeRoutedClient struct {
	t      *testing.T
	byPath map[string]string // path substring -> fixture file ("" => return error)
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
