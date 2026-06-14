package exporter

import (
	"context"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

func TestSLOCollector(t *testing.T) {
	client := newMockClientFromFixture(t, "../../testdata/api-versions/slo-response.json")
	c := newSLOCollector(client, testConfig())
	ch := make(chan prometheus.Metric, 16)
	require.NoError(t, c.Collect(context.Background(), ch))
	close(ch)

	var total float64
	emitted := 0
	for m := range ch {
		var d dto.Metric
		require.NoError(t, m.Write(&d))
		require.Empty(t, d.GetLabel(), "nbu_slo_count must be unlabeled")
		total = d.GetGauge().GetValue()
		emitted++
	}
	require.Equal(t, 1, emitted, "exactly one nbu_slo_count series expected")
	require.Equal(t, float64(3), total)
}

func TestSLOCollectorError(t *testing.T) {
	client := &errClient{}
	c := newSLOCollector(client, testConfig())
	ch := make(chan prometheus.Metric, 4)
	require.Error(t, c.Collect(context.Background(), ch))
	close(ch)
	require.Empty(t, ch)
}
