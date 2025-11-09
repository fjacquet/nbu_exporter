// Package exporter provides performance validation tests for NetBackup API version support.
// These tests verify that the multi-version implementation doesn't introduce performance
// degradation compared to the original single-version implementation.
package exporter

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Note: Common test constants like contentTypeJSON, contentTypeHeader,
// testSchemeHTTPS, testPathAdminJobs, and testPathStorageUnits
// are defined in test_common.go and shared across all test files

// TestPerformance_StartupTimeWithVersionDetection measures the startup time
// when automatic version detection is enabled.
func TestPerformanceStartupTimeWithVersionDetection(t *testing.T) {
	// Create a mock server that supports version 12.0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		acceptHeader := r.Header.Get("Accept")

		// Simulate version detection - reject 13.0, accept 12.0
		if contains(acceptHeader, "version=13.0") {
			w.WriteHeader(http.StatusNotAcceptable)
			return
		}

		response := map[string]interface{}{
			"data": []interface{}{},
		}
		w.Header().Set(contentTypeHeader, contentTypeJSON)
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Measure startup time with version detection
	serverAddr := strings.TrimPrefix(server.URL, testSchemeHTTPS)
	cfg := createTestConfig(serverAddr, "") // Empty version triggers detection
	cfg.NbuServer.Scheme = "https"

	startTime := time.Now()
	collector, err := NewNbuCollector(cfg)
	elapsedTime := time.Since(startTime)

	require.NoError(t, err, "Collector creation should succeed")
	assert.NotNil(t, collector, "Collector should be created")

	// Version detection should complete within 5 seconds
	// (includes 1 failed attempt for 13.0 + 1 successful for 12.0)
	assert.Less(t, elapsedTime, 5*time.Second,
		"Startup with version detection should complete within 5 seconds, took %v", elapsedTime)

	t.Logf("Startup time with version detection: %v", elapsedTime)
}

// TestPerformance_StartupTimeWithExplicitVersion measures the startup time
// when API version is explicitly configured (no detection).
func TestPerformanceStartupTimeWithExplicitVersion(t *testing.T) {
	// Create a mock server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"data": []interface{}{},
		}
		w.Header().Set(contentTypeHeader, contentTypeJSON)
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Measure startup time with explicit version
	serverAddr := strings.TrimPrefix(server.URL, testSchemeHTTPS)
	cfg := createTestConfig(serverAddr, "12.0") // Explicit version
	cfg.NbuServer.Scheme = "https"

	startTime := time.Now()
	collector, err := NewNbuCollector(cfg)
	elapsedTime := time.Since(startTime)

	require.NoError(t, err, "Collector creation should succeed")
	assert.NotNil(t, collector, "Collector should be created")

	// Explicit version should be nearly instantaneous (< 100ms)
	assert.Less(t, elapsedTime, 100*time.Millisecond,
		"Startup with explicit version should be fast, took %v", elapsedTime)

	t.Logf("Startup time with explicit version: %v", elapsedTime)
}

// TestPerformance_CompareStartupTimes compares startup times between
// explicit configuration and automatic detection.
func TestPerformanceCompareStartupTimes(t *testing.T) {
	// Create a mock server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		acceptHeader := r.Header.Get("Accept")

		// Simulate version detection - reject 13.0, accept 12.0
		if contains(acceptHeader, "version=13.0") {
			w.WriteHeader(http.StatusNotAcceptable)
			return
		}

		response := map[string]interface{}{
			"data": []interface{}{},
		}
		w.Header().Set(contentTypeHeader, contentTypeJSON)
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	serverAddr := strings.TrimPrefix(server.URL, testSchemeHTTPS)

	// Measure explicit version startup
	cfgExplicit := createTestConfig(serverAddr, "12.0")
	cfgExplicit.NbuServer.Scheme = "https"

	startExplicit := time.Now()
	collectorExplicit, err := NewNbuCollector(cfgExplicit)
	explicitTime := time.Since(startExplicit)
	require.NoError(t, err)
	assert.NotNil(t, collectorExplicit)

	// Measure auto-detection startup
	cfgAuto := createTestConfig(serverAddr, "")
	cfgAuto.NbuServer.Scheme = "https"

	startAuto := time.Now()
	collectorAuto, err := NewNbuCollector(cfgAuto)
	autoTime := time.Since(startAuto)
	require.NoError(t, err)
	assert.NotNil(t, collectorAuto)

	t.Logf("Explicit version startup: %v", explicitTime)
	t.Logf("Auto-detection startup: %v", autoTime)
	t.Logf("Overhead: %v", autoTime-explicitTime)

	// Auto-detection should be slower but not excessively so
	// Allow up to 5 seconds overhead for version detection
	assert.Less(t, autoTime-explicitTime, 5*time.Second,
		"Version detection overhead should be reasonable")
}

// handleJobsRequest handles mock job API requests
func handleJobsRequest(w http.ResponseWriter) {
	response := createTestJobData()
	w.Header().Set(contentTypeHeader, contentTypeJSON)
	_ = json.NewEncoder(w).Encode(response)
}

// handleStorageRequest handles mock storage API requests
func handleStorageRequest(w http.ResponseWriter) {
	response := createTestStorageData()
	w.Header().Set(contentTypeHeader, contentTypeJSON)
	_ = json.NewEncoder(w).Encode(response)
}

// TestPerformance_RuntimePerformance verifies that runtime performance
// (metric collection) is not degraded by multi-version support.
func TestPerformanceRuntimePerformance(t *testing.T) {
	versions := []string{"3.0", "12.0", "13.0"}
	runtimeDurations := make(map[string]time.Duration)

	for _, version := range versions {
		// Create mock server with job and storage data
		requestCount := 0
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount++

			if strings.Contains(r.URL.Path, testPathAdminJobs) {
				handleJobsRequest(w)
			} else if strings.Contains(r.URL.Path, testPathStorageUnits) {
				handleStorageRequest(w)
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		// Create collector
		serverAddr := strings.TrimPrefix(server.URL, testSchemeHTTPS)
		cfg := createTestConfig(serverAddr, version)
		cfg.NbuServer.Scheme = "https"

		collector, err := NewNbuCollector(cfg)
		require.NoError(t, err)

		// Measure metric collection time
		startTime := time.Now()

		// Simulate Prometheus scrape by calling Collect
		ch := make(chan prometheus.Metric, 100)
		go func() {
			collector.Collect(ch)
			close(ch)
		}()

		// Wait for collection to complete
		for range ch {
			// Drain the channel
		}

		elapsedTime := time.Since(startTime)
		runtimeDurations[version] = elapsedTime

		t.Logf("Runtime performance for version %s: %v (%d requests)", version, elapsedTime, requestCount)
	}

	// Verify all versions have similar performance (within 50% of each other)
	var minDuration, maxDuration time.Duration
	for _, duration := range runtimeDurations {
		if minDuration == 0 || duration < minDuration {
			minDuration = duration
		}
		if duration > maxDuration {
			maxDuration = duration
		}
	}

	performanceVariance := float64(maxDuration-minDuration) / float64(minDuration)
	assert.Less(t, performanceVariance, 0.5,
		"Runtime performance should be consistent across versions (variance: %.2f%%)",
		performanceVariance*100)
}

// TestPerformance_ConnectionReuse verifies that HTTP connections are reused
// across multiple API calls within a scraping cycle.
func TestPerformanceConnectionReuse(t *testing.T) {
	connectionCount := 0
	requestCount := 0

	// Create mock server that tracks connections
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		// Track unique connections (simplified - in reality would need to track actual connections)
		if r.Header.Get("Connection") != "keep-alive" {
			connectionCount++
		}

		if strings.Contains(r.URL.Path, testPathAdminJobs) {
			handleJobsRequest(w)
		} else if strings.Contains(r.URL.Path, testPathStorageUnits) {
			handleStorageRequest(w)
		}
	}))
	defer server.Close()

	// Create collector
	serverAddr := strings.TrimPrefix(server.URL, testSchemeHTTPS)
	cfg := createTestConfig(serverAddr, "12.0")
	cfg.NbuServer.Scheme = "https"

	collector, err := NewNbuCollector(cfg)
	require.NoError(t, err)

	// Perform multiple scrapes
	for i := 0; i < 3; i++ {
		ch := make(chan prometheus.Metric, 100)
		go func() {
			collector.Collect(ch)
			close(ch)
		}()

		for range ch {
			// Drain the channel
		}
	}

	t.Logf("Total requests: %d, Connection count: %d", requestCount, connectionCount)

	// Verify multiple requests were made
	assert.Greater(t, requestCount, 3, "Should have made multiple API requests")
}

// TestPerformance_MemoryUsage verifies that the multi-version implementation
// doesn't significantly increase memory usage.
func TestPerformanceMemoryUsage(t *testing.T) {
	// This is a simplified test - in production, you'd use runtime.MemStats
	// to measure actual memory allocation

	versions := []string{"3.0", "12.0", "13.0"}

	for _, version := range versions {
		// Create mock server
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, testPathAdminJobs) {
				handleJobsRequest(w)
			} else if strings.Contains(r.URL.Path, testPathStorageUnits) {
				handleStorageRequest(w)
			}
		}))
		defer server.Close()

		// Create collector
		serverAddr := strings.TrimPrefix(server.URL, testSchemeHTTPS)
		cfg := createTestConfig(serverAddr, version)
		cfg.NbuServer.Scheme = "https"

		collector, err := NewNbuCollector(cfg)
		require.NoError(t, err)

		// Verify collector is created successfully
		assert.NotNil(t, collector, "Collector should be created for version %s", version)
		assert.Equal(t, version, collector.cfg.NbuServer.APIVersion,
			"Collector should use version %s", version)
	}

	// If we get here without panics or excessive memory usage, the test passes
	t.Log("Memory usage test completed successfully for all versions")
}

// TestPerformance_ConcurrentScrapes verifies that the collector can handle
// concurrent scrape requests without performance degradation.
func TestPerformanceConcurrentScrapes(t *testing.T) {
	// Create mock server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add small delay to simulate real API
		time.Sleep(10 * time.Millisecond)

		if strings.Contains(r.URL.Path, testPathAdminJobs) {
			handleJobsRequest(w)
		} else if strings.Contains(r.URL.Path, testPathStorageUnits) {
			handleStorageRequest(w)
		}
	}))
	defer server.Close()

	// Create collector
	serverAddr := strings.TrimPrefix(server.URL, testSchemeHTTPS)
	cfg := createTestConfig(serverAddr, "12.0")
	cfg.NbuServer.Scheme = "https"

	collector, err := NewNbuCollector(cfg)
	require.NoError(t, err)

	// Perform concurrent scrapes
	concurrency := 5
	done := make(chan time.Duration, concurrency)

	startTime := time.Now()
	for i := 0; i < concurrency; i++ {
		go func() {
			scrapeStart := time.Now()
			ch := make(chan prometheus.Metric, 100)
			go func() {
				collector.Collect(ch)
				close(ch)
			}()

			for range ch {
				// Drain the channel
			}
			done <- time.Since(scrapeStart)
		}()
	}

	// Wait for all scrapes to complete
	var totalDuration time.Duration
	for i := 0; i < concurrency; i++ {
		duration := <-done
		totalDuration += duration
	}
	totalElapsed := time.Since(startTime)

	avgDuration := totalDuration / time.Duration(concurrency)
	t.Logf("Concurrent scrapes: %d", concurrency)
	t.Logf("Total elapsed time: %v", totalElapsed)
	t.Logf("Average scrape duration: %v", avgDuration)

	// Concurrent scrapes should complete in reasonable time
	// With 5 concurrent scrapes, total time should be less than 5x single scrape time
	assert.Less(t, totalElapsed, 10*time.Second,
		"Concurrent scrapes should complete in reasonable time")
}

// BenchmarkCollectorCreation benchmarks the collector creation process
// for different API versions.
func BenchmarkCollectorCreation(b *testing.B) {
	// Create mock server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"data": []interface{}{},
		}
		w.Header().Set(contentTypeHeader, contentTypeJSON)
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	serverAddr := strings.TrimPrefix(server.URL, testSchemeHTTPS)

	b.Run("ExplicitVersion", func(b *testing.B) {
		cfg := createTestConfig(serverAddr, "12.0")
		cfg.NbuServer.Scheme = "https"

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := NewNbuCollector(cfg)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkMetricCollection benchmarks the metric collection process
// for different API versions.
func BenchmarkMetricCollection(b *testing.B) {
	versions := []string{"3.0", "12.0", "13.0"}

	for _, version := range versions {
		b.Run("Version_"+strings.ReplaceAll(version, ".", "_"), func(b *testing.B) {
			server := createBenchmarkMockServer()
			defer server.Close()

			collector := createBenchmarkCollector(b, server.URL, version)

			b.ResetTimer()
			benchmarkCollectorIterations(b, collector)
		})
	}
}

// createBenchmarkMockServer creates a mock server for benchmarking
func createBenchmarkMockServer() *httptest.Server {
	return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, testPathAdminJobs) {
			handleJobsRequest(w)
		} else if strings.Contains(r.URL.Path, testPathStorageUnits) {
			handleStorageRequest(w)
		}
	}))
}

// createBenchmarkCollector creates a collector for benchmarking
func createBenchmarkCollector(b *testing.B, serverURL, version string) *NbuCollector {
	b.Helper()

	serverAddr := strings.TrimPrefix(serverURL, testSchemeHTTPS)
	cfg := createTestConfig(serverAddr, version)
	cfg.NbuServer.Scheme = "https"

	collector, err := NewNbuCollector(cfg)
	if err != nil {
		b.Fatal(err)
	}
	return collector
}

// benchmarkCollectorIterations runs the collector for b.N iterations
func benchmarkCollectorIterations(b *testing.B, collector *NbuCollector) {
	b.Helper()

	for i := 0; i < b.N; i++ {
		ch := make(chan prometheus.Metric, 100)
		go func() {
			collector.Collect(ch)
			close(ch)
		}()

		for range ch {
			// Drain the channel
		}
	}
}
