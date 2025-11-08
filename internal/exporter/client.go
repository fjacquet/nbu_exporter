// Package exporter provides HTTP client functionality and data fetching logic
// for the NetBackup REST API. It handles API communication, pagination,
// and metric collection for Prometheus exposition.
package exporter

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/fjacquet/nbu_exporter/internal/logging"
	"github.com/fjacquet/nbu_exporter/internal/models"
	"github.com/go-resty/resty/v2"
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
}

// NewNbuClient creates a new NetBackup API client with the provided configuration.
// It initializes the HTTP client with appropriate TLS settings and timeout values.
//
// The client is configured with:
//   - TLS verification based on cfg.NbuServer.InsecureSkipVerify
//   - Default timeout of 1 minute for all requests
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

	return &NbuClient{
		client: client,
		cfg:    cfg,
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
	if cfg.NbuServer.APIVersion == "" || cfg.NbuServer.APIVersion == models.APIVersion130 {
		// Only perform detection if version is empty or set to default
		// This allows users to explicitly configure a version to bypass detection
		shouldDetect := cfg.NbuServer.APIVersion == ""

		if shouldDetect {
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
		} else {
			logging.LogInfo(fmt.Sprintf("Using configured API version: %s", cfg.NbuServer.APIVersion))
		}
	} else {
		logging.LogInfo(fmt.Sprintf("Using explicitly configured API version: %s (bypassing detection)", cfg.NbuServer.APIVersion))
	}

	return client, nil
}

// getHeaders returns the standard headers for NBU API requests.
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
// Parameters:
//   - ctx: Context for request cancellation and timeout
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
	resp, err := c.client.R().
		SetContext(ctx).
		SetHeaders(c.getHeaders()).
		Get(url)

	if err != nil {
		return fmt.Errorf("HTTP request to %s failed: %w", url, err)
	}

	if resp.IsError() {
		// Handle 406 Not Acceptable - API version not supported
		if resp.StatusCode() == 406 {
			errMsg := fmt.Sprintf(
				"API version %s is not supported by the NetBackup server (HTTP 406 Not Acceptable).\n\n"+
					"The server may be running a version of NetBackup that does not support API version %s.\n\n"+
					"Supported API versions:\n"+
					"  - 3.0  (NetBackup 10.0-10.4)\n"+
					"  - 12.0 (NetBackup 10.5)\n"+
					"  - 13.0 (NetBackup 11.0)\n\n"+
					"Troubleshooting steps:\n"+
					"1. Verify your NetBackup server version: bpgetconfig -g | grep VERSION\n"+
					"2. Update the 'apiVersion' field in config.yaml to match your server version\n"+
					"3. Or remove the 'apiVersion' field to enable automatic version detection\n\n"+
					"Example configuration:\n"+
					"  nbuserver:\n"+
					"    apiVersion: \"12.0\"  # For NetBackup 10.5\n"+
					"    # Or omit apiVersion for automatic detection\n\n"+
					"Request URL: %s",
				c.cfg.NbuServer.APIVersion,
				c.cfg.NbuServer.APIVersion,
				url,
			)
			logging.LogError(errMsg)
			return fmt.Errorf("%s", errMsg)
		}

		return fmt.Errorf("HTTP request to %s returned status %d: %s", url, resp.StatusCode(), resp.Status())
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
			"NetBackup server returned non-JSON response (Content-Type: %s).\n\n"+
				"This usually indicates:\n"+
				"1. Wrong API endpoint URL (check 'uri' in config.yaml)\n"+
				"2. Authentication failure (verify API key is valid)\n"+
				"3. Server configuration issue (check NetBackup REST API is enabled)\n\n"+
				"Request URL: %s\n"+
				"Response preview: %s",
			contentType,
			url,
			bodyPreview,
		)
		logging.LogError(errMsg)
		return fmt.Errorf("server returned %s instead of JSON: %s", contentType, bodyPreview)
	}

	if err := json.Unmarshal(resp.Body(), target); err != nil {
		// Provide more context for JSON unmarshaling errors
		bodyPreview := string(resp.Body())
		if len(bodyPreview) > 200 {
			bodyPreview = bodyPreview[:200] + "..."
		}
		return fmt.Errorf("failed to unmarshal JSON response from %s: %w\nResponse preview: %s", url, err, bodyPreview)
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

// Close releases resources associated with the HTTP client.
// Note: Resty doesn't provide an explicit close method, so this clears the client reference
// to allow garbage collection. Connection pooling is managed by the underlying http.Client.
func (c *NbuClient) Close() {
	// Resty doesn't have an explicit close, but we can clear the client
	c.client = nil
}
