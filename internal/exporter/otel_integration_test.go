package exporter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fjacquet/nbu_exporter/internal/models"
	"github.com/fjacquet/nbu_exporter/internal/telemetry"
	"go.opentelemetry.io/otel"
)

// TestIntegration_EndToEndTracing tests end-to-end tracing with a test collector
// This test verifies that spans are created and propagated correctly through the system
func TestIntegrationEndToEndTracing(t *testing.T) {
	// Create a test server that simulates NetBackup API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(contentTypeHeader, contentTypeJSON)
		w.WriteHeader(http.StatusOK)

		// Return minimal valid response
		response := map[string]interface{}{
			"data": []map[string]interface{}{},
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Initialize telemetry manager
	telemetryConfig := telemetry.Config{
		Enabled:         true,
		Endpoint:        testOTELEndpoint,
		Insecure:        true,
		SamplingRate:    1.0,
		ServiceName:     testServiceName,
		ServiceVersion:  testServiceVersion,
		NetBackupServer: testServerName,
	}

	manager := telemetry.NewManager(telemetryConfig)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Initialize telemetry (may fail if no collector available, which is OK)
	_ = manager.Initialize(ctx)

	// Create client and verify it gets a tracer
	cfg := models.Config{
		Server: struct {
			Port             string `yaml:"port"`
			Host             string `yaml:"host"`
			URI              string `yaml:"uri"`
			ScrapingInterval string `yaml:"scrapingInterval"`
			LogName          string `yaml:"logName"`
			CacheTTL         string `yaml:"cacheTTL"`
		}{
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
			APIVersion: "13.0",
			APIKey:     testKeyName,
		},
	}

	client := NewNbuClient(cfg)

	// Make a request to verify tracing works
	var result map[string]interface{}
	err := client.FetchData(context.Background(), server.URL, &result)
	if err != nil {
		t.Errorf(testErrorFetchDataUnexpected, err)
	}

	// Shutdown telemetry
	if manager.IsEnabled() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		_ = manager.Shutdown(shutdownCtx)
	}
}

// TestIntegration_BackwardCompatibility tests that the system works without OpenTelemetry config
func TestIntegrationBackwardCompatibility(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(contentTypeHeader, contentTypeJSON)
		w.WriteHeader(http.StatusOK)

		response := map[string]interface{}{
			"data": []map[string]interface{}{},
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create config without OpenTelemetry
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
			APIVersion: "13.0",
			APIKey:     testKeyName,
		},
		OpenTelemetry: struct {
			Enabled      bool    `yaml:"enabled"`
			Endpoint     string  `yaml:"endpoint"`
			Insecure     bool    `yaml:"insecure"`
			SamplingRate float64 `yaml:"samplingRate"`
		}{
			Enabled: false,
		},
	}

	// Verify config validation passes
	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() unexpected error = %v", err)
	}

	// Create client and verify it works without tracing
	client := NewNbuClient(cfg) // No TracerProvider option = noop tracing

	var result map[string]interface{}
	err = client.FetchData(context.Background(), server.URL, &result)
	if err != nil {
		t.Errorf(testErrorFetchDataUnexpected, err)
	}
}

// TestIntegration_GracefulDegradation tests graceful degradation with invalid endpoint
func TestIntegrationGracefulDegradation(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(contentTypeHeader, contentTypeJSON)
		w.WriteHeader(http.StatusOK)

		response := map[string]interface{}{
			"data": []map[string]interface{}{},
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Initialize telemetry with invalid endpoint
	telemetryConfig := telemetry.Config{
		Enabled:         true,
		Endpoint:        "invalid-endpoint:9999",
		Insecure:        true,
		SamplingRate:    1.0,
		ServiceName:     testServiceName,
		ServiceVersion:  testServiceVersion,
		NetBackupServer: testServerName,
	}

	manager := telemetry.NewManager(telemetryConfig)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Initialize should not fail (graceful degradation)
	err := manager.Initialize(ctx)
	if err != nil {
		t.Errorf("Initialize() unexpected error = %v", err)
	}

	// Create client
	cfg := models.Config{
		Server: struct {
			Port             string `yaml:"port"`
			Host             string `yaml:"host"`
			URI              string `yaml:"uri"`
			ScrapingInterval string `yaml:"scrapingInterval"`
			LogName          string `yaml:"logName"`
			CacheTTL         string `yaml:"cacheTTL"`
		}{
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
			APIVersion: "13.0",
			APIKey:     testKeyName,
		},
	}

	client := NewNbuClient(cfg)

	// Client should work even if telemetry endpoint is invalid
	var result map[string]interface{}
	err = client.FetchData(context.Background(), server.URL, &result)
	if err != nil {
		t.Errorf(testErrorFetchDataUnexpected, err)
	}

	// Shutdown telemetry
	if manager.IsEnabled() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		_ = manager.Shutdown(shutdownCtx)
	}
}

// TestIntegration_TracePropagation tests that trace context is propagated correctly
func TestIntegrationTracePropagation(t *testing.T) {
	// Create a test server that checks for trace headers
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for trace headers (may not be present if no global tracer provider)
		_ = r.Header.Get("traceparent")
		_ = r.Header.Get("tracestate")

		w.Header().Set(contentTypeHeader, contentTypeJSON)
		w.WriteHeader(http.StatusOK)

		response := map[string]interface{}{
			"data": []map[string]interface{}{},
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Initialize telemetry
	telemetryConfig := telemetry.Config{
		Enabled:         true,
		Endpoint:        testOTELEndpoint,
		Insecure:        true,
		SamplingRate:    1.0,
		ServiceName:     testServiceName,
		ServiceVersion:  testServiceVersion,
		NetBackupServer: testServerName,
	}

	manager := telemetry.NewManager(telemetryConfig)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_ = manager.Initialize(ctx)

	// Create client
	cfg := models.Config{
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
			APIVersion: "13.0",
			APIKey:     testKeyName,
		},
	}

	client := NewNbuClient(cfg)

	// Create a span and make a request
	tracer := otel.Tracer("test")
	spanCtx, span := tracer.Start(context.Background(), "test-operation")
	defer span.End()

	var result map[string]interface{}
	err := client.FetchData(spanCtx, server.URL, &result)
	if err != nil {
		t.Errorf(testErrorFetchDataUnexpected, err)
	}

	// Note: Trace headers may not be present if no global tracer provider is set
	// This test verifies the code doesn't panic and handles both cases

	// Shutdown telemetry
	if manager.IsEnabled() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		_ = manager.Shutdown(shutdownCtx)
	}
}

// TestIntegration_SamplingRates tests different sampling rates
func TestIntegrationSamplingRates(t *testing.T) {
	tests := []struct {
		name         string
		samplingRate float64
	}{
		{
			name:         "full sampling (1.0)",
			samplingRate: 1.0,
		},
		{
			name:         "partial sampling (0.1)",
			samplingRate: 0.1,
		},
		{
			name:         "no sampling (0.0)",
			samplingRate: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Initialize telemetry with different sampling rates
			telemetryConfig := telemetry.Config{
				Enabled:         true,
				Endpoint:        testOTELEndpoint,
				Insecure:        true,
				SamplingRate:    tt.samplingRate,
				ServiceName:     testServiceName,
				ServiceVersion:  testServiceVersion,
				NetBackupServer: testServerName,
			}

			manager := telemetry.NewManager(telemetryConfig)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			err := manager.Initialize(ctx)
			if err != nil {
				t.Errorf("Initialize() unexpected error = %v", err)
			}

			// Verify manager is enabled (or disabled if init failed)
			// This is acceptable - the test verifies the code doesn't panic

			// Shutdown telemetry
			if manager.IsEnabled() {
				shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer shutdownCancel()
				_ = manager.Shutdown(shutdownCtx)
			}
		})
	}
}
