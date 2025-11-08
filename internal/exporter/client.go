// Package exporter provides HTTP client functionality and data fetching logic
// for the NetBackup REST API. It handles API communication, pagination,
// and metric collection for Prometheus exposition.
package exporter

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
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
				"API version %s is not supported by the NetBackup server (HTTP 406 Not Acceptable). "+
					"The server may be running an older version of NetBackup that does not support API version %s. "+
					"Please verify your NetBackup server version and update the 'apiVersion' field in your configuration "+
					"to match a supported version. Common versions: '10.0' (NetBackup 10.0-10.4), '12.0' (NetBackup 10.5+). "+
					"URL: %s",
				c.cfg.NbuServer.APIVersion,
				c.cfg.NbuServer.APIVersion,
				url,
			)
			logging.LogError(errMsg)
			return fmt.Errorf("%s", errMsg)
		}

		return fmt.Errorf("HTTP request to %s returned status %d: %s", url, resp.StatusCode(), resp.Status())
	}

	if err := json.Unmarshal(resp.Body(), target); err != nil {
		return fmt.Errorf("failed to unmarshal response from %s: %w", url, err)
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
