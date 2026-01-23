package exporter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fjacquet/nbu_exporter/internal/models"
	"github.com/fjacquet/nbu_exporter/internal/telemetry"
	"go.opentelemetry.io/otel/trace"
)

const (
	benchContentType     = "Content-Type"
	benchApplicationJSON = "application/json"
	benchTestResource    = "test-resource"
	benchTestKey         = "test-key"
	benchAPIVersion      = "13.0"
	benchEndpoint        = "localhost:4317"
	benchServiceName     = "nbu-exporter-bench"
	benchServiceVersion  = "1.0.0-bench"
	benchNetBackupServer = "bench-server"
	benchTestOperation   = "test.operation"
	benchHTTPMethod      = "GET"
	benchExampleURL      = "http://example.com"
	benchHTTPStatus      = 200
	benchResponseSize    = 100
	benchDuration        = 50
)

// BenchmarkFetchDataWithoutTracing benchmarks FetchData without tracing
func BenchmarkFetchDataWithoutTracing(b *testing.B) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(benchContentType, benchApplicationJSON)
		w.WriteHeader(http.StatusOK)

		response := map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id": "1",
					"attributes": map[string]interface{}{
						"name": benchTestResource,
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
			APIVersion: benchAPIVersion,
			APIKey:     benchTestKey,
		},
	}

	client := NewNbuClient(cfg) // No TracerProvider option = noop tracing

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result map[string]interface{}
		_ = client.FetchData(context.Background(), server.URL, &result)
	}
}

// BenchmarkFetchDataWithTracingFullSampling benchmarks FetchData with tracing (sampling=1.0)
func BenchmarkFetchDataWithTracingFullSampling(b *testing.B) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(benchContentType, benchApplicationJSON)
		w.WriteHeader(http.StatusOK)

		response := map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id": "1",
					"attributes": map[string]interface{}{
						"name": benchTestResource,
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
		Endpoint:        benchEndpoint,
		Insecure:        true,
		SamplingRate:    1.0,
		ServiceName:     benchServiceName,
		ServiceVersion:  benchServiceVersion,
		NetBackupServer: benchNetBackupServer,
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
			APIVersion: benchAPIVersion,
			APIKey:     benchTestKey,
		},
	}

	client := NewNbuClient(cfg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result map[string]interface{}
		_ = client.FetchData(context.Background(), server.URL, &result)
	}
}

// BenchmarkFetchDataWithTracingPartialSampling benchmarks FetchData with tracing (sampling=0.1)
func BenchmarkFetchDataWithTracingPartialSampling(b *testing.B) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(benchContentType, benchApplicationJSON)
		w.WriteHeader(http.StatusOK)

		response := map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id": "1",
					"attributes": map[string]interface{}{
						"name": benchTestResource,
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
		Endpoint:        benchEndpoint,
		Insecure:        true,
		SamplingRate:    0.1,
		ServiceName:     benchServiceName,
		ServiceVersion:  benchServiceVersion,
		NetBackupServer: benchNetBackupServer,
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
			APIVersion: benchAPIVersion,
			APIKey:     benchTestKey,
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
		Endpoint:        benchEndpoint,
		Insecure:        true,
		SamplingRate:    1.0,
		ServiceName:     benchServiceName,
		ServiceVersion:  benchServiceVersion,
		NetBackupServer: benchNetBackupServer,
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
			APIVersion: benchAPIVersion,
			APIKey:     benchTestKey,
		},
	}

	client := NewNbuClient(cfg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx, span := client.tracing.StartSpan(context.Background(), benchTestOperation, trace.SpanKindClient)
		span.End()
		_ = ctx
	}
}

// BenchmarkAttributeRecording benchmarks attribute recording overhead
func BenchmarkAttributeRecording(b *testing.B) {
	// Initialize telemetry
	telemetryConfig := telemetry.Config{
		Enabled:         true,
		Endpoint:        benchEndpoint,
		Insecure:        true,
		SamplingRate:    1.0,
		ServiceName:     benchServiceName,
		ServiceVersion:  benchServiceVersion,
		NetBackupServer: benchNetBackupServer,
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
			APIVersion: benchAPIVersion,
			APIKey:     benchTestKey,
		},
	}

	client := NewNbuClient(cfg)
	ctx, span := client.tracing.StartSpan(context.Background(), benchTestOperation, trace.SpanKindClient)
	defer span.End()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.recordHTTPAttributes(span, benchHTTPMethod, benchExampleURL, benchHTTPStatus, 0, benchResponseSize, benchDuration)
	}
	_ = ctx
}
