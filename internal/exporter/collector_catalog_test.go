package exporter

import (
	"context"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

func TestCatalogCollector(t *testing.T) {
	client := newMockClientFromFixture(t, "../../testdata/api-versions/catalog-count-response.json")
	c := newCatalogCollector(client, testConfig())
	ch := make(chan prometheus.Metric, 64)
	require.NoError(t, c.Collect(context.Background(), ch))
	close(ch)

	emitted := 0
	for m := range ch {
		var d dto.Metric
		require.NoError(t, m.Write(&d))
		require.True(t, strings.Contains(m.Desc().String(), "nbu_catalog_images_count"))
		require.Equal(t, float64(42), d.GetGauge().GetValue())
		emitted++
	}
	// One series per curated (malware_status, anomaly_status) combination.
	require.Equal(t, len(catalogMalwareStatuses)*len(catalogAnomalyStatuses), emitted)
}

func TestCatalogCollectorError(t *testing.T) {
	// After per-combination graceful degradation, every sub-call failing leaves
	// Collect returning nil while emitting no metrics.
	client := &errClient{}
	c := newCatalogCollector(client, testConfig())
	ch := make(chan prometheus.Metric, 64)
	require.NoError(t, c.Collect(context.Background(), ch))
	close(ch)
	require.Empty(t, ch)
}
