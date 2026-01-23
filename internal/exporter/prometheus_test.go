// Package exporter provides tests for the Prometheus collector implementation.
package exporter

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fjacquet/nbu_exporter/internal/models"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewNbuCollector_ExplicitVersion tests collector creation with an explicitly configured API version.
// It verifies that when an API version is provided in the configuration, the collector
// is created successfully without performing version detection.
func TestNewNbuCollectorExplicitVersion(t *testing.T) {
	tests := []struct {
		name       string
		apiVersion string
	}{
		{
			name:       "API version 13.0",
			apiVersion: models.APIVersion130,
		},
		{
			name:       "API version 12.0",
			apiVersion: models.APIVersion120,
		},
		{
			name:       "API version 3.0",
			apiVersion: models.APIVersion30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock server that accepts any version
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"data": []}`))
			}))
			defer server.Close()

			cfg := models.Config{}
			cfg.NbuServer.Scheme = "https"
			// Extract host and port from server URL (format: https://host:port)
			cfg.NbuServer.Host = server.Listener.Addr().(*net.TCPAddr).IP.String()
			cfg.NbuServer.Port = fmt.Sprintf("%d", server.Listener.Addr().(*net.TCPAddr).Port)
			cfg.NbuServer.URI = ""
			cfg.NbuServer.APIKey = testAPIKey
			cfg.NbuServer.APIVersion = tt.apiVersion
			cfg.NbuServer.InsecureSkipVerify = true
			cfg.Server.ScrapingInterval = "5m"

			// Create collector with explicit version
			collector, err := NewNbuCollector(cfg)

			// Verify collector was created successfully
			require.NoError(t, err, "Collector creation should succeed with explicit version")
			require.NotNil(t, collector, "Collector should not be nil")
			assert.Equal(t, tt.apiVersion, collector.cfg.NbuServer.APIVersion, "API version should match configured version")
		})
	}
}

// createVersionMockServer creates a mock server that responds to version detection requests
func createVersionMockServer(t *testing.T, responses map[string]int) *httptest.Server {
	return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		version := extractVersionFromHeader(r.Header.Get("Accept"))
		handleVersionResponse(w, version, responses)
	}))
}

// extractVersionFromHeader extracts the API version from the Accept header
func extractVersionFromHeader(acceptHeader string) string {
	for _, v := range []string{"13.0", "12.0", "3.0"} {
		if fmt.Sprintf("application/vnd.netbackup+json;version=%s", v) == acceptHeader {
			return v
		}
	}
	return ""
}

// handleVersionResponse writes the appropriate response based on version and configured responses
func handleVersionResponse(w http.ResponseWriter, version string, responses map[string]int) {
	if statusCode, ok := responses[version]; ok {
		w.WriteHeader(statusCode)
		if statusCode == http.StatusOK {
			_, _ = w.Write([]byte(`{"data": []}`))
		}
	} else {
		w.WriteHeader(http.StatusNotAcceptable)
	}
}

// assertCollectorResult validates the collector creation result
func assertCollectorResult(t *testing.T, collector *NbuCollector, err error, expectError bool, expectedVersion string) {
	if expectError {
		assert.Error(t, err, "Collector creation should fail when all versions are unsupported")
		assert.Nil(t, collector, "Collector should be nil on error")
	} else {
		require.NoError(t, err, "Collector creation should succeed with automatic detection")
		require.NotNil(t, collector, "Collector should not be nil")
		assert.Equal(t, expectedVersion, collector.cfg.NbuServer.APIVersion, "API version should match detected version")
	}
}

// createTestConfigWithServer creates a test configuration for the given server
func createTestConfigWithServer(server *httptest.Server) models.Config {
	cfg := models.Config{}
	cfg.NbuServer.Scheme = "https"
	cfg.NbuServer.Host = server.Listener.Addr().(*net.TCPAddr).IP.String()
	cfg.NbuServer.Port = fmt.Sprintf("%d", server.Listener.Addr().(*net.TCPAddr).Port)
	cfg.NbuServer.URI = ""
	cfg.NbuServer.APIKey = testAPIKey
	cfg.NbuServer.APIVersion = "" // Empty to trigger detection
	cfg.NbuServer.InsecureSkipVerify = true
	cfg.Server.ScrapingInterval = "5m"
	return cfg
}

// TestNewNbuCollector_AutomaticDetection tests collector creation with automatic version detection.
// It verifies that when no API version is configured, the collector performs automatic
// detection and selects the highest supported version.
func TestNewNbuCollectorAutomaticDetection(t *testing.T) {
	// Skip this test as it requires a real NetBackup server or complex mocking
	// The version detection logic is already tested in version_detector_test.go
	t.Skip("Skipping automatic detection test - covered by version_detector_test.go")
	tests := []struct {
		name            string
		serverResponses map[string]int // version -> HTTP status code
		expectedVersion string
		expectError     bool
	}{
		{
			name: "Detect version 13.0",
			serverResponses: map[string]int{
				"13.0": http.StatusOK,
			},
			expectedVersion: models.APIVersion130,
			expectError:     false,
		},
		{
			name: "Fallback to version 12.0",
			serverResponses: map[string]int{
				"13.0": http.StatusNotAcceptable,
				"12.0": http.StatusOK,
			},
			expectedVersion: models.APIVersion120,
			expectError:     false,
		},
		{
			name: "Fallback to version 3.0",
			serverResponses: map[string]int{
				"13.0": http.StatusNotAcceptable,
				"12.0": http.StatusNotAcceptable,
				"3.0":  http.StatusOK,
			},
			expectedVersion: models.APIVersion30,
			expectError:     false,
		},
		{
			name: "All versions fail",
			serverResponses: map[string]int{
				"13.0": http.StatusNotAcceptable,
				"12.0": http.StatusNotAcceptable,
				"3.0":  http.StatusNotAcceptable,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := createVersionMockServer(t, tt.serverResponses)
			defer server.Close()

			cfg := createTestConfigWithServer(server)
			collector, err := NewNbuCollector(cfg)

			assertCollectorResult(t, collector, err, tt.expectError, tt.expectedVersion)
		})
	}
}

// TestNewNbuCollector_DetectionFailure tests error handling when version detection fails.
// It verifies that appropriate errors are returned when the NetBackup server is unreachable
// or returns unexpected responses.
func TestNewNbuCollectorDetectionFailure(t *testing.T) {
	// Skip this test as it requires a real NetBackup server or complex mocking
	// The error handling logic is already tested in version_detector_test.go
	t.Skip("Skipping detection failure test - covered by version_detector_test.go")
	tests := []struct {
		name        string
		setupServer func() *httptest.Server
		expectError bool
		errorMsg    string
	}{
		{
			name: "Authentication failure",
			setupServer: func() *httptest.Server {
				return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusUnauthorized)
				}))
			},
			expectError: true,
			errorMsg:    "automatic API version detection failed",
		},
		{
			name: "Server error",
			setupServer: func() *httptest.Server {
				return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				}))
			},
			expectError: true,
			errorMsg:    "automatic API version detection failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.Close()

			cfg := models.Config{}
			cfg.NbuServer.Scheme = "https"
			cfg.NbuServer.Host = server.Listener.Addr().(*net.TCPAddr).IP.String()
			cfg.NbuServer.Port = fmt.Sprintf("%d", server.Listener.Addr().(*net.TCPAddr).Port)
			cfg.NbuServer.URI = ""
			cfg.NbuServer.APIKey = testAPIKey
			cfg.NbuServer.APIVersion = "" // Empty to trigger detection
			cfg.NbuServer.InsecureSkipVerify = true
			cfg.Server.ScrapingInterval = "5m"

			// Create collector - should fail
			collector, err := NewNbuCollector(cfg)

			if tt.expectError {
				assert.Error(t, err, "Collector creation should fail")
				assert.Contains(t, err.Error(), tt.errorMsg, "Error message should contain expected text")
				assert.Nil(t, collector, "Collector should be nil on error")
			}
		})
	}
}

// TestNbuCollector_APIVersionMetric tests that the API version metric is properly exposed.
// It verifies that the nbu_api_version metric is registered and contains the correct version label.
func TestNbuCollectorAPIVersionMetric(t *testing.T) {
	// Create a mock server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data": []}`))
	}))
	defer server.Close()

	cfg := models.Config{}
	cfg.NbuServer.Scheme = "https"
	cfg.NbuServer.Host = server.Listener.Addr().(*net.TCPAddr).IP.String()
	cfg.NbuServer.Port = fmt.Sprintf("%d", server.Listener.Addr().(*net.TCPAddr).Port)
	cfg.NbuServer.URI = ""
	cfg.NbuServer.APIKey = testAPIKey
	cfg.NbuServer.APIVersion = models.APIVersion130
	cfg.NbuServer.InsecureSkipVerify = true
	cfg.Server.ScrapingInterval = "5m"

	// Create collector
	collector, err := NewNbuCollector(cfg)
	require.NoError(t, err)
	require.NotNil(t, collector)

	// Verify API version metric descriptor is present
	assert.NotNil(t, collector.nbuAPIVersion, "API version metric descriptor should be initialized")

	// Create a test registry and register the collector
	registry := prometheus.NewRegistry()
	err = registry.Register(collector)
	require.NoError(t, err, "Collector should register successfully")

	// Collect metrics
	metricChan := make(chan prometheus.Metric, 10)
	go func() {
		collector.Collect(metricChan)
		close(metricChan)
	}()

	// Verify API version metric is collected
	foundAPIVersionMetric := false
	for metric := range metricChan {
		desc := metric.Desc()
		if desc.String() == collector.nbuAPIVersion.String() {
			foundAPIVersionMetric = true
			break
		}
	}

	assert.True(t, foundAPIVersionMetric, "API version metric should be collected")
}

// TestNbuCollector_Describe tests that all metric descriptors are properly registered.
// It verifies that the Describe method sends all expected metric descriptors to the channel.
func TestNbuCollectorDescribe(t *testing.T) {
	// Create a mock server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data": []}`))
	}))
	defer server.Close()

	cfg := models.Config{}
	cfg.NbuServer.Scheme = "https"
	cfg.NbuServer.Host = server.Listener.Addr().(*net.TCPAddr).IP.String()
	cfg.NbuServer.Port = fmt.Sprintf("%d", server.Listener.Addr().(*net.TCPAddr).Port)
	cfg.NbuServer.URI = ""
	cfg.NbuServer.APIKey = testAPIKey
	cfg.NbuServer.APIVersion = models.APIVersion130
	cfg.NbuServer.InsecureSkipVerify = true
	cfg.Server.ScrapingInterval = "5m"

	// Create collector
	collector, err := NewNbuCollector(cfg)
	require.NoError(t, err)
	require.NotNil(t, collector)

	// Collect descriptors
	descChan := make(chan *prometheus.Desc, 10)
	go func() {
		collector.Describe(descChan)
		close(descChan)
	}()

	// Count descriptors
	descriptorCount := 0
	expectedDescriptors := []string{
		"nbu_disk_bytes",
		"nbu_response_time_ms",
		"nbu_jobs_bytes",
		"nbu_jobs_count",
		"nbu_status_count",
		"nbu_api_version",
		"nbu_up",
		"nbu_last_scrape_timestamp_seconds",
	}

	descriptorNames := make(map[string]bool)
	for desc := range descChan {
		descriptorCount++
		descStr := desc.String()
		for _, name := range expectedDescriptors {
			if contains(descStr, name) {
				descriptorNames[name] = true
			}
		}
	}

	assert.Equal(t, len(expectedDescriptors), descriptorCount, "Should have correct number of metric descriptors")
	assert.Equal(t, len(expectedDescriptors), len(descriptorNames), "All expected descriptors should be present")
}

// TestNbuCollector_CreateScrapeSpan_NilSafe tests that span creation is nil-safe
func TestNbuCollectorCreateScrapeSpanNilSafe(t *testing.T) {
	cfg := models.Config{
		Server: struct {
			Port             string `yaml:"port"`
			Host             string `yaml:"host"`
			URI              string `yaml:"uri"`
			ScrapingInterval string `yaml:"scrapingInterval"`
			LogName          string `yaml:"logName"`
			CacheTTL         string `yaml:"cacheTTL"`
		}{
			Port:             "2112",
			Host:             "localhost",
			URI:              testPathMetrics,
			ScrapingInterval: "5m",
		},
		NbuServer: struct {
			Port               string `yaml:"port"`
			Scheme             string `yaml:"scheme"`
			URI                string `yaml:"uri"`
			Domain             string `yaml:"domain"`
			DomainType         string `yaml:"domainType"`
			Host               string `yaml:"host"`
			APIKey             string `yaml:"apiKey"`
			APIVersion         string `yaml:"apiVersion"`
			ContentType        string `yaml:"contentType"`
			InsecureSkipVerify bool   `yaml:"insecureSkipVerify"`
		}{
			Host:       testServerNBUMaster,
			Port:       "1556",
			Scheme:     "https",
			URI:        testPathNetBackup,
			APIKey:     testKeyName,
			APIVersion: "13.0",
		},
	}

	client := NewNbuClient(cfg)
	tracing := NewTracerWrapper(nil, "test-collector") // noop tracer

	collector := &NbuCollector{
		cfg:     cfg,
		client:  client,
		tracing: tracing,
	}

	ctx := context.Background()
	newCtx, span := collector.createScrapeSpan(ctx)

	// TracerWrapper always returns valid context and span
	if newCtx == nil {
		t.Error("createScrapeSpan() should return valid context")
	}

	if span == nil {
		t.Error("createScrapeSpan() should return valid span (noop if no provider)")
	}

	// Should not panic
	span.End()
}

// TestNbuCollector_CreateScrapeSpan_WithTracer tests span creation with tracer
func TestNbuCollectorCreateScrapeSpanWithTracer(t *testing.T) {
	cfg := models.Config{
		Server: struct {
			Port             string `yaml:"port"`
			Host             string `yaml:"host"`
			URI              string `yaml:"uri"`
			ScrapingInterval string `yaml:"scrapingInterval"`
			LogName          string `yaml:"logName"`
			CacheTTL         string `yaml:"cacheTTL"`
		}{
			Port:             "2112",
			Host:             "localhost",
			URI:              testPathMetrics,
			ScrapingInterval: "5m",
		},
		NbuServer: struct {
			Port               string `yaml:"port"`
			Scheme             string `yaml:"scheme"`
			URI                string `yaml:"uri"`
			Domain             string `yaml:"domain"`
			DomainType         string `yaml:"domainType"`
			Host               string `yaml:"host"`
			APIKey             string `yaml:"apiKey"`
			APIVersion         string `yaml:"apiVersion"`
			ContentType        string `yaml:"contentType"`
			InsecureSkipVerify bool   `yaml:"insecureSkipVerify"`
		}{
			Host:       testServerNBUMaster,
			Port:       "1556",
			Scheme:     "https",
			URI:        testPathNetBackup,
			APIKey:     testKeyName,
			APIVersion: "13.0",
		},
	}

	client := NewNbuClient(cfg)

	// Create TracerWrapper (nil provider = noop)
	tracing := NewTracerWrapper(nil, "test-collector")

	collector := &NbuCollector{
		cfg:     cfg,
		client:  client,
		tracing: tracing,
	}

	ctx := context.Background()
	newCtx, span := collector.createScrapeSpan(ctx)

	// TracerWrapper always returns valid context and span
	if newCtx == nil {
		t.Error("createScrapeSpan() should return valid context")
	}

	if span == nil {
		t.Error("createScrapeSpan() should return valid span")
	}

	// Should not panic
	span.End()
}

// TestNbuCollector_Collect_WithoutTracing tests Collect without tracing
func TestNbuCollectorCollectWithoutTracing(t *testing.T) {
	// This test verifies that Collect works without tracing enabled
	// We can't easily test the full Collect method without a real NetBackup server,
	// but we can verify that the tracer nil-safety works

	cfg := models.Config{
		Server: struct {
			Port             string `yaml:"port"`
			Host             string `yaml:"host"`
			URI              string `yaml:"uri"`
			ScrapingInterval string `yaml:"scrapingInterval"`
			LogName          string `yaml:"logName"`
			CacheTTL         string `yaml:"cacheTTL"`
		}{
			Port:             "2112",
			Host:             "localhost",
			URI:              testPathMetrics,
			ScrapingInterval: "5m",
		},
		NbuServer: struct {
			Port               string `yaml:"port"`
			Scheme             string `yaml:"scheme"`
			URI                string `yaml:"uri"`
			Domain             string `yaml:"domain"`
			DomainType         string `yaml:"domainType"`
			Host               string `yaml:"host"`
			APIKey             string `yaml:"apiKey"`
			APIVersion         string `yaml:"apiVersion"`
			ContentType        string `yaml:"contentType"`
			InsecureSkipVerify bool   `yaml:"insecureSkipVerify"`
		}{
			Host:       testServerNBUMaster,
			Port:       "1556",
			Scheme:     "https",
			URI:        testPathNetBackup,
			APIKey:     testKeyName,
			APIVersion: "13.0",
		},
	}

	client := NewNbuClient(cfg)
	// Disable retries for faster test execution
	client.client.SetRetryCount(0)
	tracing := NewTracerWrapper(nil, "test-collector") // noop tracer
	storageCache := NewStorageCache(cfg.GetCacheTTL())

	collector := &NbuCollector{
		cfg:          cfg,
		client:       client,
		tracing:      tracing,
		storageCache: storageCache,
		nbuDiskSize: prometheus.NewDesc(
			"nbu_disk_bytes",
			"The quantity of storage bytes",
			[]string{"name", "type", "size"}, nil,
		),
		nbuResponseTime: prometheus.NewDesc(
			"nbu_response_time_ms",
			"The server response time in milliseconds",
			nil, nil,
		),
		nbuJobsSize: prometheus.NewDesc(
			"nbu_jobs_bytes",
			"The quantity of processed bytes",
			[]string{"action", "policy_type", "status"}, nil,
		),
		nbuJobsCount: prometheus.NewDesc(
			"nbu_jobs_count",
			"The quantity of jobs",
			[]string{"action", "policy_type", "status"}, nil,
		),
		nbuJobsStatusCount: prometheus.NewDesc(
			"nbu_status_count",
			"The quantity per status",
			[]string{"action", "status"}, nil,
		),
		nbuAPIVersion: prometheus.NewDesc(
			"nbu_api_version",
			"The NetBackup API version currently in use",
			[]string{"version"}, nil,
		),
		nbuUp: prometheus.NewDesc(
			"nbu_up",
			"1 if NetBackup API is reachable, 0 if all collections failed",
			nil, nil,
		),
		nbuLastScrapeTime: prometheus.NewDesc(
			"nbu_last_scrape_timestamp_seconds",
			"Unix timestamp of the last successful metric collection",
			[]string{"source"}, nil,
		),
	}

	// Create a channel to receive metrics
	ch := make(chan prometheus.Metric, 100)

	// Run Collect in a goroutine since it will likely fail (no real server)
	// but we want to verify it doesn't panic
	done := make(chan bool)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Collect() panicked: %v", r)
			}
			done <- true
		}()
		collector.Collect(ch)
	}()

	// Wait for Collect to finish or timeout
	select {
	case <-done:
		// Success - Collect completed without panic
	case <-time.After(5 * time.Second):
		t.Error("Collect() timed out")
	}

	close(ch)
}

// TestNbuCollector_TracingDisabled tests that collector works without tracing
func TestNbuCollectorTracingDisabled(t *testing.T) {
	cfg := models.Config{
		Server: struct {
			Port             string `yaml:"port"`
			Host             string `yaml:"host"`
			URI              string `yaml:"uri"`
			ScrapingInterval string `yaml:"scrapingInterval"`
			LogName          string `yaml:"logName"`
			CacheTTL         string `yaml:"cacheTTL"`
		}{
			Port:             "2112",
			Host:             "localhost",
			URI:              testPathMetrics,
			ScrapingInterval: "5m",
		},
		NbuServer: struct {
			Port               string `yaml:"port"`
			Scheme             string `yaml:"scheme"`
			URI                string `yaml:"uri"`
			Domain             string `yaml:"domain"`
			DomainType         string `yaml:"domainType"`
			Host               string `yaml:"host"`
			APIKey             string `yaml:"apiKey"`
			APIVersion         string `yaml:"apiVersion"`
			ContentType        string `yaml:"contentType"`
			InsecureSkipVerify bool   `yaml:"insecureSkipVerify"`
		}{
			Host:       testServerNBUMaster,
			Port:       "1556",
			Scheme:     "https",
			URI:        testPathNetBackup,
			APIKey:     testKeyName,
			APIVersion: "13.0",
		},
	}

	client := NewNbuClient(cfg) // No TracerProvider option = noop tracing

	// Create TracerWrapper for collector (nil provider = noop)
	tracing := NewTracerWrapper(nil, "test-collector")

	collector := &NbuCollector{
		cfg:     cfg,
		client:  client,
		tracing: tracing,
	}

	// Verify that createScrapeSpan works with noop tracer
	ctx := context.Background()
	newCtx, span := collector.createScrapeSpan(ctx)

	if newCtx == nil {
		t.Error("createScrapeSpan() should return valid context")
	}

	if span == nil {
		t.Error("createScrapeSpan() should return valid span (noop if no provider)")
	}

	// Should not panic
	span.End()
}

// TestNbuCollector_StorageCacheIntegration tests cache integration with collector
func TestNbuCollectorStorageCacheIntegration(t *testing.T) {
	// Create a test server that returns storage data
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.netbackup+json;version=13.0")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data": []}`))
	}))
	defer server.Close()

	// Parse the server URL to get host and port
	serverHost, serverPort, _ := net.SplitHostPort(server.Listener.Addr().String())

	cfg := models.Config{
		Server: struct {
			Port             string `yaml:"port"`
			Host             string `yaml:"host"`
			URI              string `yaml:"uri"`
			ScrapingInterval string `yaml:"scrapingInterval"`
			LogName          string `yaml:"logName"`
			CacheTTL         string `yaml:"cacheTTL"`
		}{
			Port:             "2112",
			Host:             "localhost",
			URI:              testPathMetrics,
			ScrapingInterval: "5m",
			CacheTTL:         "1m", // 1 minute cache for testing
		},
		NbuServer: struct {
			Port               string `yaml:"port"`
			Scheme             string `yaml:"scheme"`
			URI                string `yaml:"uri"`
			Domain             string `yaml:"domain"`
			DomainType         string `yaml:"domainType"`
			Host               string `yaml:"host"`
			APIKey             string `yaml:"apiKey"`
			APIVersion         string `yaml:"apiVersion"`
			ContentType        string `yaml:"contentType"`
			InsecureSkipVerify bool   `yaml:"insecureSkipVerify"`
		}{
			Host:               serverHost,
			Port:               serverPort,
			Scheme:             "https",
			URI:                "",
			APIKey:             testKeyName,
			APIVersion:         "13.0",
			InsecureSkipVerify: true,
		},
	}

	collector, err := NewNbuCollector(cfg)
	if err != nil {
		t.Fatalf("NewNbuCollector() error = %v", err)
	}
	defer func() {
		_ = collector.Close()
	}()

	// Verify cache was initialized
	cache := collector.GetStorageCache()
	if cache == nil {
		t.Fatal("GetStorageCache() should return non-nil cache")
	}

	// Verify TTL is configured correctly
	expectedTTL := time.Minute
	if cache.TTL() != expectedTTL {
		t.Errorf("Cache TTL = %v, want %v", cache.TTL(), expectedTTL)
	}
}

// TestNbuCollector_HelpStringIncludesTTL verifies metric HELP string documents caching
func TestNbuCollectorHelpStringIncludesTTL(t *testing.T) {
	// Create a test server that returns storage data
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.netbackup+json;version=13.0")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data": []}`))
	}))
	defer server.Close()

	// Parse the server URL to get host and port
	serverHost, serverPort, _ := net.SplitHostPort(server.Listener.Addr().String())

	cfg := models.Config{
		Server: struct {
			Port             string `yaml:"port"`
			Host             string `yaml:"host"`
			URI              string `yaml:"uri"`
			ScrapingInterval string `yaml:"scrapingInterval"`
			LogName          string `yaml:"logName"`
			CacheTTL         string `yaml:"cacheTTL"`
		}{
			Port:             "2112",
			Host:             "localhost",
			URI:              testPathMetrics,
			ScrapingInterval: "5m",
			CacheTTL:         "2m",
		},
		NbuServer: struct {
			Port               string `yaml:"port"`
			Scheme             string `yaml:"scheme"`
			URI                string `yaml:"uri"`
			Domain             string `yaml:"domain"`
			DomainType         string `yaml:"domainType"`
			Host               string `yaml:"host"`
			APIKey             string `yaml:"apiKey"`
			APIVersion         string `yaml:"apiVersion"`
			ContentType        string `yaml:"contentType"`
			InsecureSkipVerify bool   `yaml:"insecureSkipVerify"`
		}{
			Host:               serverHost,
			Port:               serverPort,
			Scheme:             "https",
			URI:                "",
			APIKey:             testKeyName,
			APIVersion:         "13.0",
			InsecureSkipVerify: true,
		},
	}

	collector, err := NewNbuCollector(cfg)
	if err != nil {
		t.Fatalf("NewNbuCollector() error = %v", err)
	}
	defer func() {
		_ = collector.Close()
	}()

	// Get descriptors
	descCh := make(chan *prometheus.Desc, 10)
	collector.Describe(descCh)
	close(descCh)

	// Find nbu_disk_bytes descriptor and verify HELP string
	foundDiskBytes := false
	for desc := range descCh {
		descStr := desc.String()
		if !foundDiskBytes && strings.Contains(descStr, "nbu_disk_bytes") {
			foundDiskBytes = true
			// Verify the HELP string contains the caching information
			assert.Contains(t, descStr, "cached", "HELP string should mention caching")
			assert.Contains(t, descStr, "2m0s", "HELP string should include TTL duration")
		}
	}

	if !foundDiskBytes {
		t.Error("nbu_disk_bytes descriptor not found")
	}
}
