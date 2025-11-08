// Package exporter provides end-to-end integration tests for the NetBackup exporter.
// These tests verify the complete workflow from client initialization through metric collection
// across all supported API versions.
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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEndToEndWorkflowAPI130 tests the complete workflow with NetBackup 11.0 (API 13.0)
func TestEndToEndWorkflowAPI130(t *testing.T) {
	// Create mock server that responds to API 13.0 requests
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		acceptHeader := r.Header.Get("Accept")

		// Verify API version in Accept header
		if !strings.Contains(acceptHeader, "version=13.0") {
			w.WriteHeader(http.StatusNotAcceptable)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"errorCode":    "UNSUPPORTED_API_VERSION",
				"errorMessage": "API version not supported",
			})
			return
		}

		// Handle different endpoints
		switch {
		case strings.Contains(r.URL.Path, "/admin/jobs"):
			// Return jobs response with sample data
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			jobsResponse := `{
				"data": [{
					"type": "job",
					"id": "1",
					"attributes": {
						"jobId": 1,
						"jobType": "BACKUP",
						"policyType": "VMWARE",
						"status": 0,
						"kilobytesTransferred": 1024000,
						"startTime": "2025-11-08T10:00:00Z",
						"endTime": "2025-11-08T11:00:00Z",
						"clientName": "test-client"
					}
				}],
				"meta": {"pagination": {"first": 0, "last": 0, "limit": 1, "offset": 0, "next": 0}}
			}`
			_, _ = w.Write([]byte(jobsResponse))

		case strings.Contains(r.URL.Path, "/storage/storage-units"):
			// Return storage response with sample data
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			storageResponse := `{
				"data": [{
					"type": "storageUnit",
					"id": "disk-pool-1",
					"attributes": {
						"name": "disk-pool-1",
						"storageType": "Disk",
						"storageServerType": "MEDIA_SERVER",
						"freeCapacityBytes": 5368709120000,
						"usedCapacityBytes": 2684354560000,
						"totalCapacityBytes": 8053063680000
					}
				}],
				"meta": {"pagination": {"first": 0, "last": 0, "limit": 100, "offset": 0}}
			}`
			_, _ = w.Write([]byte(storageResponse))

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create configuration
	cfg := createTestConfigTLS(server.URL, "13.0")

	// Create client
	client := NewNbuClient(cfg)
	require.NotNil(t, client)

	// Test storage metrics collection
	ctx := context.Background()
	storageMetrics := make(map[string]float64)
	err := FetchStorage(ctx, client, storageMetrics)
	require.NoError(t, err)
	assert.NotEmpty(t, storageMetrics, "Storage metrics should not be empty")
	assert.Greater(t, len(storageMetrics), 0, "Should have at least one storage metric")

	// Test job metrics collection
	jobsSize := make(map[string]float64)
	jobsCount := make(map[string]float64)
	jobsStatusCount := make(map[string]float64)
	err = FetchAllJobs(ctx, client, jobsSize, jobsCount, jobsStatusCount, "24h")
	require.NoError(t, err)
	assert.NotEmpty(t, jobsCount, "Job count metrics should not be empty")
	assert.Greater(t, len(jobsCount), 0, "Should have at least one job metric")

	// Create collector and verify metrics
	collector, err := NewNbuCollector(cfg)
	require.NoError(t, err)
	require.NotNil(t, collector)

	// Register collector
	registry := prometheus.NewRegistry()
	err = registry.Register(collector)
	require.NoError(t, err)

	// Collect metrics
	metricCount := testutil.CollectAndCount(collector)
	assert.Greater(t, metricCount, 0, "Should collect at least one metric")

	// Verify API version metric is exposed
	count := testutil.CollectAndCount(collector, "nbu_api_version")
	assert.Greater(t, count, 0, "API version metric should be present")
}

// TestEndToEndWorkflowAPI120 tests the complete workflow with NetBackup 10.5 (API 12.0)
func TestEndToEndWorkflowAPI120(t *testing.T) {
	// Create mock server that responds to API 12.0 requests
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		acceptHeader := r.Header.Get("Accept")

		// Verify API version in Accept header
		if !strings.Contains(acceptHeader, "version=12.0") {
			w.WriteHeader(http.StatusNotAcceptable)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"errorCode":    "UNSUPPORTED_API_VERSION",
				"errorMessage": "API version not supported",
			})
			return
		}

		// Handle different endpoints
		switch {
		case strings.Contains(r.URL.Path, "/admin/jobs"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			jobsResponse := `{
				"data": [{
					"type": "job",
					"id": "2",
					"attributes": {
						"jobId": 2,
						"jobType": "RESTORE",
						"policyType": "STANDARD",
						"status": 1,
						"kilobytesTransferred": 512000,
						"startTime": "2025-11-08T09:00:00Z",
						"endTime": "2025-11-08T10:00:00Z",
						"clientName": "test-client-2"
					}
				}],
				"meta": {"pagination": {"first": 0, "last": 0, "limit": 1, "offset": 0, "next": 0}}
			}`
			_, _ = w.Write([]byte(jobsResponse))

		case strings.Contains(r.URL.Path, "/storage/storage-units"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			storageResponse := `{
				"data": [{
					"type": "storageUnit",
					"id": "disk-pool-2",
					"attributes": {
						"name": "disk-pool-2",
						"storageType": "Disk",
						"storageServerType": "MEDIA_SERVER",
						"freeCapacityBytes": 3221225472000,
						"usedCapacityBytes": 1610612736000,
						"totalCapacityBytes": 4831838208000
					}
				}],
				"meta": {"pagination": {"first": 0, "last": 0}}
			}`
			_, _ = w.Write([]byte(storageResponse))

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create configuration
	cfg := createTestConfigTLS(server.URL, "12.0")

	// Create collector
	collector, err := NewNbuCollector(cfg)
	require.NoError(t, err)
	require.NotNil(t, collector)

	// Verify configured version
	assert.Equal(t, "12.0", collector.cfg.NbuServer.APIVersion)

	// Register and collect metrics
	registry := prometheus.NewRegistry()
	err = registry.Register(collector)
	require.NoError(t, err)

	metricCount := testutil.CollectAndCount(collector)
	assert.Greater(t, metricCount, 0, "Should collect metrics for API 12.0")
}

// TestEndToEndFallbackScenario tests the fallback logic (13.0 → 12.0 → 3.0)
func TestEndToEndFallbackScenario(t *testing.T) {
	// Create mock server that only supports API 3.0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		acceptHeader := r.Header.Get("Accept")

		// Only accept API 3.0
		if strings.Contains(acceptHeader, "version=13.0") || strings.Contains(acceptHeader, "version=12.0") {
			w.WriteHeader(http.StatusNotAcceptable)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"errorCode":    "UNSUPPORTED_API_VERSION",
				"errorMessage": "API version not supported",
			})
			return
		}

		if strings.Contains(acceptHeader, "version=3.0") {
			// Handle API 3.0 requests
			switch {
			case strings.Contains(r.URL.Path, "/admin/jobs"):
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				jobsResponse := `{
					"data": [{
						"type": "job",
						"id": "3",
						"attributes": {
							"jobId": 3,
							"jobType": "BACKUP",
							"policyType": "STANDARD",
							"status": 0,
							"kilobytesTransferred": 2048000,
							"startTime": "2025-11-08T08:00:00Z",
							"endTime": "2025-11-08T09:00:00Z",
							"clientName": "legacy-client"
						}
					}],
					"meta": {"pagination": {"first": 0, "last": 0, "limit": 1, "offset": 0, "next": 0}}
				}`
				_, _ = w.Write([]byte(jobsResponse))

			case strings.Contains(r.URL.Path, "/storage/storage-units"):
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				storageResponse := `{
					"data": [{
						"type": "storageUnit",
						"id": "legacy-disk",
						"attributes": {
							"name": "legacy-disk",
							"storageType": "Disk",
							"storageServerType": "MEDIA_SERVER",
							"freeCapacityBytes": 1073741824000,
							"usedCapacityBytes": 536870912000,
							"totalCapacityBytes": 1610612736000
						}
					}],
					"meta": {"pagination": {"first": 0, "last": 0}}
				}`
				_, _ = w.Write([]byte(storageResponse))

			default:
				w.WriteHeader(http.StatusNotFound)
			}
			return
		}

		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	// Create configuration without explicit API version (triggers auto-detection)
	cfg := createTestConfigTLS(server.URL, "")

	// Create client with version detection
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := NewNbuClientWithVersionDetection(ctx, &cfg)
	require.NoError(t, err)
	require.NotNil(t, client)

	// Verify fallback to API 3.0
	assert.Equal(t, "3.0", cfg.NbuServer.APIVersion, "Should fallback to API 3.0")

	// Verify metrics can be collected with API 3.0
	storageMetrics := make(map[string]float64)
	err = FetchStorage(ctx, client, storageMetrics)
	require.NoError(t, err)
	assert.NotEmpty(t, storageMetrics, "Should collect storage metrics with API 3.0")
	assert.Greater(t, len(storageMetrics), 0, "Should have at least one storage metric")
}

// TestEndToEndErrorScenarios tests error handling and recovery
func TestEndToEndErrorScenarios(t *testing.T) {
	tests := []struct {
		name           string
		serverBehavior func(w http.ResponseWriter, r *http.Request)
		expectError    bool
		errorContains  string
	}{
		{
			name: "Authentication Error",
			serverBehavior: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"errorCode":    "INVALID_CREDENTIALS",
					"errorMessage": "Invalid API key",
				})
			},
			expectError:   true,
			errorContains: "401",
		},
		{
			name: "Server Error with Retry",
			serverBehavior: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"errorCode":    "INTERNAL_ERROR",
					"errorMessage": "Internal server error",
				})
			},
			expectError:   true,
			errorContains: "500",
		},
		{
			name: "Network Timeout",
			serverBehavior: func(w http.ResponseWriter, r *http.Request) {
				// Simulate slow response
				time.Sleep(5 * time.Second)
				w.WriteHeader(http.StatusOK)
			},
			expectError:   true,
			errorContains: "context deadline exceeded",
		},
		{
			name: "Invalid JSON Response",
			serverBehavior: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("invalid json"))
			},
			expectError:   true,
			errorContains: "unmarshal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(tt.serverBehavior))
			defer server.Close()

			cfg := createTestConfigTLS(server.URL, "13.0")
			client := NewNbuClient(cfg)

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			storageMetrics := make(map[string]float64)
			err := FetchStorage(ctx, client, storageMetrics)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestEndToEndMetricsConsistency verifies metrics are consistent across API versions
func TestEndToEndMetricsConsistency(t *testing.T) {
	versions := []string{"3.0", "12.0", "13.0"}

	for _, version := range versions {
		t.Run(fmt.Sprintf("API_%s", strings.ReplaceAll(version, ".", "_")), func(t *testing.T) {
			server := createVersionedMockServer(t, version)
			defer server.Close()

			cfg := createTestConfigTLS(server.URL, version)
			collector, err := NewNbuCollector(cfg)
			require.NoError(t, err)

			// Collect metrics
			registry := prometheus.NewRegistry()
			err = registry.Register(collector)
			require.NoError(t, err)

			// Verify expected metrics exist
			expectedMetrics := []string{
				"nbu_disk_bytes",
				"nbu_jobs_bytes",
				"nbu_jobs_count",
				"nbu_status_count",
				"nbu_api_version",
			}

			for _, metricName := range expectedMetrics {
				count := testutil.CollectAndCount(collector, metricName)
				assert.Greater(t, count, 0, "Metric %s should be present for API %s", metricName, version)
			}
		})
	}
}

// TestEndToEndGracefulDegradation tests that collector continues working even if some endpoints fail
func TestEndToEndGracefulDegradation(t *testing.T) {
	// Create server where storage endpoint fails but jobs endpoint works
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		acceptHeader := r.Header.Get("Accept")
		if !strings.Contains(acceptHeader, "version=13.0") {
			w.WriteHeader(http.StatusNotAcceptable)
			return
		}

		switch {
		case strings.Contains(r.URL.Path, "/storage/storage-units"):
			// Storage endpoint fails
			w.WriteHeader(http.StatusInternalServerError)

		case strings.Contains(r.URL.Path, "/admin/jobs"):
			// Jobs endpoint succeeds with data
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			jobsResponse := `{
				"data": [{
					"type": "job",
					"id": "200",
					"attributes": {
						"jobId": 200,
						"jobType": "BACKUP",
						"policyType": "STANDARD",
						"status": 0,
						"kilobytesTransferred": 512000,
						"startTime": "2025-11-08T10:00:00Z",
						"endTime": "2025-11-08T11:00:00Z",
						"clientName": "degraded-test-client"
					}
				}],
				"meta": {"pagination": {"first": 0, "last": 0, "limit": 1, "offset": 0, "next": 0}}
			}`
			_, _ = w.Write([]byte(jobsResponse))

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	cfg := createTestConfigTLS(server.URL, "13.0")
	collector, err := NewNbuCollector(cfg)
	require.NoError(t, err)

	// Collector should still work and expose job metrics
	registry := prometheus.NewRegistry()
	err = registry.Register(collector)
	require.NoError(t, err)

	// Should still collect job metrics even though storage failed
	jobMetricCount := testutil.CollectAndCount(collector, "nbu_jobs_count")
	assert.Greater(t, jobMetricCount, 0, "Should collect job metrics even when storage fails")
}

// Helper function to create test configuration for TLS servers
func createTestConfigTLS(serverURL, apiVersion string) models.Config {
	// Parse server URL
	parts := strings.Split(strings.TrimPrefix(serverURL, "https://"), ":")
	host := parts[0]
	port := "443"
	if len(parts) > 1 {
		port = parts[1]
	}

	cfg := models.Config{}
	cfg.Server.Host = "localhost"
	cfg.Server.Port = "2112"
	cfg.Server.URI = "/metrics"
	cfg.Server.ScrapingInterval = "24h"
	cfg.Server.LogName = "test.log"

	cfg.NbuServer.Scheme = "https"
	cfg.NbuServer.URI = "/netbackup"
	cfg.NbuServer.Domain = "test.domain"
	cfg.NbuServer.DomainType = "NT"
	cfg.NbuServer.Host = host
	cfg.NbuServer.Port = port
	cfg.NbuServer.APIVersion = apiVersion
	cfg.NbuServer.APIKey = "test-api-key"
	cfg.NbuServer.ContentType = fmt.Sprintf("application/vnd.netbackup+json;version=%s", apiVersion)
	cfg.NbuServer.InsecureSkipVerify = true

	return cfg
}

// Helper function to create a versioned mock server
func createVersionedMockServer(t *testing.T, version string) *httptest.Server {
	return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		acceptHeader := r.Header.Get("Accept")

		if !strings.Contains(acceptHeader, fmt.Sprintf("version=%s", version)) {
			w.WriteHeader(http.StatusNotAcceptable)
			return
		}

		switch {
		case strings.Contains(r.URL.Path, "/admin/jobs"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			jobsResponse := fmt.Sprintf(`{
				"data": [{
					"type": "job",
					"id": "100",
					"attributes": {
						"jobId": 100,
						"jobType": "BACKUP",
						"policyType": "VMWARE",
						"status": 0,
						"kilobytesTransferred": 1024000,
						"startTime": "2025-11-08T10:00:00Z",
						"endTime": "2025-11-08T11:00:00Z",
						"clientName": "test-client-v%s"
					}
				}],
				"meta": {"pagination": {"first": 0, "last": 0, "limit": 1, "offset": 0, "next": 0}}
			}`, version)
			_, _ = w.Write([]byte(jobsResponse))

		case strings.Contains(r.URL.Path, "/storage/storage-units"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			storageResponse := fmt.Sprintf(`{
				"data": [{
					"type": "storageUnit",
					"id": "disk-pool-v%s",
					"attributes": {
						"name": "disk-pool-v%s",
						"storageType": "Disk",
						"storageServerType": "MEDIA_SERVER",
						"freeCapacityBytes": 5368709120000,
						"usedCapacityBytes": 2684354560000,
						"totalCapacityBytes": 8053063680000
					}
				}],
				"meta": {"pagination": {"first": 0, "last": 0}}
			}`, version, version)
			_, _ = w.Write([]byte(storageResponse))

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}
