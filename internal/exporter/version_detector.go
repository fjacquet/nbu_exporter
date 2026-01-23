// Package exporter provides API version detection functionality for NetBackup.
// It implements automatic version detection with fallback logic to support
// multiple NetBackup versions (10.0, 10.5, and 11.0).
package exporter

import (
	"context"
	"fmt"
	"time"

	"github.com/fjacquet/nbu_exporter/internal/models"
	"github.com/go-resty/resty/v2"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// RetryConfig defines the configuration for retry logic with exponential backoff.
// It controls how many times to retry transient failures and the delay between attempts.
type RetryConfig struct {
	MaxAttempts   int           // Maximum number of retry attempts
	InitialDelay  time.Duration // Initial delay before first retry
	MaxDelay      time.Duration // Maximum delay between retries
	BackoffFactor float64       // Multiplier for exponential backoff
}

// DefaultRetryConfig provides sensible defaults for retry behavior.
// - 3 attempts total (initial + 2 retries)
// - 1 second initial delay
// - 10 second maximum delay
// - 2x backoff factor (exponential)
var DefaultRetryConfig = RetryConfig{
	MaxAttempts:   3,
	InitialDelay:  1 * time.Second,
	MaxDelay:      10 * time.Second,
	BackoffFactor: 2.0,
}

// APIVersionDetector handles automatic detection of supported NetBackup API versions.
// It tests versions in descending order (13.0 → 12.0 → 3.0) and returns the first
// working version. This allows the exporter to work with NetBackup 10.0, 10.5, and 11.0
// without manual configuration.
//
// The detector is immutable - it does not modify any configuration during detection.
// It returns the detected version string, and the caller is responsible for applying
// the detected version to the configuration.
type APIVersionDetector struct {
	client      *NbuClient   // HTTP client for making test requests
	baseURL     string       // NetBackup base URL (immutable)
	apiKey      string       // API key for authentication (immutable)
	retryConfig RetryConfig  // Retry configuration for transient failures
	tracer      trace.Tracer // OpenTelemetry tracer for distributed tracing (nil if tracing disabled)
}

// NewAPIVersionDetector creates a new version detector with the provided client and immutable configuration values.
// It uses the default retry configuration for handling transient failures.
//
// Parameters:
//   - client: Configured NetBackup API client
//   - baseURL: NetBackup base URL (e.g., "https://netbackup:1556/netbackup")
//   - apiKey: API key for authentication
//
// Returns a new APIVersionDetector instance.
func NewAPIVersionDetector(client *NbuClient, baseURL, apiKey string) *APIVersionDetector {
	// Initialize tracer from global provider if available
	var tracer trace.Tracer
	tracerProvider := otel.GetTracerProvider()
	if tracerProvider != nil {
		tracer = tracerProvider.Tracer("nbu-exporter/version-detector")
	}

	return &APIVersionDetector{
		client:      client,
		baseURL:     baseURL,
		apiKey:      apiKey,
		retryConfig: DefaultRetryConfig,
		tracer:      tracer,
	}
}

// DetectVersion attempts to detect the highest supported API version by testing
// versions in descending order (13.0 → 12.0 → 3.0). It returns the first version
// that successfully responds to a test request.
//
// The detection process:
// 1. Try API version 13.0 (NetBackup 11.0)
// 2. If HTTP 406, try API version 12.0 (NetBackup 10.5)
// 3. If HTTP 406, try API version 3.0 (NetBackup 10.0-10.4)
// 4. If all fail, return detailed error with troubleshooting steps
//
// Authentication errors (HTTP 401) cause immediate failure as they indicate
// a configuration problem, not a version compatibility issue.
//
// Parameters:
//   - ctx: Context for request cancellation and timeout
//
// Returns:
//   - Detected API version string (e.g., "13.0")
//   - Error if no compatible version is found or authentication fails
//
// Example:
//
//	detector := NewAPIVersionDetector(client, &cfg)
//	version, err := detector.DetectVersion(ctx)
//	if err != nil {
//	    log.Fatalf("Version detection failed: %v", err)
//	}
//	log.Infof("Detected API version: %s", version)
func (d *APIVersionDetector) DetectVersion(ctx context.Context) (string, error) {
	// Create span for version detection
	ctx, span := d.createVersionDetectionSpan(ctx)
	if span != nil {
		defer span.End()
	}

	log.Debug("Starting API version detection")

	// Record attempted versions as span attribute
	if span != nil {
		span.SetAttributes(
			attribute.StringSlice("netbackup.attempted_versions", models.SupportedAPIVersions),
		)
	}

	// Try each supported version in descending order
	for _, version := range models.SupportedAPIVersions {
		log.Debugf("Attempting API version %s", version)

		if d.tryVersion(ctx, version) {
			log.Infof("Successfully detected API version: %s", version)

			// Record detected version as span attribute
			if span != nil {
				span.SetAttributes(
					attribute.String("netbackup.detected_version", version),
				)
				span.SetStatus(codes.Ok, fmt.Sprintf("Detected version %s", version))
			}

			return version, nil
		}
	}

	// If we get here, no version worked
	err := fmt.Errorf(
		"failed to detect compatible NetBackup API version\n\n"+
			"Attempted versions: %v\n\n"+
			"Possible causes:\n"+
			"1. NetBackup server is running a version older than 10.0\n"+
			"2. Network connectivity issues\n"+
			"3. API endpoint is not accessible\n"+
			"4. Authentication credentials are invalid\n\n"+
			"Troubleshooting:\n"+
			"- Verify NetBackup server version: bpgetconfig -g | grep VERSION\n"+
			"- Check network connectivity to %s\n"+
			"- Verify API key is valid and not expired\n"+
			"- Try manually specifying apiVersion in config.yaml",
		models.SupportedAPIVersions,
		d.baseURL,
	)

	// Record error on span
	if span != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Version detection failed")
	}

	return "", err
}

// tryVersion tests a specific API version by making a lightweight API call.
// It uses the jobs endpoint with a limit of 1 to minimize server load.
//
// The method distinguishes between different error types:
// - HTTP 406: Version not supported, try next version
// - HTTP 401: Authentication error, fail immediately
// - Network errors: Retry with exponential backoff
// - Other errors: Log and try next version
//
// This method is immutable - it does not modify any configuration state.
//
// Parameters:
//   - ctx: Context for request cancellation and timeout
//   - version: API version to test (e.g., "13.0")
//
// Returns:
//   - true if the version is supported and working
//   - false if the version is not supported or encounters errors
func (d *APIVersionDetector) tryVersion(ctx context.Context, version string) bool {
	span := trace.SpanFromContext(ctx)

	testURL := d.buildTestURL(version)
	return d.retryVersionTest(ctx, version, testURL, span)
}

// buildTestURL builds the test URL for version detection with the specified API version.
// This method is immutable - it builds the URL without modifying any state.
//
// Parameters:
//   - version: API version to include in the URL path
//
// Returns the test URL for the jobs endpoint
func (d *APIVersionDetector) buildTestURL(version string) string {
	return fmt.Sprintf("%s/admin/jobs?page[limit]=1", d.baseURL)
}

// retryVersionTest implements retry logic with exponential backoff
func (d *APIVersionDetector) retryVersionTest(ctx context.Context, version, testURL string, span trace.Span) bool {
	delay := d.retryConfig.InitialDelay

	for attempt := 1; attempt <= d.retryConfig.MaxAttempts; attempt++ {
		resp, err := d.makeVersionTestRequest(ctx, testURL, version)

		if err != nil {
			if d.shouldRetry(attempt) {
				delay = d.retryWithBackoff(delay)
				continue
			}
			d.recordNetworkError(span, version, err)
			return false
		}

		result, shouldContinue := d.handleResponseStatus(ctx, version, resp, attempt, &delay, span)
		if !shouldContinue {
			return result
		}
	}

	return false
}

// makeVersionTestRequest makes the HTTP request to test the API version.
// It builds headers inline with the specified version instead of relying on client state.
// This ensures the request is immutable and does not depend on client configuration.
//
// Parameters:
//   - ctx: Context for request cancellation and timeout
//   - testURL: Full URL to test
//   - version: API version to test (used in Accept header)
//
// Returns the HTTP response and any error encountered
func (d *APIVersionDetector) makeVersionTestRequest(ctx context.Context, testURL, version string) (*resty.Response, error) {
	// Build headers inline with the test version
	headers := map[string]string{
		HeaderAccept:        fmt.Sprintf("application/vnd.netbackup+json;version=%s", version),
		HeaderAuthorization: d.apiKey,
	}

	return d.client.client.R().
		SetContext(ctx).
		SetHeaders(headers).
		Get(testURL)
}

// shouldRetry checks if we should retry the request
func (d *APIVersionDetector) shouldRetry(attempt int) bool {
	return attempt < d.retryConfig.MaxAttempts
}

// retryWithBackoff implements exponential backoff and returns the new delay
func (d *APIVersionDetector) retryWithBackoff(delay time.Duration) time.Duration {
	log.Debugf("Retrying in %v...", delay)
	time.Sleep(delay)
	newDelay := time.Duration(float64(delay) * d.retryConfig.BackoffFactor)
	if newDelay > d.retryConfig.MaxDelay {
		return d.retryConfig.MaxDelay
	}
	return newDelay
}

// recordNetworkError records a network error in the span
func (d *APIVersionDetector) recordNetworkError(span trace.Span, version string, err error) {
	log.Warnf("API version %s failed after %d attempts: %v", version, d.retryConfig.MaxAttempts, err)

	if span != nil {
		span.AddEvent("version_attempt_failed",
			trace.WithAttributes(
				attribute.String("version", version),
				attribute.Bool("success", false),
				attribute.String("failure_reason", "network_error"),
				attribute.String("error", err.Error()),
				attribute.Int("attempts", d.retryConfig.MaxAttempts),
			),
		)
	}
}

// handleResponseStatus handles different HTTP response status codes
// Returns (result, shouldContinue) where shouldContinue indicates if the retry loop should continue
func (d *APIVersionDetector) handleResponseStatus(ctx context.Context, version string, resp *resty.Response, attempt int, delay *time.Duration, span trace.Span) (bool, bool) {
	switch resp.StatusCode() {
	case 200:
		return d.handleSuccess(version, span), false
	case 401:
		return d.handleAuthError(version, span), false
	case 406:
		return d.handleVersionNotSupported(version, span), false
	case 500, 502, 503, 504:
		return d.handleTransientError(version, resp.StatusCode(), attempt, delay, span)
	default:
		return d.handleUnexpectedStatus(version, resp, span), false
	}
}

// handleSuccess handles a successful version test (HTTP 200)
func (d *APIVersionDetector) handleSuccess(version string, span trace.Span) bool {
	log.Debugf("API version %s is supported (HTTP 200)", version)

	if span != nil {
		span.AddEvent("version_attempt_success",
			trace.WithAttributes(
				attribute.String("version", version),
				attribute.Int("http_status", 200),
				attribute.Bool("success", true),
			),
		)
	}

	return true
}

// handleAuthError handles authentication errors (HTTP 401)
func (d *APIVersionDetector) handleAuthError(version string, span trace.Span) bool {
	log.Errorf("Authentication failed (HTTP 401). Please verify your API key is valid.")

	if span != nil {
		span.AddEvent("version_attempt_failed",
			trace.WithAttributes(
				attribute.String("version", version),
				attribute.Int("http_status", 401),
				attribute.Bool("success", false),
				attribute.String("failure_reason", "authentication_error"),
			),
		)
	}

	return false
}

// handleVersionNotSupported handles version not supported errors (HTTP 406)
func (d *APIVersionDetector) handleVersionNotSupported(version string, span trace.Span) bool {
	log.Debugf("API version %s not supported by server (HTTP 406)", version)

	if span != nil {
		span.AddEvent("version_attempt_failed",
			trace.WithAttributes(
				attribute.String("version", version),
				attribute.Int("http_status", 406),
				attribute.Bool("success", false),
				attribute.String("failure_reason", "version_not_supported"),
			),
		)
	}

	return false
}

// handleTransientError handles transient server errors (HTTP 500, 502, 503, 504)
// Returns (result, shouldContinue)
func (d *APIVersionDetector) handleTransientError(version string, statusCode, attempt int, delay *time.Duration, span trace.Span) (bool, bool) {
	log.Debugf("API version %s attempt %d/%d failed with transient error (HTTP %d)",
		version, attempt, d.retryConfig.MaxAttempts, statusCode)

	if d.shouldRetry(attempt) {
		*delay = d.retryWithBackoff(*delay)
		return false, true // Continue retrying
	}

	log.Warnf("API version %s failed after %d attempts with HTTP %d",
		version, d.retryConfig.MaxAttempts, statusCode)

	if span != nil {
		span.AddEvent("version_attempt_failed",
			trace.WithAttributes(
				attribute.String("version", version),
				attribute.Int("http_status", statusCode),
				attribute.Bool("success", false),
				attribute.String("failure_reason", "transient_server_error"),
				attribute.Int("attempts", d.retryConfig.MaxAttempts),
			),
		)
	}

	return false, false // Don't continue
}

// handleUnexpectedStatus handles unexpected HTTP status codes
func (d *APIVersionDetector) handleUnexpectedStatus(version string, resp *resty.Response, span trace.Span) bool {
	log.Warnf("API version %s returned unexpected status %d: %s",
		version, resp.StatusCode(), resp.Status())

	if span != nil {
		span.AddEvent("version_attempt_failed",
			trace.WithAttributes(
				attribute.String("version", version),
				attribute.Int("http_status", resp.StatusCode()),
				attribute.Bool("success", false),
				attribute.String("failure_reason", "unexpected_status"),
			),
		)
	}

	return false
}

// createVersionDetectionSpan creates a new span for version detection operations.
// This helper method provides nil-safe span creation and sets the span kind to client.
//
// Parameters:
//   - ctx: Parent context for the span
//
// Returns:
//   - Updated context with the span
//   - The created span (or nil if tracing is disabled)
func (d *APIVersionDetector) createVersionDetectionSpan(ctx context.Context) (context.Context, trace.Span) {
	// Nil-safe check: if tracer is not initialized, return original context and nil span
	if d.tracer == nil {
		return ctx, nil
	}

	// Start span with operation name and set span kind to client
	return d.tracer.Start(ctx, "netbackup.detect_version",
		trace.WithSpanKind(trace.SpanKindClient),
	)
}
