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
}

// SetDefaults sets default values for optional configuration fields.
// Currently sets the default API version to "12.0" (NetBackup 10.5+) if not specified.
// This method is called automatically by Validate() before validation checks.
func (c *Config) SetDefaults() {
	// Set default API version for NetBackup 10.5
	if c.NbuServer.APIVersion == "" {
		c.NbuServer.APIVersion = "12.0"
	}
}

// Validate checks if the configuration is valid and returns an error if not.
// It performs comprehensive validation of all configuration fields including:
//   - Server settings (host, port, URI, scraping interval)
//   - NetBackup server settings (host, port, scheme, API key, API version)
//   - Port ranges (1-65535)
//   - URL schemes (http/https only)
//   - API version format (X.Y pattern)
//
// This method calls SetDefaults() before validation to ensure optional fields
// have appropriate default values.
//
// Returns an error describing the first validation failure encountered.
func (c *Config) Validate() error {
	// Set defaults before validation
	c.SetDefaults()

	// Validate server configuration
	if c.Server.Port == "" {
		return errors.New("server port is required")
	}
	if port, err := strconv.Atoi(c.Server.Port); err != nil || port < 1 || port > 65535 {
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

	// Validate NBU server configuration
	if c.NbuServer.Host == "" {
		return errors.New("NBU server host is required")
	}
	if c.NbuServer.Port == "" {
		return errors.New("NBU server port is required")
	}
	if port, err := strconv.Atoi(c.NbuServer.Port); err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("invalid NBU server port: %s", c.NbuServer.Port)
	}
	if c.NbuServer.Scheme != "http" && c.NbuServer.Scheme != "https" {
		return fmt.Errorf("invalid NBU server scheme: %s (must be http or https)", c.NbuServer.Scheme)
	}
	if c.NbuServer.APIKey == "" {
		return errors.New("NBU server API key is required")
	}

	// Validate API version format (e.g., "12.0", "11.1", "10.5")
	if c.NbuServer.APIVersion != "" {
		apiVersionPattern := regexp.MustCompile(`^\d+\.\d+$`)
		if !apiVersionPattern.MatchString(c.NbuServer.APIVersion) {
			return fmt.Errorf("invalid API version format: %s (must be in format X.Y, e.g., 12.0)", c.NbuServer.APIVersion)
		}
	}

	return nil
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
	u.Path = path
	q := u.Query()
	for key, value := range queryParams {
		q.Set(key, value)
	}
	u.RawQuery = q.Encode()
	return u.String()
}
