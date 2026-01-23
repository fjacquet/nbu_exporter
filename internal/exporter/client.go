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
	"sync"
	"sync/atomic"
	"time"

	"github.com/fjacquet/nbu_exporter/internal/logging"
	"github.com/fjacquet/nbu_exporter/internal/models"
	"github.com/fjacquet/nbu_exporter/internal/telemetry"
	"github.com/go-resty/resty/v2"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const (
	defaultTimeout        = 1 * time.Minute    // Default timeout for HTTP requests
	contentType           = "application/json" // Content type for API requests
	httpContentTypeHeader = "Content-Type"     // HTTP header name for content type

	// Retry configuration
	retryCount       = 3                // Number of retry attempts
	retryWaitTime    = 5 * time.Second  // Initial wait time between retries
	retryMaxWaitTime = 60 * time.Second // Maximum wait time between retries

	// Connection pool configuration
	maxIdleConns        = 100              // Total idle connections across all hosts
	maxIdleConnsPerHost = 20               // Idle connections per host (default is 2, too low)
	idleConnTimeout     = 90 * time.Second // Timeout for idle connections
)

// ClientOption configures optional NbuClient settings.
type ClientOption func(*clientOptions)

type clientOptions struct {
	tracerProvider trace.TracerProvider
}

func defaultClientOptions() clientOptions {
	return clientOptions{
		tracerProvider: nil, // Will use noop via TracerWrapper
	}
}

// WithTracerProvider sets the TracerProvider for distributed tracing.
// If not provided, tracing operations use a noop provider (no overhead).
func WithTracerProvider(tp trace.TracerProvider) ClientOption {
	return func(o *clientOptions) {
		o.tracerProvider = tp
	}
}

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
	client  *resty.Client   // HTTP client with TLS configuration
	cfg     models.Config   // Application configuration including API settings
	tracing *TracerWrapper  // OpenTelemetry tracer wrapper for nil-safe distributed tracing

	// Connection tracking for graceful shutdown
	mu         sync.Mutex    // Protects closed and closeChan
	activeReqs int32         // Count of active requests (atomic)
	closed     bool          // Whether Close() has been called
	closeChan  chan struct{} // Signaled when all requests complete
}

// NewNbuClient creates a new NetBackup API client with the provided configuration.
// It initializes the HTTP client with appropriate TLS settings and timeout values.
// TracerProvider can be injected via WithTracerProvider option for distributed tracing.
//
// The client is configured with:
//   - TLS verification based on cfg.NbuServer.InsecureSkipVerify
//   - Default timeout of 1 minute for all requests
//   - Optional OpenTelemetry tracer via options
//
// Example:
//
//	cfg := models.Config{...}
//	client := NewNbuClient(cfg)  // Without tracing
//	client := NewNbuClient(cfg, WithTracerProvider(tp))  // With tracing
func NewNbuClient(cfg models.Config, opts ...ClientOption) *NbuClient {
	// Apply options
	options := defaultClientOptions()
	for _, opt := range opts {
		opt(&options)
	}

	// Log security warning if TLS verification is disabled
	if cfg.NbuServer.InsecureSkipVerify {
		log.Error("SECURITY WARNING: TLS certificate verification disabled - this is insecure for production use")
	}

	client := resty.New().
		SetTimeout(defaultTimeout).
		// Configure retry with exponential backoff
		SetRetryCount(retryCount).
		SetRetryWaitTime(retryWaitTime).
		SetRetryMaxWaitTime(retryMaxWaitTime).
		AddRetryCondition(func(r *resty.Response, err error) bool {
			// Retry on network errors
			if err != nil {
				return true
			}
			// Retry on rate limiting (429) and server errors (5xx)
			return r.StatusCode() == http.StatusTooManyRequests ||
				r.StatusCode() >= 500
		})

	// Enable automatic Retry-After header handling for 429 responses
	client.AddRetryAfterErrorCondition()

	// Configure connection pool and TLS in http.Transport for unified config
	httpClient := client.GetClient()
	httpClient.Transport = &http.Transport{
		MaxIdleConns:        maxIdleConns,
		MaxIdleConnsPerHost: maxIdleConnsPerHost,
		IdleConnTimeout:     idleConnTimeout,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: cfg.NbuServer.InsecureSkipVerify,
			MinVersion:         tls.VersionTLS12, // Enforce TLS 1.2 minimum
		},
	}

	// Create TracerWrapper with injected provider (uses noop if nil)
	tracing := NewTracerWrapper(options.tracerProvider, "nbu-exporter/http-client")

	return &NbuClient{
		client:     client,
		cfg:        cfg,
		tracing:    tracing,
		activeReqs: 0,
		closed:     false,
	}
}

// NewNbuClientWithVersionDetection creates a new NetBackup API client and automatically
// detects the API version if not explicitly configured. This is the recommended way to
// create a client when you want automatic version detection.
//
// The function:
//   - Creates a new HTTP client with the provided configuration and options
//   - If apiVersion is not set in config, performs automatic version detection
//   - Updates the configuration with the detected version
//   - Returns the configured client ready for use
//
// Parameters:
//   - ctx: Context for version detection requests (supports cancellation and timeout)
//   - cfg: Application configuration (will be modified if version detection occurs)
//   - opts: Optional ClientOption for configuration (e.g., WithTracerProvider)
//
// Returns:
//   - Configured NbuClient with detected or configured API version
//   - Error if version detection fails or configuration is invalid
//
// Example:
//
//	cfg := models.Config{...}
//	client, err := NewNbuClientWithVersionDetection(ctx, &cfg, WithTracerProvider(tp))
//	if err != nil {
//	    log.Fatalf("Failed to create client: %v", err)
//	}
func NewNbuClientWithVersionDetection(ctx context.Context, cfg *models.Config, opts ...ClientOption) (*NbuClient, error) {
	// Create the base client with options
	client := NewNbuClient(*cfg, opts...)

	// Perform version detection if needed
	if err := performVersionDetectionIfNeeded(ctx, client, cfg); err != nil {
		return nil, err
	}

	return client, nil
}

// performVersionDetectionIfNeeded handles version detection logic for client creation.
// This function is extracted to avoid duplication between NewNbuClientWithVersionDetection
// and NewNbuCollector.
//
// The function creates an immutable detector with only the values needed for detection.
// Config mutation happens ONLY after successful detection, in a single location.
// If detection fails or context is cancelled, config remains unchanged.
//
// Parameters:
//   - ctx: Context for version detection requests
//   - client: The NbuClient instance to use for detection
//   - cfg: Application configuration (will be modified if version detection occurs)
//
// Returns an error if version detection fails.
func performVersionDetectionIfNeeded(ctx context.Context, client *NbuClient, cfg *models.Config) error {
	if shouldPerformVersionDetection(cfg) {
		logging.LogInfo("API version not configured, performing automatic detection")

		// Create detector with immutable values - does not mutate config
		detector := NewAPIVersionDetector(
			client,
			cfg.GetNBUBaseURL(),
			cfg.NbuServer.APIKey,
		)

		detectedVersion, err := detector.DetectVersion(ctx)
		if err != nil {
			return fmt.Errorf("automatic API version detection failed: %w", err)
		}

		// Single point of config mutation - only after successful detection
		cfg.NbuServer.APIVersion = detectedVersion
		client.cfg.NbuServer.APIVersion = detectedVersion
		logging.LogInfo(fmt.Sprintf("Automatically detected API version: %s", detectedVersion))
	} else if isExplicitVersionConfigured(cfg) {
		logging.LogInfo(fmt.Sprintf("Using explicitly configured API version: %s (bypassing detection)", cfg.NbuServer.APIVersion))
	} else {
		logging.LogInfo(fmt.Sprintf("Using configured API version: %s", cfg.NbuServer.APIVersion))
	}

	return nil
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
// SECURITY: The API key is included in the Authorization header. This value should
// never be logged or included in error messages. Use Config.MaskAPIKey() if key
// context is needed for debugging.
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
	// Check if client is closed
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return fmt.Errorf("client is closed")
	}
	atomic.AddInt32(&c.activeReqs, 1)
	c.mu.Unlock()

	// Track request completion
	defer func() {
		if atomic.AddInt32(&c.activeReqs, -1) == 0 {
			c.mu.Lock()
			if c.closed && c.closeChan != nil {
				close(c.closeChan)
				c.closeChan = nil
			}
			c.mu.Unlock()
		}
	}()

	// Create span for HTTP request using TracerWrapper
	ctx, span := c.tracing.StartSpan(ctx, "http.request", trace.SpanKindClient)
	defer span.End()

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
		// Include URL, status code, and content-type in error message
		contentTypeValue := resp.Header().Get(httpContentTypeHeader)
		err := fmt.Errorf("HTTP request failed: url=%s, status=%d (%s), content-type=%s",
			url, resp.StatusCode(), resp.Status(), contentTypeValue)
		c.recordError(span, err)
		return err
	}

	// Validate Content-Type before attempting to unmarshal
	contentType := resp.Header().Get(httpContentTypeHeader)
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
		// Include URL, status code, and content-type in structured error message
		// Keep "instead of JSON" for backward compatibility with tests
		err := fmt.Errorf("server returned %s instead of JSON: url=%s, status=%d, preview=%s",
			contentType, url, resp.StatusCode(), bodyPreview)
		c.recordError(span, err)
		return err
	}

	if err := json.Unmarshal(resp.Body(), target); err != nil {
		// Provide more context for JSON unmarshaling errors including URL, status code, and content-type
		bodyPreview := string(resp.Body())
		if len(bodyPreview) > 200 {
			bodyPreview = bodyPreview[:200] + "..."
		}
		contentTypeValue := resp.Header().Get(httpContentTypeHeader)
		unmarshalErr := fmt.Errorf("failed to unmarshal JSON response: url=%s, status=%d, content-type=%s, error=%w\nResponse preview: %s",
			url, resp.StatusCode(), contentTypeValue, err, bodyPreview)
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
		return "", fmt.Errorf("failed to detect API version: url=%s, error=%w", testURL, err)
	}

	// Check for version-specific error responses
	if resp.StatusCode() == 406 {
		// 406 Not Acceptable typically means the API version is not supported
		contentTypeValue := resp.Header().Get(httpContentTypeHeader)
		return "", fmt.Errorf("API version %s not supported by NetBackup server: url=%s, status=406, content-type=%s",
			c.cfg.NbuServer.APIVersion, testURL, contentTypeValue)
	}

	if resp.IsError() {
		// Other errors might indicate connectivity issues, not version problems
		contentTypeValue := resp.Header().Get(httpContentTypeHeader)
		return "", fmt.Errorf("API connectivity test failed: url=%s, status=%d (%s), content-type=%s",
			testURL, resp.StatusCode(), resp.Status(), contentTypeValue)
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
// TracerWrapper ensures this is always safe to call (uses noop if tracing disabled).
//
// Parameters:
//   - ctx: Context containing the trace information
//   - headers: Map of HTTP headers to inject trace context into
//
// Returns the headers map with trace context injected
func (c *NbuClient) injectTraceContext(ctx context.Context, headers map[string]string) map[string]string {
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
// It waits for active requests to complete (up to 30 seconds)
// before closing connections.
//
// Returns an error if:
//   - The client is already closed
//   - Timeout exceeded while waiting for active requests
func (c *NbuClient) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return fmt.Errorf("client already closed")
	}
	c.closed = true

	// Check if there are active requests
	activeCount := atomic.LoadInt32(&c.activeReqs)
	if activeCount > 0 {
		c.closeChan = make(chan struct{})
		ch := c.closeChan // Store local reference to avoid race
		c.mu.Unlock()

		// Wait for active requests with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		select {
		case <-ch:
			log.Debug("All active requests completed during shutdown")
		case <-ctx.Done():
			log.Warnf("Timeout waiting for %d active requests during shutdown", activeCount)
		}
	} else {
		c.mu.Unlock()
	}

	// Close idle connections
	if c.client != nil {
		c.client.GetClient().CloseIdleConnections()
		c.client = nil
	}

	return nil
}

// CloseWithContext releases resources with explicit timeout control.
// Use this when you need custom shutdown timeout behavior.
//
// Parameters:
//   - ctx: Context for shutdown timeout
//
// Returns an error if:
//   - The client is already closed
//   - Context is cancelled while waiting for active requests
func (c *NbuClient) CloseWithContext(ctx context.Context) error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return fmt.Errorf("client already closed")
	}
	c.closed = true

	activeCount := atomic.LoadInt32(&c.activeReqs)
	if activeCount > 0 {
		c.closeChan = make(chan struct{})
		ch := c.closeChan // Store local reference to avoid race
		c.mu.Unlock()

		select {
		case <-ch:
			log.Debug("All active requests completed during shutdown")
		case <-ctx.Done():
			log.Warnf("Context cancelled while waiting for %d active requests", activeCount)
			return ctx.Err()
		}
	} else {
		c.mu.Unlock()
	}

	if c.client != nil {
		c.client.GetClient().CloseIdleConnections()
		c.client = nil
	}

	return nil
}
