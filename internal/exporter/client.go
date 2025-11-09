// Package exporter provides HTTP client functionality and data fetching logic
// for the NetBackup REST API. It handles API communication, pagination,
// and metric collection for Prometheus exposition.
package exporter

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/fjacquet/nbu_exporter/internal/logging"
	"github.com/fjacquet/nbu_exporter/internal/models"
	"github.com/fjacquet/nbu_exporter/internal/telemetry"
	"github.com/go-resty/resty/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const (
	defaultTimeout = 1 * time.Minute    // Default timeout for HTTP requests
	contentType    = "application/json" // Content type for API requests
)

// HTTP header names used in NetBackup API requests.
const (
	HeaderAccept        = "Accept"        // Accept header for content negotiation
	HeaderAuthorization = "Authorization" // Authorization header for API key authentication
)

// Query parameter names used in NetBackup API pagination and filtering.
const (
	QueryParamLimit  = "page[limit]"  // Maximum number of results per page
	QueryParamOffset = "page[offset]" // Starting offset for pagination
	QueryParamSort   = "sort"         // Field to sort results by
	QueryParamFilter = "filter"       // Filter expression for result filtering
)

// NbuClient handles HTTP communication with the NetBackup REST API.
// It manages TLS configuration, request headers, and provides methods for
// fetching data from various NetBackup API endpoints.
type NbuClient struct {
	client *resty.Client // HTTP client with TLS configuration
	cfg    models.Config // Application configuration including API settings
	tracer trace.Tracer  // OpenTelemetry tracer for distributed tracing (nil if tracing disabled)
}

// NewNbuClient creates a new NetBackup API client with the provided configuration.
// It initializes the HTTP client with appropriate TLS settings and timeout values.
// If OpenTelemetry is enabled globally, the client will automatically initialize
// a tracer for distributed tracing of HTTP requests.
//
// The client is configured with:
//   - TLS verification based on cfg.NbuServer.InsecureSkipVerify
//   - Default timeout of 1 minute for all requests
//   - Optional OpenTelemetry tracer from global provider
//
// Example:
//
//	cfg := models.Config{...}
//	client := NewNbuClient(cfg)
func NewNbuClient(cfg models.Config) *NbuClient {
	client := resty.New().
		SetTLSClientConfig(&tls.Config{
			InsecureSkipVerify: cfg.NbuServer.InsecureSkipVerify,
		}).
		SetTimeout(defaultTimeout)

	// Initialize tracer from global provider if available
	var tracer trace.Tracer
	tracerProvider := otel.GetTracerProvider()
	if tracerProvider != nil {
		tracer = tracerProvider.Tracer("nbu-exporter/http-client")
	}

	return &NbuClient{
		client: client,
		cfg:    cfg,
		tracer: tracer,
	}
}

// NewNbuClientWithVersionDetection creates a new NetBackup API client and automatically
// detects the API version if not explicitly configured. This is the recommended way to
// create a client when you want automatic version detection.
//
// The function:
//   - Creates a new HTTP client with the provided configuration
//   - If apiVersion is not set in config, performs automatic version detection
//   - Updates the configuration with the detected version
//   - Returns the configured client ready for use
//
// Parameters:
//   - ctx: Context for version detection requests (supports cancellation and timeout)
//   - cfg: Application configuration (will be modified if version detection occurs)
//
// Returns:
//   - Configured NbuClient with detected or configured API version
//   - Error if version detection fails or configuration is invalid
//
// Example:
//
//	cfg := models.Config{...}
//	client, err := NewNbuClientWithVersionDetection(ctx, &cfg)
//	if err != nil {
//	    log.Fatalf("Failed to create client: %v", err)
//	}
func NewNbuClientWithVersionDetection(ctx context.Context, cfg *models.Config) (*NbuClient, error) {
	// Create the base client
	client := NewNbuClient(*cfg)

	// If API version is not explicitly configured, perform version detection
	if shouldPerformVersionDetection(cfg) {
		logging.LogInfo("API version not configured, performing automatic detection")
		detector := NewAPIVersionDetector(client, cfg)
		detectedVersion, err := detector.DetectVersion(ctx)
		if err != nil {
			return nil, fmt.Errorf("automatic API version detection failed: %w", err)
		}

		// Update configuration with detected version
		cfg.NbuServer.APIVersion = detectedVersion
		client.cfg.NbuServer.APIVersion = detectedVersion
		logging.LogInfo(fmt.Sprintf("Automatically detected API version: %s", detectedVersion))
	} else if isExplicitVersionConfigured(cfg) {
		logging.LogInfo(fmt.Sprintf("Using explicitly configured API version: %s (bypassing detection)", cfg.NbuServer.APIVersion))
	} else {
		logging.LogInfo(fmt.Sprintf("Using configured API version: %s", cfg.NbuServer.APIVersion))
	}

	return client, nil
}

// shouldPerformVersionDetection determines if automatic API version detection is needed.
// Returns true when the API version is not configured in the config file.
func shouldPerformVersionDetection(cfg *models.Config) bool {
	return cfg.NbuServer.APIVersion == ""
}

// isExplicitVersionConfigured checks if the user explicitly configured an API version.
// Returns true when a non-default API version is configured, indicating the user
// intentionally set a specific version to bypass automatic detection.
func isExplicitVersionConfigured(cfg *models.Config) bool {
	return cfg.NbuServer.APIVersion != "" && cfg.NbuServer.APIVersion != models.APIVersion130
}

// getHeaders returns the standard HTTP headers required for NetBackup API requests.
// It constructs a versioned Accept header for API version negotiation and includes
// the API key for authentication.
//
// The Accept header format follows NetBackup's versioned media type convention:
//
//	application/vnd.netbackup+json;version=<apiVersion>
//
// Returns a map containing:
//   - Accept: Versioned content type header for API version negotiation
//   - Authorization: API key for authentication
//
// This method is called internally by FetchData before each HTTP request.
func (c *NbuClient) getHeaders() map[string]string {
	// Construct versioned Accept header for NetBackup API 10.5+
	acceptHeader := fmt.Sprintf("application/vnd.netbackup+json;version=%s", c.cfg.NbuServer.APIVersion)

	return map[string]string{
		HeaderAccept:        acceptHeader,
		HeaderAuthorization: c.cfg.NbuServer.APIKey,
	}
}

// FetchData sends an HTTP GET request to the specified URL and unmarshals the JSON response
// into the provided target interface. It handles API version negotiation, error responses,
// and provides detailed error messages for common failure scenarios.
//
// When OpenTelemetry tracing is enabled, this method creates a span for the HTTP request
// and records relevant attributes including method, URL, status code, and duration.
//
// Parameters:
//   - ctx: Context for request cancellation, timeout, and trace propagation
//   - url: Complete URL to fetch (including query parameters)
//   - target: Pointer to struct where JSON response will be unmarshaled
//
// Returns an error if:
//   - HTTP request fails (network error, timeout)
//   - Server returns non-2xx status code
//   - API version is not supported (HTTP 406)
//   - JSON unmarshaling fails
//
// Example:
//
//	var jobs models.Jobs
//	err := client.FetchData(ctx, "https://nbu:1556/admin/jobs", &jobs)
func (c *NbuClient) FetchData(ctx context.Context, url string, target interface{}) error {
	// Create span for HTTP request using consolidated helper
	ctx, span := createSpan(ctx, c.tracer, "http.request", trace.SpanKindClient)
	if span != nil {
		defer span.End()
	}

	// Record start time for duration calculation
	startTime := time.Now()

	// Get headers and inject trace context
	headers := c.getHeaders()
	headers = c.injectTraceContext(ctx, headers)

	// Make HTTP request
	resp, err := c.client.R().
		SetContext(ctx).
		SetHeaders(headers).
		Get(url)

	// Calculate duration
	duration := time.Since(startTime)

	if err != nil {
		// Record error on span
		c.recordError(span, err)
		return fmt.Errorf("HTTP request to %s failed: %w", url, err)
	}

	// Record HTTP attributes
	requestSize := int64(0) // GET requests typically have no body
	responseSize := int64(len(resp.Body()))
	c.recordHTTPAttributes(span, http.MethodGet, url, resp.StatusCode(), requestSize, responseSize, duration)

	if resp.IsError() {
		// Handle 406 Not Acceptable - API version not supported
		if resp.StatusCode() == 406 {
			errMsg := fmt.Sprintf(
				telemetry.ErrAPIVersionNotSupportedTemplate,
				c.cfg.NbuServer.APIVersion,
				c.cfg.NbuServer.APIVersion,
				url,
			)
			logging.LogError(errMsg)
			err := fmt.Errorf("%s", errMsg)
			c.recordError(span, err)
			return err
		}

		err := fmt.Errorf("HTTP request to %s returned status %d: %s", url, resp.StatusCode(), resp.Status())
		c.recordError(span, err)
		return err
	}

	// Validate Content-Type before attempting to unmarshal
	contentType := resp.Header().Get("Content-Type")
	if contentType != "" && !strings.Contains(contentType, "application/json") && !strings.Contains(contentType, "application/vnd.netbackup+json") {
		// Server returned non-JSON content (likely HTML error page)
		bodyPreview := string(resp.Body())
		if len(bodyPreview) > 200 {
			bodyPreview = bodyPreview[:200] + "..."
		}

		errMsg := fmt.Sprintf(
			telemetry.ErrNonJSONResponseTemplate,
			contentType,
			url,
			bodyPreview,
		)
		logging.LogError(errMsg)
		err := fmt.Errorf("server returned %s instead of JSON: %s", contentType, bodyPreview)
		c.recordError(span, err)
		return err
	}

	if err := json.Unmarshal(resp.Body(), target); err != nil {
		// Provide more context for JSON unmarshaling errors
		bodyPreview := string(resp.Body())
		if len(bodyPreview) > 200 {
			bodyPreview = bodyPreview[:200] + "..."
		}
		unmarshalErr := fmt.Errorf("failed to unmarshal JSON response from %s: %w\nResponse preview: %s", url, err, bodyPreview)
		c.recordError(span, unmarshalErr)
		return unmarshalErr
	}

	// Set span status to OK for successful requests
	if span != nil {
		span.SetStatus(codes.Ok, "Request completed successfully")
	}

	return nil
}

// DetectAPIVersion attempts to detect and validate the NetBackup API version by making
// a lightweight test request to the jobs endpoint. This helps identify API compatibility
// issues early in the application lifecycle.
//
// The method tests connectivity using the configured API version and returns:
//   - The configured API version string if successful
//   - An error if the version is not supported or connectivity fails
//
// Common error scenarios:
//   - HTTP 406: API version not supported by the NetBackup server
//   - Network errors: Connectivity issues with the NetBackup server
//
// This method is typically called during collector initialization to provide early
// feedback about API compatibility issues.
//
// Example:
//
//	version, err := client.DetectAPIVersion(ctx)
//	if err != nil {
//	    log.Warnf("API version detection failed: %v", err)
//	}
func (c *NbuClient) DetectAPIVersion(ctx context.Context) (string, error) {
	// Use a lightweight endpoint to test API connectivity
	// We'll use the jobs endpoint with a very small limit
	baseURL := c.cfg.GetNBUBaseURL()
	testURL := fmt.Sprintf("%s/admin/jobs?page[limit]=1", baseURL)

	resp, err := c.client.R().
		SetContext(ctx).
		SetHeaders(c.getHeaders()).
		Get(testURL)

	if err != nil {
		return "", fmt.Errorf("failed to detect API version: %w", err)
	}

	// Check for version-specific error responses
	if resp.StatusCode() == 406 {
		// 406 Not Acceptable typically means the API version is not supported
		return "", fmt.Errorf("API version %s not supported by NetBackup server (HTTP 406)", c.cfg.NbuServer.APIVersion)
	}

	if resp.IsError() {
		// Other errors might indicate connectivity issues, not version problems
		return "", fmt.Errorf("API connectivity test failed with status %d: %s", resp.StatusCode(), resp.Status())
	}

	// If we get here, the configured API version is working
	return c.cfg.NbuServer.APIVersion, nil
}

// recordHTTPAttributes records HTTP semantic convention attributes on the span.
// This method follows OpenTelemetry HTTP semantic conventions for consistent
// attribute naming across different tracing backends.
//
// Parameters:
//   - span: The span to record attributes on (nil-safe)
//   - method: HTTP method (e.g., "GET", "POST")
//   - url: Full request URL
//   - statusCode: HTTP response status code
//   - requestSize: Size of request body in bytes (0 if no body)
//   - responseSize: Size of response body in bytes
//   - duration: Request duration
func (c *NbuClient) recordHTTPAttributes(span trace.Span, method, url string, statusCode int, requestSize, responseSize int64, duration time.Duration) {
	// Nil-safe check: if span is nil, do nothing
	if span == nil {
		return
	}

	// Record HTTP semantic convention attributes using centralized constants
	span.SetAttributes(
		attribute.String(telemetry.AttrHTTPMethod, method),
		attribute.String(telemetry.AttrHTTPURL, url),
		attribute.Int(telemetry.AttrHTTPStatusCode, statusCode),
		attribute.Int64(telemetry.AttrHTTPRequestContentLength, requestSize),
		attribute.Int64(telemetry.AttrHTTPResponseContentLength, responseSize),
		attribute.Float64(telemetry.AttrHTTPDurationMS, float64(duration.Milliseconds())),
	)
}

// recordError records an error on the span and sets the span status to error.
// This method follows OpenTelemetry conventions for error recording.
//
// Parameters:
//   - span: The span to record the error on (nil-safe)
//   - err: The error to record
func (c *NbuClient) recordError(span trace.Span, err error) {
	// Nil-safe check: if span is nil, do nothing
	if span == nil {
		return
	}

	// Record error as span event
	span.RecordError(err)

	// Set span status to error with error message
	span.SetStatus(codes.Error, err.Error())

	// Add error attribute using centralized constant
	span.SetAttributes(
		attribute.String(telemetry.AttrError, err.Error()),
	)
}

// injectTraceContext injects trace context into HTTP request headers using W3C Trace Context propagation.
// This enables distributed tracing across service boundaries.
//
// Parameters:
//   - ctx: Context containing the trace information
//   - headers: Map of HTTP headers to inject trace context into
//
// Returns the headers map with trace context injected (if tracing is enabled)
func (c *NbuClient) injectTraceContext(ctx context.Context, headers map[string]string) map[string]string {
	// If tracer is not initialized, return headers unchanged
	if c.tracer == nil {
		return headers
	}

	// Create a carrier that implements the TextMapCarrier interface
	carrier := propagation.MapCarrier{}
	for k, v := range headers {
		carrier.Set(k, v)
	}

	// Inject trace context into the carrier using the global propagator
	otel.GetTextMapPropagator().Inject(ctx, carrier)

	// Convert carrier back to map
	result := make(map[string]string)
	for k, v := range carrier {
		result[k] = v
	}

	return result
}

// Close releases resources associated with the HTTP client.
// Note: Resty doesn't provide an explicit close method, so this clears the client reference
// to allow garbage collection. Connection pooling is managed by the underlying http.Client.
func (c *NbuClient) Close() {
	// Resty doesn't have an explicit close, but we can clear the client
	c.client = nil
}
