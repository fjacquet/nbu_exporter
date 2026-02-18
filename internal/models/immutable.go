// Package models defines the core data structures for the NBU exporter application.
package models

import (
	"time"
)

// ImmutableConfig Usage Pattern
//
// ImmutableConfig is designed for incremental adoption. The recommended pattern is:
//
//   1. Parse and validate Config from YAML (existing code)
//   2. Run version detection if needed (existing code)
//   3. Create ImmutableConfig: immCfg, err := NewImmutableConfig(cfg)
//   4. Pass ImmutableConfig to new components
//
// For existing components (NbuClient, NbuCollector), migration can be done incrementally
// in future phases. The ImmutableConfig type is available now for:
//   - New code paths that need thread-safe config access
//   - Gradual migration of existing components
//   - Clear separation between setup-time and runtime configuration
//
// Full migration of NbuClient and NbuCollector to use only ImmutableConfig internally
// is planned for a future phase. This provides the type and pattern; incremental
// adoption can follow without breaking changes.

// ImmutableConfig holds configuration values that are fixed after initialization.
// This type is created after validation and version detection complete, ensuring
// all values are finalized and cannot be modified during execution.
//
// Design rationale:
// - Separates mutable YAML parsing (Config) from immutable runtime use (ImmutableConfig)
// - Guarantees thread-safety: no synchronization needed for reads
// - Makes dependencies explicit: components declare they need finalized config
// - Prevents accidental modification bugs
type ImmutableConfig struct {
	// NBU server connection settings
	baseURL            string
	apiKey             string
	apiVersion         string
	insecureSkipVerify bool

	// Server settings
	serverAddress    string
	metricsURI       string
	scrapingInterval time.Duration
	logName          string

	// OpenTelemetry settings
	otelEnabled      bool
	otelEndpoint     string
	otelInsecure     bool
	otelSamplingRate float64
}

// NewImmutableConfig creates an ImmutableConfig from a validated Config.
// This should be called AFTER:
//  1. Config.Validate() has passed
//  2. Version detection has completed (if needed)
//  3. All config mutations are complete
//
// Returns an error if scraping interval cannot be parsed.
func NewImmutableConfig(cfg *Config) (ImmutableConfig, error) {
	scrapingDuration, err := cfg.GetScrapingDuration()
	if err != nil {
		return ImmutableConfig{}, err
	}

	return ImmutableConfig{
		// NBU server
		baseURL:            cfg.GetNBUBaseURL(),
		apiKey:             cfg.NbuServer.APIKey,
		apiVersion:         cfg.NbuServer.APIVersion,
		insecureSkipVerify: cfg.NbuServer.InsecureSkipVerify,

		// Server
		serverAddress:    cfg.GetServerAddress(),
		metricsURI:       cfg.Server.URI,
		scrapingInterval: scrapingDuration,
		logName:          cfg.Server.LogName,

		// OpenTelemetry
		otelEnabled:      cfg.OpenTelemetry.Enabled,
		otelEndpoint:     cfg.OpenTelemetry.Endpoint,
		otelInsecure:     cfg.OpenTelemetry.Insecure,
		otelSamplingRate: cfg.OpenTelemetry.SamplingRate,
	}, nil
}

// Accessor methods - all return copies/values, not references

// BaseURL returns the complete NBU server base URL.
func (c ImmutableConfig) BaseURL() string {
	return c.baseURL
}

// APIKey returns the NBU API key for authentication.
// SECURITY: Handle with care - do not log this value.
func (c ImmutableConfig) APIKey() string {
	return c.apiKey
}

// APIVersion returns the NBU API version string.
func (c ImmutableConfig) APIVersion() string {
	return c.apiVersion
}

// InsecureSkipVerify returns whether TLS verification is disabled.
func (c ImmutableConfig) InsecureSkipVerify() bool {
	return c.insecureSkipVerify
}

// ServerAddress returns the HTTP server bind address (host:port).
func (c ImmutableConfig) ServerAddress() string {
	return c.serverAddress
}

// MetricsURI returns the metrics endpoint URI path.
func (c ImmutableConfig) MetricsURI() string {
	return c.metricsURI
}

// ScrapingInterval returns the job data time window.
func (c ImmutableConfig) ScrapingInterval() time.Duration {
	return c.scrapingInterval
}

// ScrapingIntervalString returns the scraping interval as a duration string.
func (c ImmutableConfig) ScrapingIntervalString() string {
	return c.scrapingInterval.String()
}

// LogName returns the log file name.
func (c ImmutableConfig) LogName() string {
	return c.logName
}

// OTelEnabled returns whether OpenTelemetry is enabled.
func (c ImmutableConfig) OTelEnabled() bool {
	return c.otelEnabled
}

// OTelEndpoint returns the OTLP endpoint address.
func (c ImmutableConfig) OTelEndpoint() string {
	return c.otelEndpoint
}

// OTelInsecure returns whether OTLP uses insecure connection.
func (c ImmutableConfig) OTelInsecure() bool {
	return c.otelInsecure
}

// OTelSamplingRate returns the trace sampling rate.
func (c ImmutableConfig) OTelSamplingRate() float64 {
	return c.otelSamplingRate
}

// MaskedAPIKey returns a masked version of the API key for safe logging.
func (c ImmutableConfig) MaskedAPIKey() string {
	if len(c.apiKey) <= 8 {
		return "****"
	}
	return c.apiKey[:4] + "****" + c.apiKey[len(c.apiKey)-4:]
}
