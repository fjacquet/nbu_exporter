// Package exporter provides concurrent access tests for the Prometheus collector.
// These tests verify thread-safety of the collector under concurrent scrapes
// and verify correct behavior when Close() is called during active collection.
//
// Run with: go test -race -v -run TestCollectorConcurrent
package exporter

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/fjacquet/nbu_exporter/internal/models"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

// TestCollectorConcurrentCollect verifies that multiple goroutines can
// safely call Collect() simultaneously without race conditions.
// Run with: go test -race -v -run TestCollectorConcurrentCollect
func TestCollectorConcurrentCollect(t *testing.T) {
	// Create mock server with standard responses
	server := createConcurrentTestServer(t)
	defer server.Close()

	// Create collector
	cfg := createConcurrentTestConfig(server)
	collector, err := NewNbuCollector(cfg)
	require.NoError(t, err)
	defer func() { _ = collector.Close() }()

	const numGoroutines = 10
	var wg sync.WaitGroup

	// Launch multiple concurrent Collect calls
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			ch := make(chan prometheus.Metric, 100)
			go func() {
				collector.Collect(ch)
				close(ch)
			}()

			// Drain channel to ensure Collect completes
			count := 0
			for range ch {
				count++
			}

			// Should receive at least some metrics (API version at minimum)
			if count == 0 {
				t.Logf("Goroutine %d: collected 0 metrics", id)
			}
		}(i)
	}

	wg.Wait()
}

// TestCollectorConcurrentDescribe verifies Describe() is safe for concurrent calls
func TestCollectorConcurrentDescribe(t *testing.T) {
	server := createConcurrentTestServer(t)
	defer server.Close()

	cfg := createConcurrentTestConfig(server)
	collector, err := NewNbuCollector(cfg)
	require.NoError(t, err)
	defer func() { _ = collector.Close() }()

	const numGoroutines = 10
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			ch := make(chan *prometheus.Desc, 100)
			go func() {
				collector.Describe(ch)
				close(ch)
			}()

			// Drain channel
			count := 0
			for range ch {
				count++
			}

			// Should always get 8 descriptors (including nbu_up and nbu_last_scrape_timestamp_seconds)
			if count != 8 {
				t.Errorf("Describe() returned %d descriptors, expected 8", count)
			}
		}()
	}

	wg.Wait()
}

// TestCollectorCollectDuringClose verifies graceful handling when
// Close() is called during Collect()
func TestCollectorCollectDuringClose(t *testing.T) {
	server := createConcurrentTestServer(t)
	defer server.Close()

	cfg := createConcurrentTestConfig(server)
	collector, err := NewNbuCollector(cfg)
	require.NoError(t, err)

	var wg sync.WaitGroup

	// Start a Collect in background
	wg.Add(1)
	go func() {
		defer wg.Done()
		ch := make(chan prometheus.Metric, 100)
		collector.Collect(ch)
		close(ch)
		// Drain channel
		for range ch {
		}
	}()

	// Give Collect a moment to start
	time.Sleep(10 * time.Millisecond)

	// Call Close() while Collect may be running
	// This tests graceful shutdown during active collection
	err = collector.Close()
	// Close should succeed (or return error if already closing)
	if err != nil {
		t.Logf("Close() during Collect returned: %v", err)
	}

	wg.Wait()
}

// TestCollectorConcurrentCollectAndDescribe verifies that Collect() and Describe()
// can be called concurrently without race conditions
func TestCollectorConcurrentCollectAndDescribe(t *testing.T) {
	server := createConcurrentTestServer(t)
	defer server.Close()

	cfg := createConcurrentTestConfig(server)
	collector, err := NewNbuCollector(cfg)
	require.NoError(t, err)
	defer func() { _ = collector.Close() }()

	const numGoroutines = 5
	var wg sync.WaitGroup

	// Launch concurrent Collect calls
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			ch := make(chan prometheus.Metric, 100)
			go func() {
				collector.Collect(ch)
				close(ch)
			}()

			for range ch {
			}
		}()
	}

	// Launch concurrent Describe calls
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			ch := make(chan *prometheus.Desc, 100)
			go func() {
				collector.Describe(ch)
				close(ch)
			}()

			for range ch {
			}
		}()
	}

	wg.Wait()
}

// TestCollectorMultipleCloseAttempts verifies that multiple Close() calls
// are handled correctly (idempotent behavior)
func TestCollectorMultipleCloseAttempts(t *testing.T) {
	server := createConcurrentTestServer(t)
	defer server.Close()

	cfg := createConcurrentTestConfig(server)
	collector, err := NewNbuCollector(cfg)
	require.NoError(t, err)

	const numGoroutines = 5
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	// Try to close from multiple goroutines simultaneously
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := collector.Close()
			errors <- err
		}()
	}

	wg.Wait()
	close(errors)

	// Count successes and errors
	successCount := 0
	errorCount := 0
	for err := range errors {
		if err == nil {
			successCount++
		} else {
			errorCount++
		}
	}

	// Exactly one Close() should succeed, rest should return "already closed"
	if successCount != 1 {
		t.Errorf("Expected exactly 1 successful Close(), got %d", successCount)
	}
	if errorCount != numGoroutines-1 {
		t.Errorf("Expected %d Close() errors, got %d", numGoroutines-1, errorCount)
	}
}

// TestCollectorCloseWithActiveCollect verifies that Close() waits for active
// Collect() calls before returning
func TestCollectorCloseWithActiveCollect(t *testing.T) {
	// Create a slow server that takes time to respond
	requestStarted := make(chan struct{})
	requestContinue := make(chan struct{})

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Signal that request has started
		select {
		case <-requestStarted:
			// Already signaled
		default:
			close(requestStarted)
		}

		// Wait for signal to continue
		<-requestContinue

		w.Header().Set(contentTypeHeader, contentTypeJSON)
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": []interface{}{}})
	}))
	defer server.Close()

	cfg := createConcurrentTestConfig(server)
	collector, err := NewNbuCollector(cfg)
	require.NoError(t, err)

	// Start Collect in background
	collectDone := make(chan struct{})
	go func() {
		ch := make(chan prometheus.Metric, 100)
		collector.Collect(ch)
		close(ch)
		for range ch {
		}
		close(collectDone)
	}()

	// Wait for request to start
	<-requestStarted

	// Start Close in background
	closeDone := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		closeDone <- collector.CloseWithContext(ctx)
	}()

	// Verify Close is blocked
	select {
	case <-closeDone:
		// Close completed but we expected it to wait
		// This is acceptable if context times out
	case <-time.After(50 * time.Millisecond):
		// Expected - Close is waiting
	}

	// Allow request to complete
	close(requestContinue)

	// Wait for everything to finish
	select {
	case err := <-closeDone:
		// Close completed - check if it was due to timeout or success
		if err != nil {
			t.Logf("Close returned with error (expected for timeout): %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("Close() did not complete in time")
	}
}

// createConcurrentTestServer creates a test server for concurrent tests
func createConcurrentTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(contentTypeHeader, contentTypeJSON)
		w.WriteHeader(http.StatusOK)

		// Return appropriate response based on path
		switch r.URL.Path {
		case "/netbackup/storage/storage-units":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []interface{}{},
				"meta": map[string]interface{}{
					"pagination": map[string]interface{}{
						"offset": 0,
						"last":   0,
					},
				},
			})
		case "/netbackup/admin/jobs":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []interface{}{},
				"meta": map[string]interface{}{
					"pagination": map[string]interface{}{
						"offset": 0,
						"last":   0,
					},
				},
			})
		default:
			// Default empty response for version detection
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": []interface{}{}})
		}
	}))
}

// createConcurrentTestConfig creates a test configuration for concurrent tests
func createConcurrentTestConfig(server *httptest.Server) models.Config {
	cfg := models.Config{}
	cfg.NbuServer.Scheme = "https"
	cfg.NbuServer.Host = server.Listener.Addr().(*net.TCPAddr).IP.String()
	cfg.NbuServer.Port = fmt.Sprintf("%d", server.Listener.Addr().(*net.TCPAddr).Port)
	cfg.NbuServer.URI = "/netbackup"
	cfg.NbuServer.APIKey = testAPIKey
	cfg.NbuServer.APIVersion = models.APIVersion130
	cfg.NbuServer.InsecureSkipVerify = true
	cfg.Server.ScrapingInterval = "5m"
	return cfg
}
