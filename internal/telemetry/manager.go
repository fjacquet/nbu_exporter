// Package telemetry provides OpenTelemetry integration for the NBU Exporter.
// This package manages the lifecycle of OpenTelemetry tracing and provides
// instrumentation for NetBackup API calls and Prometheus scrape cycles.
package telemetry

import (
	"context"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"google.golang.org/grpc/credentials/insecure"
)

// Manager handles OpenTelemetry initialization, lifecycle management, and shutdown.
// It centralizes the configuration and management of the TracerProvider and ensures
// proper resource cleanup during application shutdown.
type Manager struct {
	enabled        bool
	tracerProvider *sdktrace.TracerProvider
	config         Config
}

// Config holds OpenTelemetry configuration settings for the telemetry manager.
// These settings control how traces are collected, sampled, and exported.
type Config struct {
	// Enabled indicates whether OpenTelemetry tracing is active
	Enabled bool

	// Endpoint is the OTLP gRPC collector endpoint (e.g., "localhost:4317")
	Endpoint string

	// Insecure controls whether to use an insecure connection (no TLS)
	Insecure bool

	// SamplingRate determines the percentage of traces to sample (0.0 to 1.0)
	// 1.0 = sample all traces, 0.1 = sample 10% of traces
	SamplingRate float64

	// ServiceName is the name of the service for resource attributes
	ServiceName string

	// ServiceVersion is the version of the service for resource attributes
	ServiceVersion string

	// NetBackupServer is the target NetBackup server hostname for resource attributes
	NetBackupServer string
}

// NewManager creates a new telemetry manager with the provided configuration.
// The manager is not initialized until Initialize() is called.
//
// Parameters:
//   - cfg: Configuration settings for OpenTelemetry
//
// Returns a new Manager instance.
func NewManager(cfg Config) *Manager {
	return &Manager{
		enabled: cfg.Enabled,
		config:  cfg,
	}
}

// Initialize sets up the OpenTelemetry providers and exporters.
// This method creates the OTLP gRPC exporter, configures the TracerProvider
// with batch span processing and sampling, sets resource attributes, and
// registers the global tracer provider.
//
// If initialization fails, the manager logs a warning and disables tracing
// to allow the application to continue operating without telemetry.
//
// Parameters:
//   - ctx: Context for initialization operations
//
// Returns an error if initialization fails (though the manager will continue
// in disabled mode).
func (m *Manager) Initialize(ctx context.Context) error {
	// Skip initialization if not enabled
	if !m.config.Enabled {
		logrus.Debug("OpenTelemetry is disabled in configuration")
		return nil
	}

	// Create OTLP gRPC exporter with configured endpoint and TLS settings
	exporter, err := m.createExporter(ctx)
	if err != nil {
		logrus.Warnf("Failed to initialize OpenTelemetry: %v. Continuing without tracing.", err)
		m.enabled = false
		return nil // Don't fail startup
	}

	// Create resource with service attributes
	res, err := m.createResource()
	if err != nil {
		logrus.Warnf("Failed to create OpenTelemetry resource: %v. Continuing without tracing.", err)
		m.enabled = false
		return nil
	}

	// Configure sampling based on sampling rate
	sampler := m.createSampler()

	// Create TracerProvider with batch span processor
	m.tracerProvider = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	// Register global tracer provider
	otel.SetTracerProvider(m.tracerProvider)

	logrus.Infof("OpenTelemetry initialized successfully (endpoint: %s, sampling: %.2f)",
		m.config.Endpoint, m.config.SamplingRate)

	return nil
}

// createExporter creates an OTLP gRPC exporter with the configured endpoint and TLS settings.
func (m *Manager) createExporter(ctx context.Context) (sdktrace.SpanExporter, error) {
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(m.config.Endpoint),
	}

	// Configure TLS settings
	if m.config.Insecure {
		opts = append(opts, otlptracegrpc.WithTLSCredentials(insecure.NewCredentials()))
	}

	exporter, err := otlptracegrpc.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	return exporter, nil
}

// createResource creates a resource with service and host attributes.
func (m *Manager) createResource() (*resource.Resource, error) {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	// Build resource attributes
	attrs := []resource.Option{
		resource.WithAttributes(
			semconv.ServiceNameKey.String(m.config.ServiceName),
			semconv.ServiceVersionKey.String(m.config.ServiceVersion),
			semconv.HostNameKey.String(hostname),
		),
	}

	// Add NetBackup server as custom attribute if provided
	if m.config.NetBackupServer != "" {
		attrs = append(attrs, resource.WithAttributes(
			semconv.PeerServiceKey.String(m.config.NetBackupServer),
		))
	}

	return resource.New(context.Background(), attrs...)
}

// createSampler creates a sampler based on the configured sampling rate.
func (m *Manager) createSampler() sdktrace.Sampler {
	if m.config.SamplingRate >= 1.0 {
		// Sample all traces
		return sdktrace.AlwaysSample()
	}
	// Sample based on trace ID ratio
	return sdktrace.TraceIDRatioBased(m.config.SamplingRate)
}

// Shutdown flushes pending telemetry data and cleans up resources.
// This method should be called during application shutdown to ensure
// all spans are exported before the application terminates.
//
// Parameters:
//   - ctx: Context with timeout for shutdown operations
//
// Returns an error if shutdown fails.
func (m *Manager) Shutdown(ctx context.Context) error {
	// Skip shutdown if not enabled or not initialized
	if !m.enabled || m.tracerProvider == nil {
		logrus.Debug("OpenTelemetry shutdown skipped (not enabled or not initialized)")
		return nil
	}

	logrus.Info("Shutting down OpenTelemetry TracerProvider...")

	// Call TracerProvider.Shutdown() to flush pending spans
	if err := m.tracerProvider.Shutdown(ctx); err != nil {
		logrus.Errorf("Error during OpenTelemetry shutdown: %v", err)
		return fmt.Errorf("failed to shutdown TracerProvider: %w", err)
	}

	logrus.Info("OpenTelemetry shutdown completed successfully")
	return nil
}

// IsEnabled returns whether OpenTelemetry tracing is currently enabled.
// This can be false if tracing was disabled in configuration or if
// initialization failed.
//
// Returns true if tracing is enabled and operational, false otherwise.
func (m *Manager) IsEnabled() bool {
	return m.enabled
}

// TracerProvider returns the configured TracerProvider for explicit injection.
// Returns nil if telemetry is not initialized or disabled.
//
// This method enables dependency injection of the TracerProvider to components
// that need distributed tracing, avoiding global state access.
//
// Returns the TracerProvider instance, or nil if not available.
func (m *Manager) TracerProvider() trace.TracerProvider {
	return m.tracerProvider
}
