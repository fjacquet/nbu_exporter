// Package models defines the core data structures for the NBU exporter application.
// It includes configuration models and API response structures that match the
// NetBackup REST API JSON:API format.
package models

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"time"
)

// Supported NetBackup API versions
const (
	// APIVersion30 represents NetBackup 10.0-10.4 API version
	APIVersion30 = "3.0"
	// APIVersion120 represents NetBackup 10.5 API version
	APIVersion120 = "12.0"
	// APIVersion130 represents NetBackup 11.0 API version
	APIVersion130 = "13.0"
)

// SupportedAPIVersions contains all supported NetBackup API versions in descending order.
// This list is used for version detection fallback (newest to oldest).
var SupportedAPIVersions = []string{APIVersion130, APIVersion120, APIVersion30}

// Config represents the complete application configuration for the NBU exporter.
// It includes settings for the server and the NBU server.
type Config struct {
	Server struct {
		Port             string `yaml:"port"`
		Host             string `yaml:"host"`
		URI              string `yaml:"uri"`
		ScrapingInterval string `yaml:"scrapingInterval"`
		LogName          string `yaml:"logName"`
	} `yaml:"server"`

	NbuServer struct {
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
	} `yaml:"nbuserver"`

	OpenTelemetry struct {
		Enabled      bool    `yaml:"enabled"`
		Endpoint     string  `yaml:"endpoint"`
		Insecure     bool    `yaml:"insecure"`
		SamplingRate float64 `yaml:"samplingRate"`
	} `yaml:"opentelemetry"`
}

// SetDefaults sets default values for optional configuration fields.
// Currently sets:
//   - Default API version to "13.0" (NetBackup 11.0) if not specified
//   - Default NBU server URI to "/netbackup" if not specified
//
// This method is called automatically by Validate() before validation checks.
func (c *Config) SetDefaults() {
	// Set default API version for NetBackup 11.0
	if c.NbuServer.APIVersion == "" {
		c.NbuServer.APIVersion = APIVersion130
	}

	// Set default NBU server URI
	if c.NbuServer.URI == "" {
		c.NbuServer.URI = "/netbackup"
	}
}

// Validate checks if the configuration is valid and returns an error if not.
// It performs comprehensive validation of all configuration fields including:
//   - Server settings (host, port, URI, scraping interval)
//   - NetBackup server settings (host, port, scheme, API key, API version)
//   - Port ranges (1-65535)
//   - URL schemes (http/https only)
//   - API version format (X.Y pattern)
//   - API version is in the supported versions list
//
// This method calls SetDefaults() before validation to ensure optional fields
// have appropriate default values.
//
// Returns an error describing the first validation failure encountered.
func (c *Config) Validate() error {
	// Set defaults before validation
	c.SetDefaults()

	if err := c.validateServerConfig(); err != nil {
		return err
	}

	if err := c.validateNBUServerConfig(); err != nil {
		return err
	}

	if err := c.validateAPIVersion(); err != nil {
		return err
	}

	if err := c.validateOpenTelemetryConfig(); err != nil {
		return err
	}

	return nil
}

// validateServerConfig validates the server configuration settings
func (c *Config) validateServerConfig() error {
	if c.Server.Port == "" {
		return errors.New("server port is required")
	}
	if err := validatePort(c.Server.Port); err != nil {
		return fmt.Errorf("invalid server port: %s", c.Server.Port)
	}
	if c.Server.Host == "" {
		return errors.New("server host is required")
	}
	if c.Server.URI == "" {
		return errors.New("server URI is required")
	}
	if _, err := time.ParseDuration(c.Server.ScrapingInterval); err != nil {
		return fmt.Errorf("invalid scraping interval: %w", err)
	}
	return nil
}

// validateNBUServerConfig validates the NetBackup server configuration settings
func (c *Config) validateNBUServerConfig() error {
	if c.NbuServer.Host == "" {
		return errors.New("NBU server host is required")
	}
	if c.NbuServer.Port == "" {
		return errors.New("NBU server port is required")
	}
	if err := validatePort(c.NbuServer.Port); err != nil {
		return fmt.Errorf("invalid NBU server port: %s", c.NbuServer.Port)
	}
	if c.NbuServer.Scheme != "http" && c.NbuServer.Scheme != "https" {
		return fmt.Errorf("invalid NBU server scheme: %s (must be http or https)", c.NbuServer.Scheme)
	}
	if c.NbuServer.APIKey == "" {
		return errors.New("NBU server API key is required")
	}
	return nil
}

// validateAPIVersion validates the API version format and checks if it's supported
func (c *Config) validateAPIVersion() error {
	if c.NbuServer.APIVersion == "" {
		return nil
	}

	apiVersionPattern := regexp.MustCompile(`^\d+\.\d+$`)
	if !apiVersionPattern.MatchString(c.NbuServer.APIVersion) {
		return fmt.Errorf("invalid API version format: %s (must be in format X.Y, e.g., 13.0)", c.NbuServer.APIVersion)
	}

	if !isVersionSupported(c.NbuServer.APIVersion) {
		return fmt.Errorf("unsupported API version: %s (supported versions: %v)", c.NbuServer.APIVersion, SupportedAPIVersions)
	}

	return nil
}

// validateOpenTelemetryConfig validates the OpenTelemetry configuration if enabled
func (c *Config) validateOpenTelemetryConfig() error {
	if !c.OpenTelemetry.Enabled {
		return nil
	}

	if err := c.validateOTelEndpoint(); err != nil {
		return err
	}

	if c.OpenTelemetry.SamplingRate < 0.0 || c.OpenTelemetry.SamplingRate > 1.0 {
		return fmt.Errorf("OpenTelemetry sampling rate must be between 0.0 and 1.0, got: %f", c.OpenTelemetry.SamplingRate)
	}

	return nil
}

// validatePort validates that a port string is a valid integer in the range 1-65535
func validatePort(portStr string) error {
	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		return errors.New("port must be between 1 and 65535")
	}
	return nil
}

// isVersionSupported checks if the given API version is in the supported versions list
func isVersionSupported(version string) bool {
	for _, v := range SupportedAPIVersions {
		if version == v {
			return true
		}
	}
	return false
}

// validateOTelEndpoint validates the OpenTelemetry endpoint format and port range.
// It checks that the endpoint follows the "host:port" pattern and that the port
// is within the valid range (1-65535).
//
// Returns an error if:
//   - The endpoint is empty
//   - The endpoint format is not "host:port"
//   - The port is not a valid integer
//   - The port is outside the range 1-65535
func (c *Config) validateOTelEndpoint() error {
	if c.OpenTelemetry.Endpoint == "" {
		return errors.New("OpenTelemetry endpoint is required when enabled")
	}

	// Split endpoint into host and port
	host, port, err := splitHostPort(c.OpenTelemetry.Endpoint)
	if err != nil {
		return fmt.Errorf("invalid OpenTelemetry endpoint format: %s (expected host:port)", c.OpenTelemetry.Endpoint)
	}

	// Validate host is not empty
	if host == "" {
		return fmt.Errorf("invalid OpenTelemetry endpoint: host cannot be empty in %s", c.OpenTelemetry.Endpoint)
	}

	// Validate port is a valid integer in range 1-65535
	portNum, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("invalid OpenTelemetry endpoint port: %s (must be a valid integer)", port)
	}
	if portNum < 1 || portNum > 65535 {
		return fmt.Errorf("invalid OpenTelemetry endpoint port: %d (must be between 1 and 65535)", portNum)
	}

	return nil
}

// splitHostPort splits a "host:port" string into host and port components.
// This is a helper function for endpoint validation that handles the parsing
// of the endpoint string.
//
// Returns the host and port as separate strings, or an error if the format is invalid.
func splitHostPort(endpoint string) (host, port string, err error) {
	// Find the last colon to handle IPv6 addresses like [::1]:4317
	lastColon := -1
	for i := len(endpoint) - 1; i >= 0; i-- {
		if endpoint[i] == ':' {
			lastColon = i
			break
		}
	}

	if lastColon == -1 {
		return "", "", fmt.Errorf("missing port in endpoint")
	}

	host = endpoint[:lastColon]
	port = endpoint[lastColon+1:]

	if host == "" || port == "" {
		return "", "", fmt.Errorf("invalid host:port format")
	}

	return host, port, nil
}

// GetNBUBaseURL returns the complete base URL for the NetBackup server.
// Format: scheme://host:port/uri
//
// Example: "https://nbu-master.example.com:1556/netbackup"
func (c *Config) GetNBUBaseURL() string {
	return fmt.Sprintf("%s://%s:%s%s", c.NbuServer.Scheme, c.NbuServer.Host, c.NbuServer.Port, c.NbuServer.URI)
}

// GetServerAddress returns the complete server address for HTTP server binding.
// Format: host:port
//
// Example: "0.0.0.0:2112"
func (c *Config) GetServerAddress() string {
	return fmt.Sprintf("%s:%s", c.Server.Host, c.Server.Port)
}

// GetScrapingDuration parses and returns the scraping interval as a time.Duration.
// The scraping interval defines the time window for job data collection.
//
// Example: "5m" -> 5 * time.Minute
//
// Returns an error if the interval string cannot be parsed.
func (c *Config) GetScrapingDuration() (time.Duration, error) {
	return time.ParseDuration(c.Server.ScrapingInterval)
}

// MaskAPIKey returns a masked version of the API key for safe logging.
// Shows the first 4 and last 4 characters with asterisks in between.
//
// Example: "abcd1234efgh5678" -> "abcd****5678"
//
// For keys shorter than 8 characters, returns "****".
func (c *Config) MaskAPIKey() string {
	if len(c.NbuServer.APIKey) <= 8 {
		return "****"
	}
	return c.NbuServer.APIKey[:4] + "****" + c.NbuServer.APIKey[len(c.NbuServer.APIKey)-4:]
}

// BuildURL constructs a complete URL from the base URL, path, and query parameters.
// It properly encodes query parameters and handles URL construction.
//
// Parameters:
//   - path: API endpoint path (e.g., "/admin/jobs")
//   - queryParams: Map of query parameter names to values
//
// Example:
//
//	url := cfg.BuildURL("/admin/jobs", map[string]string{
//	    "page[limit]": "100",
//	    "page[offset]": "0",
//	})
//	// Returns: "https://nbu:1556/netbackup/admin/jobs?page[limit]=100&page[offset]=0"
func (c *Config) BuildURL(path string, queryParams map[string]string) string {
	u, _ := url.Parse(c.GetNBUBaseURL())
	// Append the path to the existing base URL path (e.g., /netbackup + /admin/jobs)
	u.Path = u.Path + path
	q := u.Query()
	for key, value := range queryParams {
		q.Set(key, value)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// IsOTelEnabled returns whether OpenTelemetry tracing is enabled in the configuration.
// This is a convenience method to check the OpenTelemetry.Enabled field.
//
// Returns true if OpenTelemetry is enabled, false otherwise.
func (c *Config) IsOTelEnabled() bool {
	return c.OpenTelemetry.Enabled
}

// OTelConfig represents OpenTelemetry configuration settings.
type OTelConfig struct {
	Enabled      bool
	Endpoint     string
	Insecure     bool
	SamplingRate float64
}

// GetOTelConfig returns a copy of the OpenTelemetry configuration settings.
// This method extracts the OpenTelemetry configuration for use by the telemetry manager.
//
// Returns an OTelConfig struct containing:
//   - Enabled: Whether OpenTelemetry is enabled
//   - Endpoint: OTLP gRPC endpoint address
//   - Insecure: Whether to use insecure connection (no TLS)
//   - SamplingRate: Trace sampling rate (0.0 to 1.0)
func (c *Config) GetOTelConfig() OTelConfig {
	return OTelConfig{
		Enabled:      c.OpenTelemetry.Enabled,
		Endpoint:     c.OpenTelemetry.Endpoint,
		Insecure:     c.OpenTelemetry.Insecure,
		SamplingRate: c.OpenTelemetry.SamplingRate,
	}
}
