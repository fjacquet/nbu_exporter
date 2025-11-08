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
	defaultTimeout = 1 * time.Minute
	contentType    = "application/json"
)

// HTTPHeaders contains common HTTP header names.
const (
	HeaderAccept        = "Accept"
	HeaderAuthorization = "Authorization"
)

// QueryParams contains common query parameter names.
const (
	QueryParamLimit  = "page[limit]"
	QueryParamOffset = "page[offset]"
	QueryParamSort   = "sort"
	QueryParamFilter = "filter"
)

// NbuClient handles HTTP communication with the NetBackup API.
type NbuClient struct {
	client *resty.Client
	cfg    models.Config
}

// NewNbuClient creates a new NetBackup API client with the provided configuration.
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

// FetchData sends an HTTP GET request and unmarshals the response into the target.
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

// DetectAPIVersion attempts to detect the NetBackup API version by making a lightweight API call.
// It returns the configured API version and logs the detection result.
// If detection fails, it returns an error but does not prevent the client from functioning.
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

// Close closes the underlying HTTP client connections.
func (c *NbuClient) Close() {
	// Resty doesn't have an explicit close, but we can clear the client
	c.client = nil
}
