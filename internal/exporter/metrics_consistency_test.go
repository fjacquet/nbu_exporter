// Package exporter provides metrics consistency tests for NetBackup API version support.
// These tests verify that metric names, labels, and values remain consistent across
// different API versions (3.0, 12.0, 13.0).
package exporter

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMetricsConsistency_JobMetrics verifies that job metrics have consistent
// names and labels across all API versions.
func TestMetricsConsistencyJobMetrics(t *testing.T) {
	versions := []string{"3.0", "12.0", "13.0"}

	for _, version := range versions {
		t.Run("Version_"+strings.ReplaceAll(version, ".", "_"), func(t *testing.T) {
			// Create mock server with job data
			server := createMockServerWithJobs(version)
			defer server.Close()

			// Create collector
			serverAddr := strings.TrimPrefix(server.URL, testSchemeHTTPS)
			cfg := createTestConfig(serverAddr, version)
			cfg.NbuServer.Scheme = "https"

			collector, err := NewNbuCollector(cfg)
			require.NoError(t, err, "Collector creation should succeed for version %s", version)

			// Collect metrics
			registry := prometheus.NewRegistry()
			registry.MustRegister(collector)

			// Verify metric names exist
			metricFamilies, err := registry.Gather()
			require.NoError(t, err, "Gathering metrics should succeed")

			// Check for expected metric names
			expectedMetrics := []string{
				"nbu_jobs_bytes",
				"nbu_jobs_count",
				"nbu_status_count",
				"nbu_api_version",
			}

			foundMetrics := make(map[string]bool)
			for _, mf := range metricFamilies {
				foundMetrics[mf.GetName()] = true
			}

			for _, expected := range expectedMetrics {
				assert.True(t, foundMetrics[expected],
					"Metric %s should exist for API version %s", expected, version)
			}

			// Verify API version metric
			apiVersionMetric := findMetricFamily(metricFamilies, "nbu_api_version")
			require.NotNil(t, apiVersionMetric, "API version metric should exist")
			assert.Equal(t, 1, len(apiVersionMetric.GetMetric()),
				"API version metric should have one value")
			assert.Equal(t, version, apiVersionMetric.GetMetric()[0].GetLabel()[0].GetValue(),
				"API version label should match configured version")
		})
	}
}

// TestMetricsConsistency_StorageMetrics verifies that storage metrics have consistent
// names and labels across all API versions.
func TestMetricsConsistencyStorageMetrics(t *testing.T) {
	versions := []string{"3.0", "12.0", "13.0"}

	for _, version := range versions {
		t.Run("Version_"+strings.ReplaceAll(version, ".", "_"), func(t *testing.T) {
			// Create mock server with storage data
			server := createMockServerWithStorage(version)
			defer server.Close()

			// Create collector
			serverAddr := strings.TrimPrefix(server.URL, testSchemeHTTPS)
			cfg := createTestConfig(serverAddr, version)
			cfg.NbuServer.Scheme = "https"

			collector, err := NewNbuCollector(cfg)
			require.NoError(t, err, "Collector creation should succeed for version %s", version)

			// Collect metrics
			registry := prometheus.NewRegistry()
			registry.MustRegister(collector)

			// Verify metric names exist
			metricFamilies, err := registry.Gather()
			require.NoError(t, err, "Gathering metrics should succeed")

			// Check for storage metric
			storageMetric := findMetricFamily(metricFamilies, "nbu_disk_bytes")
			require.NotNil(t, storageMetric, "Storage metric should exist for version %s", version)

			// Verify labels
			if len(storageMetric.GetMetric()) > 0 {
				labels := storageMetric.GetMetric()[0].GetLabel()
				labelNames := make([]string, len(labels))
				for i, label := range labels {
					labelNames[i] = label.GetName()
				}

				// Expected labels for storage metrics
				expectedLabels := []string{"name", "type", "size"}
				for _, expected := range expectedLabels {
					assert.Contains(t, labelNames, expected,
						"Storage metric should have label %s for version %s", expected, version)
				}
			}
		})
	}
}

// TestMetricsConsistency_LabelValues verifies that label values remain consistent
// across API versions for the same data.
func TestMetricsConsistencyLabelValues(t *testing.T) {
	jobData := createTestJobData()
	versions := []string{"3.0", "12.0", "13.0"}

	collectedLabels := collectLabelsForAllVersions(t, jobData, versions)
	verifyLabelConsistency(t, collectedLabels)
}

// collectLabelsForAllVersions collects metric labels for all API versions
func collectLabelsForAllVersions(t *testing.T, jobData interface{}, versions []string) map[string]map[string]bool {
	t.Helper()
	collectedLabels := make(map[string]map[string]bool)

	for _, version := range versions {
		server := createMockServerWithJobData(jobData)
		defer server.Close()

		metricFamilies := collectMetricsForVersion(t, server.URL, version)
		collectedLabels[version] = extractJobLabels(metricFamilies)
	}

	return collectedLabels
}

// createMockServerWithJobData creates a mock server that returns the given job data
func createMockServerWithJobData(jobData interface{}) *httptest.Server {
	return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(contentTypeHeader, contentTypeJSON)
		_ = json.NewEncoder(w).Encode(jobData)
	}))
}

// collectMetricsForVersion collects metrics for a specific API version
func collectMetricsForVersion(t *testing.T, serverURL, version string) []*dto.MetricFamily {
	t.Helper()

	serverAddr := strings.TrimPrefix(serverURL, testSchemeHTTPS)
	cfg := createTestConfig(serverAddr, version)
	cfg.NbuServer.Scheme = "https"

	collector, err := NewNbuCollector(cfg)
	require.NoError(t, err)

	registry := prometheus.NewRegistry()
	registry.MustRegister(collector)

	metricFamilies, err := registry.Gather()
	require.NoError(t, err)

	return metricFamilies
}

// extractJobLabels extracts label combinations from job metrics
func extractJobLabels(metricFamilies []*dto.MetricFamily) map[string]bool {
	labels := make(map[string]bool)
	jobsMetric := findMetricFamily(metricFamilies, "nbu_jobs_count")

	if jobsMetric != nil {
		for _, metric := range jobsMetric.GetMetric() {
			labelStr := buildLabelString(metric.GetLabel())
			labels[labelStr] = true
		}
	}

	return labels
}

// buildLabelString builds a string representation of metric labels
func buildLabelString(labels []*dto.LabelPair) string {
	labelStr := ""
	for _, label := range labels {
		labelStr += label.GetName() + "=" + label.GetValue() + ","
	}
	return labelStr
}

// verifyLabelConsistency verifies that all versions have the same label combinations
func verifyLabelConsistency(t *testing.T, collectedLabels map[string]map[string]bool) {
	t.Helper()

	if len(collectedLabels) <= 1 {
		return
	}

	var referenceLabels map[string]bool
	var referenceVersion string

	for version, labels := range collectedLabels {
		if referenceLabels == nil {
			referenceLabels = labels
			referenceVersion = version
		} else {
			assert.Equal(t, referenceLabels, labels,
				"Label combinations should be identical between version %s and %s",
				referenceVersion, version)
		}
	}
}

// TestMetricsConsistency_MetricTypes verifies that metric types (gauge, counter)
// remain consistent across API versions.
func TestMetricsConsistencyMetricTypes(t *testing.T) {
	versions := []string{"3.0", "12.0", "13.0"}
	metricTypes := make(map[string]map[string]string) // version -> metric name -> type

	for _, version := range versions {
		// Create mock server
		server := createMockServerWithJobs(version)
		defer server.Close()

		// Create collector
		serverAddr := strings.TrimPrefix(server.URL, testSchemeHTTPS)
		cfg := createTestConfig(serverAddr, version)
		cfg.NbuServer.Scheme = "https"

		collector, err := NewNbuCollector(cfg)
		require.NoError(t, err)

		// Collect metrics
		registry := prometheus.NewRegistry()
		registry.MustRegister(collector)

		metricFamilies, err := registry.Gather()
		require.NoError(t, err)

		// Extract metric types
		metricTypes[version] = make(map[string]string)
		for _, mf := range metricFamilies {
			metricTypes[version][mf.GetName()] = mf.GetType().String()
		}
	}

	// Verify all versions have the same metric types
	var referenceTypes map[string]string
	var referenceVersion string
	for version, types := range metricTypes {
		if referenceTypes == nil {
			referenceTypes = types
			referenceVersion = version
		} else {
			for metricName, metricType := range referenceTypes {
				assert.Equal(t, metricType, types[metricName],
					"Metric %s should have type %s in both version %s and %s",
					metricName, metricType, referenceVersion, version)
			}
		}
	}
}

// TestMetricsConsistency_GrafanaDashboard verifies that the Grafana dashboard
// queries work with metrics from all API versions.
func TestMetricsConsistencyGrafanaDashboard(t *testing.T) {
	// Read the Grafana dashboard JSON
	dashboardPath := "../../grafana/NBU Statistics-1629904585394.json"
	if _, err := os.Stat(dashboardPath); os.IsNotExist(err) {
		t.Skip("Grafana dashboard file not found, skipping test")
		return
	}

	dashboardData, err := os.ReadFile(dashboardPath)
	require.NoError(t, err, "Should be able to read Grafana dashboard")

	// Parse dashboard JSON
	var dashboard map[string]interface{}
	err = json.Unmarshal(dashboardData, &dashboard)
	require.NoError(t, err, "Should be able to parse Grafana dashboard JSON")

	// Extract metric names from dashboard queries
	// This is a simplified check - in reality, you'd parse the dashboard structure
	dashboardStr := string(dashboardData)

	// Check that expected metric names are referenced in the dashboard
	expectedMetrics := []string{
		"nbu_jobs_bytes",
		"nbu_jobs_count",
		"nbu_disk_bytes",
		"nbu_status_count",
	}

	for _, metric := range expectedMetrics {
		assert.Contains(t, dashboardStr, metric,
			"Grafana dashboard should reference metric %s", metric)
	}

	// Verify that the dashboard doesn't reference version-specific metrics
	// that wouldn't exist in all versions
	versionSpecificMetrics := []string{
		"nbu_jobs_v3",
		"nbu_jobs_v12",
		"nbu_jobs_v13",
	}

	for _, metric := range versionSpecificMetrics {
		assert.NotContains(t, dashboardStr, metric,
			"Grafana dashboard should not reference version-specific metric %s", metric)
	}
}

// TestMetricsConsistency_PrometheusExport verifies that metrics can be exported
// in Prometheus format consistently across versions.
func TestMetricsConsistencyPrometheusExport(t *testing.T) {
	versions := []string{"3.0", "12.0", "13.0"}

	for _, version := range versions {
		t.Run("Version_"+strings.ReplaceAll(version, ".", "_"), func(t *testing.T) {
			// Create mock server
			server := createMockServerWithJobs(version)
			defer server.Close()

			// Create collector
			serverAddr := strings.TrimPrefix(server.URL, testSchemeHTTPS)
			cfg := createTestConfig(serverAddr, version)
			cfg.NbuServer.Scheme = "https"

			collector, err := NewNbuCollector(cfg)
			require.NoError(t, err)

			// Export metrics in Prometheus format
			registry := prometheus.NewRegistry()
			registry.MustRegister(collector)

			// Verify metrics can be collected without errors
			count := testutil.CollectAndCount(collector)
			assert.Greater(t, count, 0, "Should collect at least one metric for version %s", version)

			// Verify no duplicate metrics
			metricFamilies, err := registry.Gather()
			require.NoError(t, err)

			metricNames := make(map[string]int)
			for _, mf := range metricFamilies {
				metricNames[mf.GetName()]++
			}

			for name, count := range metricNames {
				assert.Equal(t, 1, count,
					"Metric %s should appear exactly once for version %s", name, version)
			}
		})
	}
}

// Helper function to create a mock server with job data
func createMockServerWithJobs(version string) *httptest.Server {
	return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return appropriate response based on endpoint
		if strings.Contains(r.URL.Path, "/admin/jobs") {
			response := createTestJobData()
			w.Header().Set(contentTypeHeader, contentTypeJSON)
			_ = json.NewEncoder(w).Encode(response)
		} else if strings.Contains(r.URL.Path, "/storage/storage-units") {
			response := createTestStorageData()
			w.Header().Set(contentTypeHeader, contentTypeJSON)
			_ = json.NewEncoder(w).Encode(response)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

// Helper function to create a mock server with storage data
func createMockServerWithStorage(version string) *httptest.Server {
	return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/storage/storage-units") {
			response := createTestStorageData()
			w.Header().Set(contentTypeHeader, contentTypeJSON)
			_ = json.NewEncoder(w).Encode(response)
		} else if strings.Contains(r.URL.Path, "/admin/jobs") {
			// Return empty jobs response
			response := map[string]interface{}{
				"data": []interface{}{},
			}
			w.Header().Set(contentTypeHeader, contentTypeJSON)
			_ = json.NewEncoder(w).Encode(response)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

// Helper function to create test job data
func createTestJobData() map[string]interface{} {
	return map[string]interface{}{
		"data": []map[string]interface{}{
			{
				"type": "job",
				"id":   "1",
				"attributes": map[string]interface{}{
					"jobId":                1,
					"jobType":              "BACKUP",
					"policyType":           "Standard",
					"status":               0, // 0 = success/done
					"state":                "DONE",
					"kilobytesTransferred": 1024000,
					"startTime":            time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
					"endTime":              time.Now().Format(time.RFC3339),
				},
			},
		},
		"meta": map[string]interface{}{
			"pagination": map[string]interface{}{
				"count": 1,
			},
		},
	}
}

// Helper function to create test storage data
func createTestStorageData() map[string]interface{} {
	return map[string]interface{}{
		"data": []map[string]interface{}{
			{
				"type": "storageUnit",
				"id":   "disk-pool-1",
				"attributes": map[string]interface{}{
					"name":               "disk-pool-1",
					"storageServerType":  "DISK",
					"totalCapacityBytes": 1000000000000,
					"usedCapacityBytes":  500000000000,
					"freeCapacityBytes":  500000000000,
				},
			},
		},
	}
}

// Helper function to find a metric family by name
func findMetricFamily(families []*dto.MetricFamily, name string) *dto.MetricFamily {
	for _, mf := range families {
		if mf.GetName() == name {
			return mf
		}
	}
	return nil
}
