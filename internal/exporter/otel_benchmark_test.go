package exporter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fjacquet/nbu_exporter/internal/models"
	"github.com/fjacquet/nbu_exporter/internal/telemetry"
)

// BenchmarkFetchData_WithoutTracing benchmarks FetchData without tracing
func BenchmarkFetchData_WithoutTracing(b *testing.B) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		response := map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id": "1",
					"attributes": map[string]interface{}{
						"name": "test-resource",
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

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
			APIKey:     "test-key",
		},
	}

	client := NewNbuClient(cfg)
	client.tracer = nil // Ensure tracing is disabled

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result map[string]interface{}
		_ = client.FetchData(context.Background(), server.URL, &result)
	}
}

// BenchmarkFetchData_WithTracing_FullSampling benchmarks FetchData with tracing (sampling=1.0)
func BenchmarkFetchData_WithTracing_FullSampling(b *testing.B) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		response := map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id": "1",
					"attributes": map[string]interface{}{
						"name": "test-resource",
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Initialize telemetry with full sampling
	telemetryConfig := telemetry.Config{
		Enabled:         true,
		Endpoint:        "localhost:4317",
		Insecure:        true,
		SamplingRate:    1.0,
		ServiceName:     "nbu-exporter-bench",
		ServiceVersion:  "1.0.0-bench",
		NetBackupServer: "bench-server",
	}

	manager := telemetry.NewManager(telemetryConfig)
	ctx := context.Background()
	_ = manager.Initialize(ctx)
	defer func() {
		if manager.IsEnabled() {
			_ = manager.Shutdown(ctx)
		}
	}()

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
			APIKey:     "test-key",
		},
	}

	client := NewNbuClient(cfg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result map[string]interface{}
		_ = client.FetchData(context.Background(), server.URL, &result)
	}
}

// BenchmarkFetchData_WithTracing_PartialSampling benchmarks FetchData with tracing (sampling=0.1)
func BenchmarkFetchData_WithTracing_PartialSampling(b *testing.B) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		response := map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id": "1",
					"attributes": map[string]interface{}{
						"name": "test-resource",
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Initialize telemetry with partial sampling
	telemetryConfig := telemetry.Config{
		Enabled:         true,
		Endpoint:        "localhost:4317",
		Insecure:        true,
		SamplingRate:    0.1,
		ServiceName:     "nbu-exporter-bench",
		ServiceVersion:  "1.0.0-bench",
		NetBackupServer: "bench-server",
	}

	manager := telemetry.NewManager(telemetryConfig)
	ctx := context.Background()
	_ = manager.Initialize(ctx)
	defer func() {
		if manager.IsEnabled() {
			_ = manager.Shutdown(ctx)
		}
	}()

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
			APIKey:     "test-key",
		},
	}

	client := NewNbuClient(cfg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result map[string]interface{}
		_ = client.FetchData(context.Background(), server.URL, &result)
	}
}

// BenchmarkSpanCreation benchmarks span creation overhead
func BenchmarkSpanCreation(b *testing.B) {
	// Initialize telemetry
	telemetryConfig := telemetry.Config{
		Enabled:         true,
		Endpoint:        "localhost:4317",
		Insecure:        true,
		SamplingRate:    1.0,
		ServiceName:     "nbu-exporter-bench",
		ServiceVersion:  "1.0.0-bench",
		NetBackupServer: "bench-server",
	}

	manager := telemetry.NewManager(telemetryConfig)
	ctx := context.Background()
	_ = manager.Initialize(ctx)
	defer func() {
		if manager.IsEnabled() {
			_ = manager.Shutdown(ctx)
		}
	}()

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
			APIKey:     "test-key",
		},
	}

	client := NewNbuClient(cfg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx, span := client.createHTTPSpan(context.Background(), "test.operation")
		if span != nil {
			span.End()
		}
		_ = ctx
	}
}

// BenchmarkAttributeRecording benchmarks attribute recording overhead
func BenchmarkAttributeRecording(b *testing.B) {
	// Initialize telemetry
	telemetryConfig := telemetry.Config{
		Enabled:         true,
		Endpoint:        "localhost:4317",
		Insecure:        true,
		SamplingRate:    1.0,
		ServiceName:     "nbu-exporter-bench",
		ServiceVersion:  "1.0.0-bench",
		NetBackupServer: "bench-server",
	}

	manager := telemetry.NewManager(telemetryConfig)
	ctx := context.Background()
	_ = manager.Initialize(ctx)
	defer func() {
		if manager.IsEnabled() {
			_ = manager.Shutdown(ctx)
		}
	}()

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
			APIKey:     "test-key",
		},
	}

	client := NewNbuClient(cfg)
	ctx, span := client.createHTTPSpan(context.Background(), "test.operation")
	defer func() {
		if span != nil {
			span.End()
		}
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.recordHTTPAttributes(span, "GET", "http://example.com", 200, 0, 100, 50)
	}
}
