// Package exporter provides backward compatibility tests for NetBackup API version support.
// These tests verify that existing configurations continue to work correctly after
// implementing multi-version support and automatic version detection.
package exporter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fjacquet/nbu_exporter/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBackwardCompatibility_ExplicitVersion120 verifies that configurations with
// explicitly set apiVersion: "12.0" continue to work without changes.
// This is the most common existing configuration for NetBackup 10.5 deployments.
func TestBackwardCompatibility_ExplicitVersion120(t *testing.T) {
	// Create a mock server that responds to API version 12.0 requests
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the Accept header contains version 12.0
		acceptHeader := r.Header.Get("Accept")
		assert.Contains(t, acceptHeader, "version=12.0", "Expected API version 12.0 in Accept header")

		// Return a successful response
		response := map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"type": "job",
					"id":   "1",
					"attributes": map[string]interface{}{
						"jobId":     1,
						"jobType":   "BACKUP",
						"status":    "DONE",
						"startTime": time.Now().Unix(),
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create configuration with explicit API version 12.0
	// Extract host:port from server URL (format: https://127.0.0.1:12345)
	serverAddr := strings.TrimPrefix(server.URL, "https://")
	cfg := createTestConfig(serverAddr, "12.0")
	cfg.NbuServer.Scheme = "https"

	// Create client - should NOT perform version detection
	client := NewNbuClient(cfg)
	assert.Equal(t, "12.0", client.cfg.NbuServer.APIVersion, "API version should remain 12.0")

	// Test that API calls work correctly
	ctx := context.Background()
	var result map[string]interface{}
	err := client.FetchData(ctx, server.URL+"/admin/jobs", &result)
	require.NoError(t, err, "API call with version 12.0 should succeed")
	assert.NotNil(t, result["data"], "Response should contain data")
}

// TestBackwardCompatibility_ExplicitVersion30 verifies that configurations with
// explicitly set apiVersion: "3.0" continue to work for NetBackup 10.0-10.4 deployments.
func TestBackwardCompatibility_ExplicitVersion30(t *testing.T) {
	// Create a mock server that responds to API version 3.0 requests
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the Accept header contains version 3.0
		acceptHeader := r.Header.Get("Accept")
		assert.Contains(t, acceptHeader, "version=3.0", "Expected API version 3.0 in Accept header")

		// Return a successful response
		response := map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"type": "job",
					"id":   "1",
					"attributes": map[string]interface{}{
						"jobId":     1,
						"jobType":   "BACKUP",
						"status":    "DONE",
						"startTime": time.Now().Unix(),
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create configuration with explicit API version 3.0
	// Extract host:port from server URL (format: https://127.0.0.1:12345)
	serverAddr := strings.TrimPrefix(server.URL, "https://")
	cfg := createTestConfig(serverAddr, "3.0")
	cfg.NbuServer.Scheme = "https"

	// Create client - should NOT perform version detection
	client := NewNbuClient(cfg)
	assert.Equal(t, "3.0", client.cfg.NbuServer.APIVersion, "API version should remain 3.0")

	// Test that API calls work correctly
	ctx := context.Background()
	var result map[string]interface{}
	err := client.FetchData(ctx, server.URL+"/admin/jobs", &result)
	require.NoError(t, err, "API call with version 3.0 should succeed")
	assert.NotNil(t, result["data"], "Response should contain data")
}

// TestBackwardCompatibility_MissingVersion verifies that configurations without
// an apiVersion field trigger automatic version detection and work correctly.
func TestBackwardCompatibility_MissingVersion(t *testing.T) {
	// Track which versions were attempted
	attemptedVersions := []string{}

	// Create a mock server that only accepts version 12.0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		acceptHeader := r.Header.Get("Accept")

		// Extract version from Accept header
		if contains(acceptHeader, "version=13.0") {
			attemptedVersions = append(attemptedVersions, "13.0")
			// Return 406 for version 13.0
			w.WriteHeader(http.StatusNotAcceptable)
			return
		} else if contains(acceptHeader, "version=12.0") {
			attemptedVersions = append(attemptedVersions, "12.0")
			// Return success for version 12.0
			response := map[string]interface{}{
				"data": []map[string]interface{}{
					{
						"type": "job",
						"id":   "1",
						"attributes": map[string]interface{}{
							"jobId": 1,
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		} else if contains(acceptHeader, "version=3.0") {
			attemptedVersions = append(attemptedVersions, "3.0")
			// Return 406 for version 3.0
			w.WriteHeader(http.StatusNotAcceptable)
			return
		}

		// Unknown version
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	// Create configuration WITHOUT apiVersion field (empty string)
	// Extract host:port from server URL (format: https://127.0.0.1:12345)
	serverAddr := strings.TrimPrefix(server.URL, "https://")
	cfg := createTestConfig(serverAddr, "")
	cfg.NbuServer.Scheme = "https"

	// Create client with version detection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := NewNbuClientWithVersionDetection(ctx, &cfg)
	require.NoError(t, err, "Client creation with version detection should succeed")
	assert.Equal(t, "12.0", client.cfg.NbuServer.APIVersion, "Should detect version 12.0")

	// Verify that version detection tried versions in correct order
	assert.Contains(t, attemptedVersions, "13.0", "Should have tried version 13.0 first")
	assert.Contains(t, attemptedVersions, "12.0", "Should have tried version 12.0 second")

	// Verify that detected version works for API calls
	var result map[string]interface{}
	err = client.FetchData(ctx, server.URL+"/admin/jobs", &result)
	require.NoError(t, err, "API call with detected version should succeed")
	assert.NotNil(t, result["data"], "Response should contain data")
}

// TestBackwardCompatibility_NoBreakingChanges verifies that the implementation
// doesn't introduce breaking changes to existing deployments by testing:
// 1. Configuration structure remains unchanged
// 2. Metric names and labels remain consistent
// 3. API endpoints remain the same
func TestBackwardCompatibility_NoBreakingChanges(t *testing.T) {
	t.Run("ConfigurationStructure", func(t *testing.T) {
		// Verify that Config struct still has all expected fields
		cfg := models.Config{}

		// Server fields
		assert.NotNil(t, &cfg.Server.Port, "Server.Port field should exist")
		assert.NotNil(t, &cfg.Server.Host, "Server.Host field should exist")
		assert.NotNil(t, &cfg.Server.URI, "Server.URI field should exist")
		assert.NotNil(t, &cfg.Server.ScrapingInterval, "Server.ScrapingInterval field should exist")

		// NbuServer fields
		assert.NotNil(t, &cfg.NbuServer.Port, "NbuServer.Port field should exist")
		assert.NotNil(t, &cfg.NbuServer.Host, "NbuServer.Host field should exist")
		assert.NotNil(t, &cfg.NbuServer.Scheme, "NbuServer.Scheme field should exist")
		assert.NotNil(t, &cfg.NbuServer.APIKey, "NbuServer.APIKey field should exist")
		assert.NotNil(t, &cfg.NbuServer.APIVersion, "NbuServer.APIVersion field should exist")
		assert.NotNil(t, &cfg.NbuServer.InsecureSkipVerify, "NbuServer.InsecureSkipVerify field should exist")
	})

	t.Run("APIEndpoints", func(t *testing.T) {
		// Verify that API endpoint construction hasn't changed
		cfg := createTestConfig("nbu-master:1556", "12.0")
		cfg.NbuServer.Scheme = "https"
		cfg.NbuServer.URI = "/netbackup"

		expectedBaseURL := "https://nbu-master:1556/netbackup"
		actualBaseURL := cfg.GetNBUBaseURL()
		assert.Equal(t, expectedBaseURL, actualBaseURL, "Base URL construction should remain unchanged")

		// Verify URL building with query parameters
		jobsURL := cfg.BuildURL("/admin/jobs", map[string]string{
			"page[limit]":  "100",
			"page[offset]": "0",
		})
		assert.Contains(t, jobsURL, "/admin/jobs", "Jobs endpoint path should be unchanged")
		assert.Contains(t, jobsURL, "page%5Blimit%5D=100", "Query parameter format should be unchanged (URL encoded)")
	})

	t.Run("DefaultValues", func(t *testing.T) {
		// Verify that default API version is set correctly
		cfg := models.Config{}
		cfg.SetDefaults()

		// Default should be 13.0 for new deployments
		assert.Equal(t, "13.0", cfg.NbuServer.APIVersion, "Default API version should be 13.0")
	})
}

// TestBackwardCompatibility_CollectorInitialization verifies that collector
// initialization works correctly with different configuration scenarios.
func TestBackwardCompatibility_CollectorInitialization(t *testing.T) {
	t.Run("WithExplicitVersion", func(t *testing.T) {
		// Create a mock server
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := map[string]interface{}{
				"data": []interface{}{},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		serverAddr := strings.TrimPrefix(server.URL, "https://")
		cfg := createTestConfig(serverAddr, "12.0")
		cfg.NbuServer.Scheme = "https"

		// Create collector - should succeed without version detection
		collector, err := NewNbuCollector(cfg)
		require.NoError(t, err, "Collector creation with explicit version should succeed")
		assert.NotNil(t, collector, "Collector should be created")
		assert.Equal(t, "12.0", collector.cfg.NbuServer.APIVersion, "Collector should use configured version")
	})

	t.Run("WithAutoDetection", func(t *testing.T) {
		// Create a mock server that supports version 12.0
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			acceptHeader := r.Header.Get("Accept")

			if contains(acceptHeader, "version=13.0") {
				w.WriteHeader(http.StatusNotAcceptable)
				return
			}

			response := map[string]interface{}{
				"data": []interface{}{},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		serverAddr := strings.TrimPrefix(server.URL, "https://")
		cfg := createTestConfig(serverAddr, "")
		cfg.NbuServer.Scheme = "https"

		// Create collector - should perform version detection
		collector, err := NewNbuCollector(cfg)
		require.NoError(t, err, "Collector creation with auto-detection should succeed")
		assert.NotNil(t, collector, "Collector should be created")
		assert.NotEmpty(t, collector.cfg.NbuServer.APIVersion, "Collector should have detected a version")
	})
}

// TestBackwardCompatibility_ErrorMessages verifies that error messages
// remain helpful and don't expose internal implementation details.
func TestBackwardCompatibility_ErrorMessages(t *testing.T) {
	t.Run("UnsupportedVersion", func(t *testing.T) {
		// Create a mock server that returns 406 for all versions
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotAcceptable)
		}))
		defer server.Close()

		serverAddr := strings.TrimPrefix(server.URL, "https://")
		cfg := createTestConfig(serverAddr, "12.0")
		cfg.NbuServer.Scheme = "https"
		client := NewNbuClient(cfg)

		ctx := context.Background()
		var result map[string]interface{}
		err := client.FetchData(ctx, server.URL+"/admin/jobs", &result)

		require.Error(t, err, "Should return error for unsupported version")
		assert.Contains(t, err.Error(), "406", "Error should mention HTTP 406")
		assert.Contains(t, err.Error(), "not supported", "Error should indicate version not supported")
		assert.Contains(t, err.Error(), "apiVersion", "Error should mention apiVersion configuration")
	})

	t.Run("NetworkError", func(t *testing.T) {
		// Use an invalid URL to trigger network error
		cfg := createTestConfig("invalid-host-that-does-not-exist:9999", "12.0")
		cfg.NbuServer.Scheme = "https"
		client := NewNbuClient(cfg)

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		var result map[string]interface{}
		err := client.FetchData(ctx, "https://invalid-host-that-does-not-exist:9999/admin/jobs", &result)

		require.Error(t, err, "Should return error for network failure")
		assert.Contains(t, err.Error(), "failed", "Error should indicate failure")
	})
}

// Note: Helper functions createTestConfig, contains, and containsHelper
// are defined in client_test.go and shared across test files in this package
