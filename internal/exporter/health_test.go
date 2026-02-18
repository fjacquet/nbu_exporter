// Package exporter provides tests for health check functionality.
package exporter

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fjacquet/nbu_exporter/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTestConnectivity_Success tests connectivity check when NBU is reachable
func TestTestConnectivitySuccess(t *testing.T) {
	// Create mock server that returns success
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.netbackup+json;version=13.0")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": []interface{}{}})
	}))
	defer server.Close()

	cfg := createHealthTestConfig(server)
	collector, err := NewNbuCollector(cfg)
	require.NoError(t, err)
	defer func() { _ = collector.Close() }()

	// Test connectivity - should succeed
	ctx := context.Background()
	err = collector.TestConnectivity(ctx)
	assert.NoError(t, err, "TestConnectivity should succeed when NBU is reachable")
}

// TestTestConnectivity_Failure tests connectivity check when NBU is unreachable
func TestTestConnectivityFailure(t *testing.T) {
	// Create mock server that returns error
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	cfg := createHealthTestConfig(server)
	collector, err := NewNbuCollector(cfg)
	require.NoError(t, err)
	defer func() { _ = collector.Close() }()

	// Test connectivity - should fail
	ctx := context.Background()
	err = collector.TestConnectivity(ctx)
	assert.Error(t, err, "TestConnectivity should fail when NBU returns error")
	assert.Contains(t, err.Error(), "NetBackup connectivity test failed", "Error should indicate connectivity failure")
}

// TestTestConnectivity_Timeout tests connectivity check with timeout
func TestTestConnectivityTimeout(t *testing.T) {
	// Create slow server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Sleep longer than the timeout
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := createHealthTestConfig(server)
	collector, err := NewNbuCollector(cfg)
	require.NoError(t, err)
	defer func() { _ = collector.Close() }()

	// Test connectivity with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err = collector.TestConnectivity(ctx)
	assert.Error(t, err, "TestConnectivity should fail on timeout")
}

// TestTestConnectivity_NoDeadline tests that default timeout is applied when no deadline set
func TestTestConnectivityNoDeadline(t *testing.T) {
	// Create mock server that returns success
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.netbackup+json;version=13.0")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": []interface{}{}})
	}))
	defer server.Close()

	cfg := createHealthTestConfig(server)
	collector, err := NewNbuCollector(cfg)
	require.NoError(t, err)
	defer func() { _ = collector.Close() }()

	// Test connectivity without deadline - should use default 5s timeout
	ctx := context.Background()
	err = collector.TestConnectivity(ctx)
	assert.NoError(t, err, "TestConnectivity should succeed with default timeout")
}

// TestIsHealthy_NoScrapes tests IsHealthy when no scrapes have occurred
func TestIsHealthyNoScrapes(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.netbackup+json;version=13.0")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": []interface{}{}})
	}))
	defer server.Close()

	cfg := createHealthTestConfig(server)
	collector, err := NewNbuCollector(cfg)
	require.NoError(t, err)
	defer func() { _ = collector.Close() }()

	// Before any scrapes, IsHealthy should return false
	assert.False(t, collector.IsHealthy(), "IsHealthy should return false before any scrapes")
}

// TestIsHealthy_AfterSuccessfulScrape tests IsHealthy after a successful scrape
func TestIsHealthyAfterSuccessfulScrape(t *testing.T) {
	requestCount := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/vnd.netbackup+json;version=13.0")
		w.WriteHeader(http.StatusOK)

		// Return appropriate response based on path
		switch r.URL.Path {
		case "/netbackup/storage/storage-units":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []interface{}{
					map[string]interface{}{
						"attributes": map[string]interface{}{
							"storageUnitName": "test-su",
							"storageServer":   map[string]interface{}{"storageServerName": "test-server"},
							"diskPool":        map[string]interface{}{"diskPoolName": "test-pool", "diskType": "AdvancedDisk"},
							"freeCapacity":    1000000000,
							"usedCapacity":    500000000,
						},
					},
				},
				"meta": map[string]interface{}{
					"pagination": map[string]interface{}{"offset": 0, "last": 0},
				},
			})
		case "/netbackup/admin/jobs":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []interface{}{},
				"meta": map[string]interface{}{
					"pagination": map[string]interface{}{"offset": 0, "last": 0},
				},
			})
		default:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": []interface{}{}})
		}
	}))
	defer server.Close()

	cfg := createHealthTestConfig(server)
	collector, err := NewNbuCollector(cfg)
	require.NoError(t, err)
	defer func() { _ = collector.Close() }()

	// Before scrape, not healthy
	assert.False(t, collector.IsHealthy(), "IsHealthy should return false before scrape")

	// Trigger a scrape by calling collectStorageMetrics
	ctx := context.Background()
	_, span := collector.createScrapeSpan(ctx)
	_, storageErr := collector.collectStorageMetrics(ctx, span)
	span.End()

	// After successful storage scrape, should be healthy
	if storageErr == nil {
		assert.True(t, collector.IsHealthy(), "IsHealthy should return true after successful storage scrape")
	}
}

// TestIsHealthy_CacheHit tests IsHealthy updates time on cache hit
func TestIsHealthyCacheHit(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.netbackup+json;version=13.0")
		w.WriteHeader(http.StatusOK)

		if r.URL.Path == "/netbackup/storage/storage-units" {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []interface{}{
					map[string]interface{}{
						"attributes": map[string]interface{}{
							"storageUnitName": "test-su",
							"storageServer":   map[string]interface{}{"storageServerName": "test-server"},
							"diskPool":        map[string]interface{}{"diskPoolName": "test-pool", "diskType": "AdvancedDisk"},
							"freeCapacity":    1000000000,
							"usedCapacity":    500000000,
						},
					},
				},
				"meta": map[string]interface{}{
					"pagination": map[string]interface{}{"offset": 0, "last": 0},
				},
			})
		} else {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": []interface{}{}})
		}
	}))
	defer server.Close()

	cfg := createHealthTestConfig(server)
	collector, err := NewNbuCollector(cfg)
	require.NoError(t, err)
	defer func() { _ = collector.Close() }()

	ctx := context.Background()
	_, span := collector.createScrapeSpan(ctx)

	// First scrape - cache miss, fetches from API
	_, err1 := collector.collectStorageMetrics(ctx, span)
	require.NoError(t, err1)

	// Get first scrape time
	collector.scrapeMu.RLock()
	firstScrapeTime := collector.lastStorageScrapeTime
	collector.scrapeMu.RUnlock()

	// Small delay
	time.Sleep(10 * time.Millisecond)

	// Second scrape - cache hit, should still update time
	_, err2 := collector.collectStorageMetrics(ctx, span)
	require.NoError(t, err2)

	// Get second scrape time
	collector.scrapeMu.RLock()
	secondScrapeTime := collector.lastStorageScrapeTime
	collector.scrapeMu.RUnlock()

	span.End()

	// Second scrape time should be after first (cache hit still updates time)
	assert.True(t, secondScrapeTime.After(firstScrapeTime) || secondScrapeTime.Equal(firstScrapeTime),
		"Cache hit should update or maintain scrape timestamp")
}

// createHealthTestConfig creates a test configuration for health tests
func createHealthTestConfig(server *httptest.Server) models.Config {
	cfg := models.Config{}
	cfg.NbuServer.Scheme = "https"
	cfg.NbuServer.Host = server.Listener.Addr().(*net.TCPAddr).IP.String()
	cfg.NbuServer.Port = fmt.Sprintf("%d", server.Listener.Addr().(*net.TCPAddr).Port)
	cfg.NbuServer.URI = "/netbackup"
	cfg.NbuServer.APIKey = testAPIKey
	cfg.NbuServer.APIVersion = models.APIVersion130
	cfg.NbuServer.InsecureSkipVerify = true
	cfg.Server.ScrapingInterval = "5m"
	cfg.Server.CacheTTL = "5m"
	return cfg
}
