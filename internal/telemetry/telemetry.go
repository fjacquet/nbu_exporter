// Package telemetry provides OpenTelemetry integration for the NBU Exporter.
//
// This package manages the lifecycle of OpenTelemetry tracing and provides
// instrumentation for NetBackup API calls and Prometheus scrape cycles.
//
// # Key Components
//
// Manager: Handles OpenTelemetry initialization, lifecycle management, and shutdown.
// The Manager centralizes TracerProvider configuration and ensures proper resource cleanup.
//
// Attributes: Centralized span attribute constants organized by category (HTTP, NetBackup, Scrape).
// Using constants prevents typos and enables IDE autocomplete.
//
// Error Templates: Reusable error message templates for common failure scenarios with
// actionable troubleshooting steps.
//
// # Usage Example
//
// Initializing telemetry:
//
//	cfg := telemetry.Config{
//	    Enabled:         true,
//	    Endpoint:        "localhost:4317",
//	    Insecure:        true,
//	    SamplingRate:    1.0,
//	    ServiceName:     "nbu-exporter",
//	    ServiceVersion:  "1.0.0",
//	    NetBackupServer: "nbu-master",
//	}
//	manager := telemetry.NewManager(cfg)
//	if err := manager.Initialize(ctx); err != nil {
//	    log.Fatalf("Failed to initialize telemetry: %v", err)
//	}
//	defer manager.Shutdown(ctx)
//
// Using span attributes:
//
//	span.SetAttributes(
//	    attribute.String(telemetry.AttrHTTPMethod, "GET"),
//	    attribute.String(telemetry.AttrHTTPURL, url),
//	    attribute.Int(telemetry.AttrHTTPStatusCode, 200),
//	)
//
// Using error templates:
//
//	if resp.StatusCode() == http.StatusNotAcceptable {
//	    return fmt.Errorf(telemetry.ErrAPIVersionNotSupportedTemplate,
//	        apiVersion, apiVersion, url)
//	}
//
// # Design Patterns
//
// Graceful Degradation: If OpenTelemetry initialization fails, the manager
// disables tracing and allows the application to continue without telemetry.
// This ensures monitoring failures don't impact core functionality.
//
// Centralized Constants: All span attribute keys are defined as package-level
// constants to prevent typos, enable compile-time validation, and improve
// maintainability.
//
// Batch Attribute Recording: Span attributes should be batched in single
// SetAttributes() calls for better performance.
//
// # Configuration
//
// The telemetry package supports the following configuration options:
//
//   - Enabled: Toggle tracing on/off
//   - Endpoint: OTLP gRPC collector endpoint (e.g., "localhost:4317")
//   - Insecure: Use insecure connection (no TLS)
//   - SamplingRate: Percentage of traces to sample (0.0 to 1.0)
//   - ServiceName: Service name for resource attributes
//   - ServiceVersion: Service version for resource attributes
//   - NetBackupServer: Target NetBackup server for resource attributes
//
// # Sampling Strategies
//
// The package supports two sampling strategies:
//
//   - AlwaysSample: Sample all traces (SamplingRate = 1.0)
//   - TraceIDRatioBased: Sample based on trace ID ratio (SamplingRate < 1.0)
//
// Use lower sampling rates in high-volume production environments to reduce
// overhead and storage costs.
package telemetry
