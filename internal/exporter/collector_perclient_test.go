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

	require.Len(t, mock.urls, 1)
	u, err := neturl.Parse(mock.urls[0])
	require.NoError(t, err)
	require.Equal(t, "-endTime", u.Query().Get("sort"))
	require.Equal(t, "1", u.Query().Get("page[limit]"))
	filter := u.Query().Get("filter")
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
