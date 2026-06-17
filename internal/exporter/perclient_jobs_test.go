package exporter

import (
	"strings"
	"testing"
	"time"

	"github.com/fjacquet/nbu_exporter/internal/models"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

// collectPerClientLabels runs the job-aggregate emission for one site and returns
// "<metric>|site|client|..." keys for the two per-client lifecycle metrics.
func collectPerClientLabels(t *testing.T, c *NbuCollector, site string, agg *JobAggregator) []string {
	t.Helper()
	ch := make(chan prometheus.Metric, 64)
	c.exposeJobAggregateMetrics(ch, site, agg)
	close(ch)
	var out []string
	for m := range ch {
		var d dto.Metric
		require.NoError(t, m.Write(&d))
		desc := m.Desc().String()
		switch {
		case strings.Contains(desc, "nbu_client_jobs_count"):
			out = append(out, "nbu_client_jobs_count|"+labelValue(&d, "site")+"|"+labelValue(&d, "client")+"|"+labelValue(&d, "action")+"|"+labelValue(&d, "status"))
		case strings.Contains(desc, "nbu_client_last_job_success_seconds"):
			out = append(out, "nbu_client_last_job_success_seconds|"+labelValue(&d, "site")+"|"+labelValue(&d, "client")+"|"+labelValue(&d, "policy")+"|"+labelValue(&d, "action"))
		}
	}
	return out
}

// TestExposePerClient_PerSiteCacheIsolation verifies the per-client metrics are
// emitted with the correct site label, and that the persistent last-success cache
// is keyed by site so one site's cached client never leaks under another site.
func TestExposePerClient_PerSiteCacheIsolation(t *testing.T) {
	cfg := testConfig()
	cfg.Collectors.PerClient.Enabled = true
	cfg.Collectors.PerClient.Allowlist = []string{"db01"}
	c := newBaseCollector(cfg, nil)

	aggA := NewJobAggregator()
	aggA.EnablePerClient(true, []string{"db01"})
	aggregateJob(aggA, models.JobAttributes{ClientName: "db01", JobType: "BACKUP", Status: 0, PolicyName: "P1", EndTime: time.Unix(1000, 0)})

	gotA := collectPerClientLabels(t, c, "siteA", aggA)
	require.Contains(t, gotA, "nbu_client_jobs_count|siteA|db01|BACKUP|0")
	require.Contains(t, gotA, "nbu_client_last_job_success_seconds|siteA|db01|P1|BACKUP")

	// Site B sees no jobs this cycle; site A's cached db01 must not emit under site B.
	aggB := NewJobAggregator()
	aggB.EnablePerClient(true, []string{"db01"})
	gotB := collectPerClientLabels(t, c, "siteB", aggB)
	for _, k := range gotB {
		require.NotContains(t, k, "|siteB|db01|", "site A's cached client must not leak under site B")
	}
}

// TestJobAggregator_PerClientGating verifies the per-client lifecycle maps are
// populated only when the opt-in is enabled and only for allowlisted clients, so
// cardinality (and the persistent last-success cache) stays bounded by the allowlist.
func TestJobAggregator_PerClientGating(t *testing.T) {
	// Disabled (default): no per-client tracking at all.
	agg := NewJobAggregator()
	require.False(t, agg.trackClient("db01"))
	aggregateJob(agg, models.JobAttributes{ClientName: "db01", JobType: "BACKUP", Status: 0})
	require.Empty(t, agg.ClientCount, "disabled per-client => no client series")
	require.Empty(t, agg.ClientLastSuccess)

	// Enabled with an allowlist: only allowlisted clients are folded in.
	agg2 := NewJobAggregator()
	agg2.EnablePerClient(true, []string{"db01"})
	require.True(t, agg2.trackClient("db01"))
	require.False(t, agg2.trackClient("app02"))

	aggregateJob(agg2, models.JobAttributes{ClientName: "db01", JobType: "BACKUP", Status: 0, PolicyName: "P1", EndTime: time.Unix(1000, 0)})
	aggregateJob(agg2, models.JobAttributes{ClientName: "app02", JobType: "BACKUP", Status: 0, PolicyName: "P1", EndTime: time.Unix(2000, 0)})

	require.Equal(t, float64(1), agg2.ClientCount[ClientJobKey{Client: "db01", Action: "BACKUP", Status: "0"}])
	_, hasApp := agg2.ClientCount[ClientJobKey{Client: "app02", Action: "BACKUP", Status: "0"}]
	require.False(t, hasApp, "non-allowlisted client must not be folded in")
	require.Equal(t, float64(1000), agg2.ClientLastSuccess[ClientSuccessKey{Client: "db01", Policy: "P1", Action: "BACKUP"}])
}
