package exporter

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/fjacquet/nbu_exporter/internal/models"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

// splitTestServerURL splits an "http://host:port" httptest server URL into its
// host and port. Shared by the test config builders.
func splitTestServerURL(serverURL string) (host, port string) {
	parts := strings.SplitN(strings.TrimPrefix(serverURL, "http://"), ":", 2)
	host = parts[0]
	if len(parts) == 2 {
		port = parts[1]
	}
	return host, port
}

// fixtureClient is a minimal NetBackupClient mock whose FetchData reads a JSON
// fixture from disk and unmarshals it into the supplied target. It is shared by
// the opt-in sub-collector tests (alerts, malware, catalog, SLO).
type fixtureClient struct {
	t    *testing.T
	path string
}

// newMockClientFromFixture returns a NetBackupClient whose FetchData serves the
// given fixture file. The path is resolved relative to the test package
// directory (internal/exporter), so callers pass e.g.
// "../../testdata/api-versions/alerts-response.json".
func newMockClientFromFixture(t *testing.T, path string) NetBackupClient {
	t.Helper()
	return &fixtureClient{t: t, path: path}
}

func (f *fixtureClient) FetchData(_ context.Context, _ string, target interface{}) error {
	f.t.Helper()
	data, err := os.ReadFile(f.path)
	require.NoError(f.t, err)
	return json.Unmarshal(data, target)
}

func (f *fixtureClient) DetectAPIVersion(context.Context) (string, error) {
	return models.APIVersion140, nil
}

func (f *fixtureClient) Close() error { return nil }

// errClient is a NetBackupClient mock whose FetchData always fails, used to
// exercise sub-collector graceful-degradation/error paths.
type errClient struct{}

func (errClient) FetchData(context.Context, string, interface{}) error {
	return errors.New("fetch failed")
}
func (errClient) DetectAPIVersion(context.Context) (string, error) {
	return models.APIVersion140, nil
}
func (errClient) Close() error { return nil }

// testConfig returns a minimal valid config with a base URL set, enough for
// sub-collectors to build request URLs.
func testConfig() models.Config {
	var cfg models.Config
	cfg.NbuServer.Scheme = "https"
	cfg.NbuServer.Host = "nbu.example.com"
	cfg.NbuServer.Port = "1556"
	cfg.NbuServer.URI = "/netbackup"
	cfg.NbuServer.APIVersion = models.APIVersion140
	return cfg
}

// labelValue returns the value of the named label on a written metric, or "".
func labelValue(d *dto.Metric, name string) string {
	for _, l := range d.GetLabel() {
		if l.GetName() == name {
			return l.GetValue()
		}
	}
	return ""
}
