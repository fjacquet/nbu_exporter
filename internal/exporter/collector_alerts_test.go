package exporter

import (
	"context"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

func TestAlertsCollector(t *testing.T) {
	client := newMockClientFromFixture(t, "../../testdata/api-versions/alerts-response.json")
	c := newAlertsCollector(client, testConfig())
	ch := make(chan prometheus.Metric, 16)
	require.NoError(t, c.Collect(context.Background(), ch))
	close(ch)

	counts := map[string]float64{}
	for m := range ch {
		var d dto.Metric
		require.NoError(t, m.Write(&d))
		key := labelValue(&d, "severity") + "/" + labelValue(&d, "category")
		counts[key] = d.GetGauge().GetValue()
	}
	require.Equal(t, float64(2), counts["ERROR/JOB"])
	require.Equal(t, float64(1), counts["WARNING/JOB"])
}
