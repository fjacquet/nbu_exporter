// Package exporter provides interfaces for NetBackup API client abstraction.
// These interfaces enable better testability and allow for mock implementations
// in unit tests without requiring actual NetBackup server connectivity.
package exporter

import (
	"context"
)

// NetBackupClient defines the interface for interacting with the NetBackup REST API.
// This interface abstracts the HTTP client implementation and enables easy mocking
// in unit tests.
//
// Implementations must provide:
//   - Data fetching from NetBackup API endpoints
//   - API version detection and validation
//   - Resource cleanup
//
// The primary implementation is NbuClient, which uses Resty for HTTP communication.
type NetBackupClient interface {
	// FetchData sends an HTTP GET request to the specified URL and unmarshals
	// the JSON response into the provided target interface.
	//
	// Parameters:
	//   - ctx: Context for request cancellation and timeout
	//   - url: Complete URL to fetch (including query parameters)
	//   - target: Pointer to struct where JSON response will be unmarshaled
	//
	// Returns an error if the request fails, server returns non-2xx status,
	// or JSON unmarshaling fails.
	FetchData(ctx context.Context, url string, target interface{}) error

	// DetectAPIVersion attempts to detect and validate the NetBackup API version
	// by making a lightweight test request. This helps identify API compatibility
	// issues early in the application lifecycle.
	//
	// Parameters:
	//   - ctx: Context for request cancellation and timeout
	//
	// Returns:
	//   - The detected/validated API version string (e.g., "13.0")
	//   - Error if version detection fails or API is not accessible
	DetectAPIVersion(ctx context.Context) (string, error)

	// Close releases resources associated with the HTTP client, including
	// closing idle connections in the connection pool.
	//
	// Returns an error if the client is already closed or cleanup fails.
	Close() error
}
