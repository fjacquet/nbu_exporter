// Package exporter provides tests for the Prometheus collector implementation.
package exporter

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fjacquet/nbu_exporter/internal/models"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
)

// TestNewNbuCollector_ExplicitVersion tests collector creation with an explicitly configured API version.
// It verifies that when an API version is provided in the configuration, the collector
// is created successfully without performing version detection.
func TestNewNbuCollector_ExplicitVersion(t *testing.T) {
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
			cfg.NbuServer.APIKey = "test-api-key"
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

// TestNewNbuCollector_AutomaticDetection tests collector creation with automatic version detection.
// It verifies that when no API version is configured, the collector performs automatic
// detection and selects the highest supported version.
func TestNewNbuCollector_AutomaticDetection(t *testing.T) {
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
			// Create a mock server that responds based on the Accept header version
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				acceptHeader := r.Header.Get("Accept")

				// Extract version from Accept header
				var version string
				for v := range tt.serverResponses {
					if fmt.Sprintf("application/vnd.netbackup+json;version=%s", v) == acceptHeader {
						version = v
						break
					}
				}

				if statusCode, ok := tt.serverResponses[version]; ok {
					w.WriteHeader(statusCode)
					if statusCode == http.StatusOK {
						_, _ = w.Write([]byte(`{"data": []}`))
					}
				} else {
					w.WriteHeader(http.StatusNotAcceptable)
				}
			}))
			defer server.Close()

			cfg := models.Config{}
			cfg.NbuServer.Scheme = "https"
			cfg.NbuServer.Host = server.Listener.Addr().(*net.TCPAddr).IP.String()
			cfg.NbuServer.Port = fmt.Sprintf("%d", server.Listener.Addr().(*net.TCPAddr).Port)
			cfg.NbuServer.URI = ""
			cfg.NbuServer.APIKey = "test-api-key"
			cfg.NbuServer.APIVersion = "" // Empty to trigger detection
			cfg.NbuServer.InsecureSkipVerify = true
			cfg.Server.ScrapingInterval = "5m"

			// Create collector with automatic detection
			collector, err := NewNbuCollector(cfg)

			if tt.expectError {
				assert.Error(t, err, "Collector creation should fail when all versions are unsupported")
				assert.Nil(t, collector, "Collector should be nil on error")
			} else {
				require.NoError(t, err, "Collector creation should succeed with automatic detection")
				require.NotNil(t, collector, "Collector should not be nil")
				assert.Equal(t, tt.expectedVersion, collector.cfg.NbuServer.APIVersion, "API version should match detected version")
			}
		})
	}
}

// TestNewNbuCollector_DetectionFailure tests error handling when version detection fails.
// It verifies that appropriate errors are returned when the NetBackup server is unreachable
// or returns unexpected responses.
func TestNewNbuCollector_DetectionFailure(t *testing.T) {
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
			cfg.NbuServer.APIKey = "test-api-key"
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
func TestNbuCollector_APIVersionMetric(t *testing.T) {
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
	cfg.NbuServer.APIKey = "test-api-key"
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
func TestNbuCollector_Describe(t *testing.T) {
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
	cfg.NbuServer.APIKey = "test-api-key"
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
func TestNbuCollector_CreateScrapeSpan_NilSafe(t *testing.T) {
	cfg := models.Config{
		Server: struct {
			Port             string `yaml:"port"`
			Host             string `yaml:"host"`
			URI              string `yaml:"uri"`
			ScrapingInterval string `yaml:"scrapingInterval"`
			LogName          string `yaml:"logName"`
		}{
			Port:             "2112",
			Host:             "localhost",
			URI:              "/metrics",
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
			Host:       "nbu-master",
			Port:       "1556",
			Scheme:     "https",
			URI:        "/netbackup",
			APIKey:     "test-key",
			APIVersion: "13.0",
		},
	}

	client := NewNbuClient(cfg)
	collector := &NbuCollector{
		cfg:    cfg,
		client: client,
		tracer: nil, // Ensure tracer is nil
	}

	ctx := context.Background()
	newCtx, span := collector.createScrapeSpan(ctx)

	// Should return original context and nil span when tracer is nil
	if newCtx != ctx {
		t.Error("createScrapeSpan() should return original context when tracer is nil")
	}

	if span != nil {
		t.Error("createScrapeSpan() should return nil span when tracer is nil")
	}
}

// TestNbuCollector_CreateScrapeSpan_WithTracer tests span creation with tracer
func TestNbuCollector_CreateScrapeSpan_WithTracer(t *testing.T) {
	cfg := models.Config{
		Server: struct {
			Port             string `yaml:"port"`
			Host             string `yaml:"host"`
			URI              string `yaml:"uri"`
			ScrapingInterval string `yaml:"scrapingInterval"`
			LogName          string `yaml:"logName"`
		}{
			Port:             "2112",
			Host:             "localhost",
			URI:              "/metrics",
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
			Host:       "nbu-master",
			Port:       "1556",
			Scheme:     "https",
			URI:        "/netbackup",
			APIKey:     "test-key",
			APIVersion: "13.0",
		},
	}

	client := NewNbuClient(cfg)

	// Get tracer from global provider (may be nil if not initialized)
	tracer := otel.Tracer("nbu-exporter-test")

	collector := &NbuCollector{
		cfg:    cfg,
		client: client,
		tracer: tracer,
	}

	ctx := context.Background()
	newCtx, span := collector.createScrapeSpan(ctx)

	// Context should be different (has span attached)
	if newCtx == ctx {
		t.Error("createScrapeSpan() should return new context with span")
	}

	// Span may be nil if no global tracer provider is set
	// This is acceptable - the test verifies the code doesn't panic
	if span != nil {
		span.End()
	}
}

// TestNbuCollector_Collect_WithoutTracing tests Collect without tracing
func TestNbuCollector_Collect_WithoutTracing(t *testing.T) {
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
		}{
			Port:             "2112",
			Host:             "localhost",
			URI:              "/metrics",
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
			Host:       "nbu-master",
			Port:       "1556",
			Scheme:     "https",
			URI:        "/netbackup",
			APIKey:     "test-key",
			APIVersion: "13.0",
		},
	}

	client := NewNbuClient(cfg)
	collector := &NbuCollector{
		cfg:    cfg,
		client: client,
		tracer: nil, // Ensure tracer is nil
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
func TestNbuCollector_TracingDisabled(t *testing.T) {
	cfg := models.Config{
		Server: struct {
			Port             string `yaml:"port"`
			Host             string `yaml:"host"`
			URI              string `yaml:"uri"`
			ScrapingInterval string `yaml:"scrapingInterval"`
			LogName          string `yaml:"logName"`
		}{
			Port:             "2112",
			Host:             "localhost",
			URI:              "/metrics",
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
			Host:       "nbu-master",
			Port:       "1556",
			Scheme:     "https",
			URI:        "/netbackup",
			APIKey:     "test-key",
			APIVersion: "13.0",
		},
	}

	client := NewNbuClient(cfg)
	client.tracer = nil // Ensure tracer is nil

	collector := &NbuCollector{
		cfg:    cfg,
		client: client,
		tracer: nil,
	}

	// Verify that createScrapeSpan works without tracer
	ctx := context.Background()
	newCtx, span := collector.createScrapeSpan(ctx)

	if newCtx != ctx {
		t.Error("createScrapeSpan() should return original context when tracer is nil")
	}

	if span != nil {
		t.Error("createScrapeSpan() should return nil span when tracer is nil")
	}
}
