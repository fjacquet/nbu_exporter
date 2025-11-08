// Package exporter provides API version detection functionality for NetBackup.
// It implements automatic version detection with fallback logic to support
// multiple NetBackup versions (10.0, 10.5, and 11.0).
package exporter

import (
	"context"
	"fmt"
	"time"

	"github.com/fjacquet/nbu_exporter/internal/models"
	log "github.com/sirupsen/logrus"
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
type APIVersionDetector struct {
	client      *NbuClient     // HTTP client for making test requests
	cfg         *models.Config // Application configuration
	retryConfig RetryConfig    // Retry configuration for transient failures
}

// NewAPIVersionDetector creates a new version detector with the provided client and configuration.
// It uses the default retry configuration for handling transient failures.
//
// Parameters:
//   - client: Configured NetBackup API client
//   - cfg: Application configuration
//
// Returns a new APIVersionDetector instance.
func NewAPIVersionDetector(client *NbuClient, cfg *models.Config) *APIVersionDetector {
	return &APIVersionDetector{
		client:      client,
		cfg:         cfg,
		retryConfig: DefaultRetryConfig,
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
	log.Debug("Starting API version detection")

	// Try each supported version in descending order
	for _, version := range models.SupportedAPIVersions {
		log.Debugf("Attempting API version %s", version)

		if d.tryVersion(ctx, version) {
			log.Infof("Successfully detected API version: %s", version)
			return version, nil
		}
	}

	// If we get here, no version worked
	return "", fmt.Errorf(
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
		d.cfg.GetNBUBaseURL(),
	)
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
// Parameters:
//   - ctx: Context for request cancellation and timeout
//   - version: API version to test (e.g., "13.0")
//
// Returns:
//   - true if the version is supported and working
//   - false if the version is not supported or encounters errors
func (d *APIVersionDetector) tryVersion(ctx context.Context, version string) bool {
	// Temporarily set the version in both the detector's config and the client's config
	originalVersion := d.cfg.NbuServer.APIVersion
	d.cfg.NbuServer.APIVersion = version
	d.client.cfg.NbuServer.APIVersion = version
	defer func() {
		d.cfg.NbuServer.APIVersion = originalVersion
		d.client.cfg.NbuServer.APIVersion = originalVersion
	}()

	// Use a lightweight endpoint to test API connectivity
	baseURL := d.cfg.GetNBUBaseURL()
	testURL := fmt.Sprintf("%s/admin/jobs?page[limit]=1", baseURL)

	// Implement retry logic with exponential backoff
	delay := d.retryConfig.InitialDelay
	for attempt := 1; attempt <= d.retryConfig.MaxAttempts; attempt++ {
		resp, err := d.client.client.R().
			SetContext(ctx).
			SetHeaders(d.client.getHeaders()).
			Get(testURL)

		if err != nil {
			// Network error - retry with backoff
			log.Debugf("API version %s attempt %d/%d failed with network error: %v",
				version, attempt, d.retryConfig.MaxAttempts, err)

			if attempt < d.retryConfig.MaxAttempts {
				log.Debugf("Retrying in %v...", delay)
				time.Sleep(delay)
				delay = time.Duration(float64(delay) * d.retryConfig.BackoffFactor)
				if delay > d.retryConfig.MaxDelay {
					delay = d.retryConfig.MaxDelay
				}
				continue
			}

			log.Warnf("API version %s failed after %d attempts: %v", version, d.retryConfig.MaxAttempts, err)
			return false
		}

		// Check response status
		switch resp.StatusCode() {
		case 200:
			// Success - this version works
			log.Debugf("API version %s is supported (HTTP 200)", version)
			return true

		case 401:
			// Authentication error - fail immediately, don't try other versions
			log.Errorf("Authentication failed (HTTP 401). Please verify your API key is valid.")
			return false

		case 406:
			// Version not supported - try next version
			log.Debugf("API version %s not supported by server (HTTP 406)", version)
			return false

		case 500, 502, 503, 504:
			// Transient server errors - retry with backoff
			log.Debugf("API version %s attempt %d/%d failed with transient error (HTTP %d)",
				version, attempt, d.retryConfig.MaxAttempts, resp.StatusCode())

			if attempt < d.retryConfig.MaxAttempts {
				log.Debugf("Retrying in %v...", delay)
				time.Sleep(delay)
				delay = time.Duration(float64(delay) * d.retryConfig.BackoffFactor)
				if delay > d.retryConfig.MaxDelay {
					delay = d.retryConfig.MaxDelay
				}
				continue
			}

			log.Warnf("API version %s failed after %d attempts with HTTP %d",
				version, d.retryConfig.MaxAttempts, resp.StatusCode())
			return false

		default:
			// Other error - log and try next version
			log.Warnf("API version %s returned unexpected status %d: %s",
				version, resp.StatusCode(), resp.Status())
			return false
		}
	}

	return false
}
