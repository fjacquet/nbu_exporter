// Package exporter provides tests for the additional NetBackup metrics
// (storage attributes and extended job aggregates).
package exporter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fjacquet/nbu_exporter/internal/models"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDurationAccum_Observe(t *testing.T) {
	acc := newDurationAccum()
	// 30s, 120s, 5000s -> falls into buckets <=60, <=300, <=7200 respectively.
	acc.Observe(30)
	acc.Observe(120)
	acc.Observe(5000)

	assert.Equal(t, uint64(3), acc.Count, "all observations counted")
	assert.InDelta(t, 5150.0, acc.Sum, 0.001, "sum of durations")

	buckets := acc.Buckets()
	assert.Equal(t, uint64(1), buckets[60], "only the 30s obs is <= 60")
	assert.Equal(t, uint64(2), buckets[300], "30s and 120s are <= 300")
	assert.Equal(t, uint64(2), buckets[3600], "still 2 at <= 3600")
	assert.Equal(t, uint64(3), buckets[7200], "all three <= 7200")
	assert.Equal(t, uint64(3), buckets[86400], "all three <= 86400")
}

func TestStorageUnitInfo_InfoLabels(t *testing.T) {
	u := StorageUnitInfo{
		Name:               "pool1",
		Type:               "MEDIA_SERVER",
		SubType:            "PureDisk",
		IsCloud:            false,
		WormCapable:        true,
		UseWorm:            false,
		ReplicationCapable: true,
		InstantAccess:      true,
	}
	got := u.InfoLabels()
	want := []string{"pool1", "MEDIA_SERVER", "PureDisk", "false", "true", "false", "true", "true"}
	assert.Equal(t, want, got)
}

func TestAggregateJob_PopulatesAllAggregates(t *testing.T) {
	agg := NewJobAggregator()
	end := time.Now().UTC()
	start := end.Add(-10 * time.Minute) // 600s duration

	// Two completed BACKUP/STANDARD jobs with dedup ratios 2.5 and 1.5 (mean 2.0).
	aggregateJob(agg, models.JobAttributes{
		JobType: "BACKUP", PolicyType: "STANDARD", Status: 0, State: "DONE",
		NumberOfFiles: 100, KilobytesTransferred: 1024, DedupRatio: 2.5,
		StartTime: start, EndTime: end,
	})
	aggregateJob(agg, models.JobAttributes{
		JobType: "BACKUP", PolicyType: "STANDARD", Status: 0, State: "DONE",
		NumberOfFiles: 50, KilobytesTransferred: 512, DedupRatio: 1.5,
		StartTime: start, EndTime: end,
	})
	// One QUEUED job (no end time -> excluded from duration).
	aggregateJob(agg, models.JobAttributes{
		JobType: "BACKUP", PolicyType: "STANDARD", Status: 0, State: jobStateQueued,
		JobQueueReason: 7,
	})

	policyKey := JobPolicyKey{Action: "BACKUP", PolicyType: "STANDARD"}

	assert.Equal(t, 150.0, agg.FilesCount[policyKey], "files summed across jobs")
	assert.Equal(t, 4.0, agg.DedupSum[policyKey], "dedup ratios summed (2.5 + 1.5 + 0)")
	assert.Equal(t, 3.0, agg.DedupCount[policyKey], "all three jobs counted for dedup mean")

	assert.Equal(t, 2.0, agg.StateCount[JobStateKey{Action: "BACKUP", State: "DONE"}])
	assert.Equal(t, 1.0, agg.StateCount[JobStateKey{Action: "BACKUP", State: jobStateQueued}])
	assert.Equal(t, 1.0, agg.QueuedCount[JobQueueKey{Action: "BACKUP", Reason: "7"}])

	require.NotNil(t, agg.Duration[policyKey], "duration histogram created")
	assert.Equal(t, uint64(2), agg.Duration[policyKey].Count, "only completed jobs in histogram")
	assert.InDelta(t, 1200.0, agg.Duration[policyKey].Sum, 0.5, "two 600s jobs")
}

func TestAggregateJob_EmptyStateBecomesUnknown(t *testing.T) {
	agg := NewJobAggregator()
	aggregateJob(agg, models.JobAttributes{JobType: "BACKUP", PolicyType: "STANDARD", State: ""})
	assert.Equal(t, 1.0, agg.StateCount[JobStateKey{Action: "BACKUP", State: "UNKNOWN"}])
}

// storageJSONWithAttributes is a storage API payload with one rich DISK unit and
// one Tape unit (which must be excluded from metrics). Built as JSON so the test
// does not depend on the exact shape of the anonymous struct in models.Storages.
const storageJSONWithAttributes = `{
  "data": [
    {
      "type": "storageUnit",
      "id": "disk-pool-1",
      "attributes": {
        "name": "disk-pool-1",
        "storageType": "DISK",
        "storageSubType": "PureDisk",
        "storageServerType": "MEDIA_SERVER",
        "freeCapacityBytes": 400,
        "usedCapacityBytes": 600,
        "totalCapacityBytes": 1000,
        "maxConcurrentJobs": 8,
        "maxFragmentSizeMegabytes": 512,
        "wormCapable": true,
        "instantAccessEnabled": true
      }
    },
    {
      "type": "storageUnit",
      "id": "tape-1",
      "attributes": {
        "name": "tape-1",
        "storageType": "Tape",
        "storageServerType": "MEDIA_SERVER"
      }
    }
  ]
}`

// storageResponseWithAttributes decodes storageJSONWithAttributes into the model.
func storageResponseWithAttributes(t *testing.T) *models.Storages {
	t.Helper()
	var s models.Storages
	require.NoError(t, json.Unmarshal([]byte(storageJSONWithAttributes), &s))
	return &s
}

func TestFetchStorageFull_ReturnsUnits(t *testing.T) {
	storageResp := storageResponseWithAttributes(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(contentTypeHeader, fmt.Sprintf(contentTypeNetBackupJSONFormat, "12.0"))
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(storageResp)
	}))
	defer server.Close()

	cfg := createTestConfig(server.URL, "12.0")
	client := NewNbuClient(cfg)

	metrics, units, err := FetchStorageFull(context.Background(), client)
	require.NoError(t, err)

	// Tape excluded: only the disk unit yields free+used metrics and one unit info.
	assert.Len(t, metrics, 2, "free + used for the single disk unit")
	require.Len(t, units, 1, "tape unit excluded from unit info")

	u := units[0]
	assert.Equal(t, "disk-pool-1", u.Name)
	assert.Equal(t, 1000.0, u.TotalCapacityBytes)
	assert.Equal(t, 8.0, u.MaxConcurrentJobs)
	assert.Equal(t, float64(512*1024*1024), u.MaxFragmentBytes, "MB converted to bytes")
	assert.True(t, u.WormCapable)
	assert.True(t, u.InstantAccess)
	assert.False(t, u.IsCloud)
}

// combinedServer routes storage and jobs endpoints for a full collector scrape.
func combinedServer(t *testing.T) *httptest.Server {
	t.Helper()
	jobs := &models.Jobs{}
	jobs.Data = make([]models.JobData, 1)
	end := time.Now().UTC()
	jobs.Data[0].Attributes = models.JobAttributes{
		JobType: "BACKUP", PolicyType: "STANDARD", Status: 0, State: "DONE",
		NumberOfFiles: 42, KilobytesTransferred: 2048, DedupRatio: 3.0,
		StartTime: end.Add(-5 * time.Minute), EndTime: end,
	}
	jobs.Meta.Pagination.Offset = 0
	jobs.Meta.Pagination.Last = 0

	storageResp := storageResponseWithAttributes(t)
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(contentTypeHeader, fmt.Sprintf(contentTypeNetBackupJSONFormat, "12.0"))
		w.WriteHeader(http.StatusOK)
		if strings.Contains(r.URL.Path, "/admin/jobs") {
			_ = json.NewEncoder(w).Encode(jobs)
			return
		}
		_ = json.NewEncoder(w).Encode(storageResp)
	}))
}

func TestCollector_ExposesNewMetrics(t *testing.T) {
	server := combinedServer(t)
	defer server.Close()

	cfg := createTestConfig(server.URL, "12.0")
	cfg.Server.ScrapingInterval = "5m"
	collector, err := NewNbuCollector(cfg)
	require.NoError(t, err)
	defer func() { _ = collector.Close() }()

	newNames := []string{
		"nbu_disk_capacity_bytes",
		"nbu_storage_max_concurrent_jobs",
		"nbu_storage_max_fragment_size_bytes",
		"nbu_storage_info",
		"nbu_jobs_state_count",
		"nbu_jobs_files_count",
		"nbu_jobs_dedup_ratio",
		"nbu_job_duration_seconds",
	}
	count := testutil.CollectAndCount(collector, newNames...)
	assert.Positive(t, count, "new metric families should be collected")

	// Spot-check concrete values via a gather.
	registry := prometheus.NewRegistry()
	require.NoError(t, registry.Register(collector))
	families, err := registry.Gather()
	require.NoError(t, err)

	byName := make(map[string]*dto.MetricFamily, len(families))
	for _, f := range families {
		byName[f.GetName()] = f
	}

	require.Contains(t, byName, "nbu_disk_capacity_bytes")
	assert.Equal(t, 1000.0, byName["nbu_disk_capacity_bytes"].Metric[0].GetGauge().GetValue())

	require.Contains(t, byName, "nbu_jobs_files_count")
	assert.Equal(t, 42.0, byName["nbu_jobs_files_count"].Metric[0].GetGauge().GetValue())

	require.Contains(t, byName, "nbu_jobs_dedup_ratio")
	assert.Equal(t, 3.0, byName["nbu_jobs_dedup_ratio"].Metric[0].GetGauge().GetValue())

	require.Contains(t, byName, "nbu_job_duration_seconds")
	h := byName["nbu_job_duration_seconds"].Metric[0].GetHistogram()
	assert.Equal(t, uint64(1), h.GetSampleCount())
	assert.InDelta(t, 300.0, h.GetSampleSum(), 1.0, "one ~300s job")
}
