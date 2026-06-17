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
	// APIVersion100 represents the NetBackup 10.0-10.4 API version (media-type version=10.0).
	APIVersion100 = "10.0"
	// APIVersion120 represents NetBackup 10.5 API version
	APIVersion120 = "12.0"
	// APIVersion130 represents NetBackup 11.0 API version
	APIVersion130 = "13.0"
	// APIVersion140 represents NetBackup 11.2 API version
	APIVersion140 = "14.0"
)

// SupportedAPIVersions contains all supported NetBackup API versions in descending order.
// This list is used for version detection fallback (newest to oldest).
var SupportedAPIVersions = []string{APIVersion140, APIVersion130, APIVersion120, APIVersion100}

// NbuServerConfig holds connection settings for a single NetBackup master server.
// It mirrors the legacy inline NbuServer struct but adds a Site identifier used
// by the multi-site feature. The Site field defaults to the Host value when the
// legacy single nbuserver: block is auto-mapped.
type NbuServerConfig struct {
	Site               string `yaml:"site"`
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
}

// Config represents the complete application configuration for the NBU exporter.
// It includes settings for the server and the NBU server.
type Config struct {
	Server struct {
		Port               string `yaml:"port"`
		Host               string `yaml:"host"`
		URI                string `yaml:"uri"`
		ScrapingInterval   string `yaml:"scrapingInterval"`
		LogName            string `yaml:"logName"`
		CacheTTL           string `yaml:"cacheTTL"`           // TTL for storage metrics cache (e.g., "5m")
		CollectionInterval string `yaml:"collectionInterval"` // Background collection loop poll interval (e.g., "5m")
	} `yaml:"server"`

	// NbuServer is the legacy single-server configuration block. It is kept for
	// backward compatibility and is automatically promoted into NbuServers[0]
	// during Validate()/SetDefaults() if NbuServers is empty.
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

	// NbuServers is the multi-site server list. When empty and NbuServer.Host is
	// set, SetDefaults() auto-maps the legacy block into this slice.
	NbuServers []NbuServerConfig `yaml:"nbuservers"`

	OpenTelemetry struct {
		Enabled      bool    `yaml:"enabled"`
		Endpoint     string  `yaml:"endpoint"`
		Insecure     bool    `yaml:"insecure"`
		SamplingRate float64 `yaml:"samplingRate"`
	} `yaml:"opentelemetry"`

	Collectors struct {
		Alerts  CollectorToggle `yaml:"alerts"`
		Malware CollectorToggle `yaml:"malware"`
		Catalog CollectorToggle `yaml:"catalog"`
		SLO     CollectorToggle `yaml:"slo"`
	} `yaml:"collectors"`
}

// CollectorToggle enables an optional metric collector. New collectors default
// to disabled so existing deployments and older NetBackup versions are unaffected.
type CollectorToggle struct {
	Enabled bool `yaml:"enabled"`
}

// SetDefaults sets default values for optional configuration fields.
// Currently sets:
//   - Default NBU server URI to "/netbackup" if not specified
//   - Default storage cache TTL to "5m" if not specified
//
// APIVersion is intentionally NOT defaulted here. An omitted apiVersion is left
// empty so the client performs automatic version detection
// (14.0 -> 13.0 -> 12.0 -> 10.0); forcing a default would silently disable
// auto-detect and hard-fail the exporter against NetBackup < 11.2.
//
// This method is called automatically by Validate() before validation checks.
func (c *Config) SetDefaults() {
	// Set default NBU server URI
	if c.NbuServer.URI == "" {
		c.NbuServer.URI = "/netbackup"
	}

	// Set default storage cache TTL (5 minutes)
	if c.Server.CacheTTL == "" {
		c.Server.CacheTTL = "5m"
	}

	// Set default collection interval (5 minutes)
	if c.Server.CollectionInterval == "" {
		c.Server.CollectionInterval = "5m"
	}

	// Auto-map legacy single nbuserver: block into NbuServers when the list is empty.
	// deprecated single nbuserver auto-mapped
	if len(c.NbuServers) == 0 && c.NbuServer.Host != "" {
		c.NbuServers = []NbuServerConfig{
			{
				Site:               c.NbuServer.Host,
				Port:               c.NbuServer.Port,
				Scheme:             c.NbuServer.Scheme,
				URI:                c.NbuServer.URI,
				Domain:             c.NbuServer.Domain,
				DomainType:         c.NbuServer.DomainType,
				Host:               c.NbuServer.Host,
				APIKey:             c.NbuServer.APIKey,
				APIVersion:         c.NbuServer.APIVersion,
				ContentType:        c.NbuServer.ContentType,
				InsecureSkipVerify: c.NbuServer.InsecureSkipVerify,
			},
		}
	}

	// Reverse-map: when only nbuservers[] is provided (no legacy nbuserver: block),
	// mirror the primary entry into the legacy NbuServer fields so legacy
	// single-server code paths (validation, GetNBUBaseURL, telemetry, MaskAPIKey)
	// operate on the primary site.
	if c.NbuServer.Host == "" && len(c.NbuServers) > 0 {
		first := c.NbuServers[0]
		c.NbuServer.Port = first.Port
		c.NbuServer.Scheme = first.Scheme
		c.NbuServer.URI = first.URI
		c.NbuServer.Domain = first.Domain
		c.NbuServer.DomainType = first.DomainType
		c.NbuServer.Host = first.Host
		c.NbuServer.APIKey = first.APIKey
		c.NbuServer.APIVersion = first.APIVersion
		c.NbuServer.ContentType = first.ContentType
		c.NbuServer.InsecureSkipVerify = first.InsecureSkipVerify
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
//   - NBU base URL format (validates URL can be parsed)
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

	if err := c.validateNbuServers(); err != nil {
		return err
	}

	// Validate the composed URL is valid
	if err := c.validateNBUBaseURL(); err != nil {
		return err
	}

	if err := c.validateTLSConfig(); err != nil {
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
	if c.Server.ScrapingInterval == "" {
		return errors.New("server scraping interval is required")
	}
	if _, err := time.ParseDuration(c.Server.ScrapingInterval); err != nil {
		return fmt.Errorf("invalid scraping interval format '%s': %w (expected format: 5m, 1h, 30s)", c.Server.ScrapingInterval, err)
	}
	// Validate cache TTL if provided
	if c.Server.CacheTTL != "" {
		if _, err := time.ParseDuration(c.Server.CacheTTL); err != nil {
			return fmt.Errorf("invalid cache TTL format '%s': %w (expected format: 5m, 1h, 30s)", c.Server.CacheTTL, err)
		}
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

// validateNbuServers validates the NbuServers slice: requires at least one entry,
// each entry must have a non-empty Site and unique Site across the slice, and each
// entry's connection fields (host/port/scheme/apiKey) must be present and valid.
func (c *Config) validateNbuServers() error {
	if len(c.NbuServers) == 0 {
		return errors.New("at least one entry in nbuservers is required")
	}

	seen := make(map[string]bool, len(c.NbuServers))
	for i, s := range c.NbuServers {
		if s.Site == "" {
			return fmt.Errorf("nbuservers[%d]: site is required", i)
		}
		if seen[s.Site] {
			return fmt.Errorf("nbuservers: duplicate site name %q", s.Site)
		}
		seen[s.Site] = true

		if s.Host == "" {
			return fmt.Errorf("nbuservers[%d] (%s): host is required", i, s.Site)
		}
		if s.Port == "" {
			return fmt.Errorf("nbuservers[%d] (%s): port is required", i, s.Site)
		}
		if err := validatePort(s.Port); err != nil {
			return fmt.Errorf("nbuservers[%d] (%s): invalid port %s", i, s.Site, s.Port)
		}
		if s.Scheme != "http" && s.Scheme != "https" {
			return fmt.Errorf("nbuservers[%d] (%s): invalid scheme %q (must be http or https)", i, s.Site, s.Scheme)
		}
		if s.APIKey == "" {
			return fmt.Errorf("nbuservers[%d] (%s): apiKey is required", i, s.Site)
		}
	}
	return nil
}

// validateNBUBaseURL validates that the NBU server configuration produces a valid URL.
// This catches malformed host, scheme, or port values early during startup.
func (c *Config) validateNBUBaseURL() error {
	baseURL := c.GetNBUBaseURL()
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return fmt.Errorf("invalid NBU server URL '%s': %w", baseURL, err)
	}

	// Verify the URL has required components
	if parsedURL.Scheme == "" {
		return fmt.Errorf("NBU server URL missing scheme: %s", baseURL)
	}
	if parsedURL.Host == "" {
		return fmt.Errorf("NBU server URL missing host: %s", baseURL)
	}

	return nil
}

// validateTLSConfig validates TLS security settings.
// InsecureSkipVerify is controlled via config file - no additional validation needed.
// The client.go logs an Error-level warning when insecure mode is enabled.
func (c *Config) validateTLSConfig() error {
	// TLS configuration is validated by the config file setting.
	// Warning is logged at client initialization time.
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
// Example: "0.0.0.0:9440"
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

// GetCacheTTL parses and returns the cache TTL as time.Duration.
// Returns 5 minutes default if parsing fails or not configured.
//
// The cache TTL determines how long storage metrics are cached before
// requiring a fresh API call to NetBackup.
//
// Example: "5m" -> 5 * time.Minute
func (c *Config) GetCacheTTL() time.Duration {
	if c.Server.CacheTTL == "" {
		return 5 * time.Minute
	}
	duration, err := time.ParseDuration(c.Server.CacheTTL)
	if err != nil {
		return 5 * time.Minute
	}
	return duration
}

// GetCollectionInterval parses and returns the background collection loop poll
// interval as a time.Duration. Returns 5 minutes if unset or unparseable.
//
// This is how often the snapshot collection loop polls every configured site;
// it decouples backend API load from Prometheus scrape frequency.
//
// Example: "5m" -> 5 * time.Minute
func (c *Config) GetCollectionInterval() time.Duration {
	if c.Server.CollectionInterval == "" {
		return 5 * time.Minute
	}
	duration, err := time.ParseDuration(c.Server.CollectionInterval)
	if err != nil {
		return 5 * time.Minute
	}
	return duration
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
// IMPORTANT: This method assumes the configuration has been validated via Config.Validate()
// prior to use. Validation ensures the base URL is well-formed and parseable. If called
// on an unvalidated config, URL parsing errors will be silently ignored.
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
