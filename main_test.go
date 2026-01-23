// Package main contains integration tests for the NBU Exporter HTTP server,
// configuration validation, and lifecycle management.
package main

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/fjacquet/nbu_exporter/internal/models"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testdataPath returns the absolute path to the testdata directory.
func testdataPath(t *testing.T, filename string) string {
	t.Helper()
	// Get the directory containing this test file
	wd, err := os.Getwd()
	require.NoError(t, err, "Failed to get working directory")
	return filepath.Join(wd, "testdata", filename)
}

// createTestConfig creates a valid Config for testing without external dependencies.
func createTestConfig() models.Config {
	return models.Config{
		Server: struct {
			Port             string `yaml:"port"`
			Host             string `yaml:"host"`
			URI              string `yaml:"uri"`
			ScrapingInterval string `yaml:"scrapingInterval"`
			LogName          string `yaml:"logName"`
			CacheTTL         string `yaml:"cacheTTL"`
		}{
			Host:             "127.0.0.1",
			Port:             "0", // Let OS assign port
			URI:              "/metrics",
			ScrapingInterval: "5m",
			LogName:          "/dev/null",
			CacheTTL:         "5m",
		},
		NbuServer: struct {
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
		}{
			Scheme:             "https",
			Host:               "localhost",
			Port:               "1556",
			URI:                "/netbackup",
			APIKey:             "test-api-key-12345678",
			APIVersion:         "13.0",
			InsecureSkipVerify: true,
		},
	}
}

// TestNewServer verifies Server struct initialization.
func TestNewServer(t *testing.T) {
	cfg := createTestConfig()
	server := NewServer(cfg)

	assert.NotNil(t, server, "NewServer should return non-nil server")
	assert.NotNil(t, server.registry, "Server should have initialized Prometheus registry")
	assert.NotNil(t, server.serverErrChan, "Server should have initialized error channel")
	assert.Nil(t, server.telemetryManager, "Telemetry manager should be nil when OTel disabled")
	assert.Nil(t, server.httpSrv, "HTTP server should be nil before Start()")
}

// TestNewServerWithOTel verifies Server creation with OpenTelemetry enabled.
func TestNewServerWithOTel(t *testing.T) {
	cfg := createTestConfig()
	cfg.OpenTelemetry.Enabled = true
	cfg.OpenTelemetry.Endpoint = "localhost:4317"
	cfg.OpenTelemetry.SamplingRate = 1.0

	server := NewServer(cfg)

	assert.NotNil(t, server, "NewServer should return non-nil server")
	assert.NotNil(t, server.telemetryManager, "Telemetry manager should be initialized when OTel enabled")
}

// TestValidateConfig_Success verifies successful configuration validation.
func TestValidateConfig_Success(t *testing.T) {
	configPath := testdataPath(t, "valid_config.yaml")

	cfg, err := validateConfig(configPath)

	require.NoError(t, err, "validateConfig should succeed with valid config")
	require.NotNil(t, cfg, "Config should not be nil")
	assert.Equal(t, "127.0.0.1", cfg.Server.Host)
	assert.Equal(t, "/metrics", cfg.Server.URI)
	assert.Equal(t, "localhost", cfg.NbuServer.Host)
	assert.Equal(t, "13.0", cfg.NbuServer.APIVersion)
}

// TestValidateConfig_FileNotFound verifies error when config file doesn't exist.
func TestValidateConfig_FileNotFound(t *testing.T) {
	cfg, err := validateConfig("/nonexistent/path/config.yaml")

	require.Error(t, err, "validateConfig should fail with nonexistent file")
	assert.Nil(t, cfg, "Config should be nil on error")
	assert.Contains(t, err.Error(), "config file not found")
}

// TestValidateConfig_InvalidConfig verifies error when config is invalid.
func TestValidateConfig_InvalidConfig(t *testing.T) {
	configPath := testdataPath(t, "invalid_config.yaml")

	cfg, err := validateConfig(configPath)

	require.Error(t, err, "validateConfig should fail with invalid config")
	assert.Nil(t, cfg, "Config should be nil on error")
	assert.Contains(t, err.Error(), "invalid configuration")
}

// TestValidateConfig_MalformedYAML verifies error when YAML is malformed.
func TestValidateConfig_MalformedYAML(t *testing.T) {
	configPath := testdataPath(t, "malformed_config.yaml")

	cfg, err := validateConfig(configPath)

	require.Error(t, err, "validateConfig should fail with malformed YAML")
	assert.Nil(t, cfg, "Config should be nil on error")
}

// TestSetupLogging_Success verifies successful logging initialization.
func TestSetupLogging_Success(t *testing.T) {
	// Create a temp file for logging
	tmpFile, err := os.CreateTemp("", "test_log_*.log")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	cfg := createTestConfig()
	cfg.Server.LogName = tmpFile.Name()

	// Capture original log level
	originalLevel := logrus.GetLevel()
	defer logrus.SetLevel(originalLevel)

	err = setupLogging(cfg, false)
	require.NoError(t, err, "setupLogging should succeed")
}

// TestSetupLogging_DebugMode verifies debug mode enables debug level logging.
func TestSetupLogging_DebugMode(t *testing.T) {
	// Create a temp file for logging
	tmpFile, err := os.CreateTemp("", "test_log_*.log")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	cfg := createTestConfig()
	cfg.Server.LogName = tmpFile.Name()

	// Capture original log level
	originalLevel := logrus.GetLevel()
	defer logrus.SetLevel(originalLevel)

	err = setupLogging(cfg, true)
	require.NoError(t, err, "setupLogging with debug should succeed")
	assert.Equal(t, logrus.DebugLevel, logrus.GetLevel(), "Log level should be Debug")
}

// TestSetupLogging_InvalidPath verifies error when log path is invalid.
func TestSetupLogging_InvalidPath(t *testing.T) {
	cfg := createTestConfig()
	cfg.Server.LogName = "/nonexistent/directory/test.log"

	err := setupLogging(cfg, false)
	require.Error(t, err, "setupLogging should fail with invalid path")
	assert.Contains(t, err.Error(), "failed to initialize logging")
}

// TestHealthHandler verifies the /health endpoint returns 200 OK.
// When collector is nil (before Start()), returns "OK (starting)" to indicate startup phase.
func TestHealthHandler(t *testing.T) {
	cfg := createTestConfig()
	server := NewServer(cfg)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	server.healthHandler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code, "Health handler should return 200")
	// Before Start() is called, collector is nil, so returns "OK (starting)"
	assert.Equal(t, "OK (starting)\n", rec.Body.String(), "Health handler should return 'OK (starting)' before collector init")
}

// TestHealthHandler_AllMethods verifies health endpoint accepts various HTTP methods.
func TestHealthHandler_AllMethods(t *testing.T) {
	cfg := createTestConfig()
	server := NewServer(cfg)

	methods := []string{http.MethodGet, http.MethodHead, http.MethodPost}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/health", nil)
			rec := httptest.NewRecorder()

			server.healthHandler(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code, "Health handler should return 200 for %s", method)
		})
	}
}

// TestServerErrorChan verifies the error channel is accessible.
func TestServerErrorChan(t *testing.T) {
	cfg := createTestConfig()
	server := NewServer(cfg)

	errChan := server.ErrorChan()
	assert.NotNil(t, errChan, "ErrorChan should return non-nil channel")

	// Verify channel is receive-only (compile-time check)
	var _ <-chan error = server.ErrorChan()
}

// TestServerShutdown_NoStart verifies Shutdown is safe when Start wasn't called.
func TestServerShutdown_NoStart(t *testing.T) {
	cfg := createTestConfig()
	server := NewServer(cfg)

	// Shutdown without Start should not panic
	err := server.Shutdown()
	assert.NoError(t, err, "Shutdown without Start should not error")
}

// TestWaitForShutdown_Signal verifies shutdown on SIGINT.
func TestWaitForShutdown_Signal(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping signal test in short mode")
	}

	errChan := make(chan error, 1)

	// Create a done channel to signal test completion
	done := make(chan struct{})

	go func() {
		err := waitForShutdown(errChan)
		assert.NoError(t, err, "waitForShutdown should return nil on signal")
		close(done)
	}()

	// Give goroutine time to set up signal handler
	time.Sleep(50 * time.Millisecond)

	// Send signal to trigger shutdown
	p, err := os.FindProcess(os.Getpid())
	require.NoError(t, err)
	require.NoError(t, p.Signal(syscall.SIGINT))

	// Wait for completion with timeout
	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("waitForShutdown did not return in time")
	}

	// Reset signal handler
	signal.Reset(syscall.SIGINT, syscall.SIGTERM)
}

// TestWaitForShutdown_Error verifies server errors are propagated.
func TestWaitForShutdown_Error(t *testing.T) {
	errChan := make(chan error, 1)

	done := make(chan struct{})
	var gotErr error

	go func() {
		gotErr = waitForShutdown(errChan)
		close(done)
	}()

	// Give goroutine time to start
	time.Sleep(50 * time.Millisecond)

	// Send an actual error through channel
	testErr := http.ErrServerClosed
	errChan <- testErr

	// Wait for completion
	select {
	case <-done:
		// Server error should propagate through
		assert.Error(t, gotErr, "Error should be propagated from error channel")
		assert.Equal(t, testErr, gotErr, "Error should match the sent error")
	case <-time.After(2 * time.Second):
		t.Fatal("waitForShutdown did not return in time")
	}
}

// TestWaitForShutdown_NilError verifies handling of nil error on channel.
func TestWaitForShutdown_NilError(t *testing.T) {
	errChan := make(chan error, 1)

	done := make(chan struct{})
	var gotErr error

	go func() {
		gotErr = waitForShutdown(errChan)
		close(done)
	}()

	// Give goroutine time to start
	time.Sleep(50 * time.Millisecond)

	// Send nil error
	errChan <- nil

	select {
	case <-done:
		assert.NoError(t, gotErr, "waitForShutdown should return nil for nil error")
	case <-time.After(2 * time.Second):
		t.Fatal("waitForShutdown did not return in time")
	}
}

// TestServerStartShutdown_Integration tests the full server lifecycle.
// This test requires a mock NBU server to avoid actual API calls.
func TestServerStartShutdown_Integration(t *testing.T) {
	// Create a mock NBU server
	mockNBU := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return empty JSON for any request
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data": []}`))
	}))
	defer mockNBU.Close()

	// Parse mock server URL to get host and port
	host, port, err := net.SplitHostPort(mockNBU.Listener.Addr().String())
	require.NoError(t, err)

	// Create config pointing to mock server
	cfg := createTestConfig()
	cfg.NbuServer.Host = host
	cfg.NbuServer.Port = port
	cfg.NbuServer.Scheme = "https"
	cfg.NbuServer.InsecureSkipVerify = true

	// Create server
	server := NewServer(cfg)
	require.NotNil(t, server)

	// Start server
	err = server.Start()
	require.NoError(t, err, "Server.Start should succeed")

	// Verify HTTP server is running
	assert.NotNil(t, server.httpSrv, "HTTP server should be initialized after Start")

	// Shutdown server
	err = server.Shutdown()
	assert.NoError(t, err, "Server.Shutdown should succeed")
}

// TestServerPrometheusRegistry verifies custom registry isolation.
func TestServerPrometheusRegistry(t *testing.T) {
	cfg := createTestConfig()
	server1 := NewServer(cfg)
	server2 := NewServer(cfg)

	// Each server should have its own registry
	assert.NotSame(t, server1.registry, server2.registry, "Servers should have separate registries")

	// Registries should be independent from default registry
	assert.NotSame(t, server1.registry, prometheus.DefaultRegisterer, "Server registry should not be default")
}

// TestExtractTraceContextMiddleware verifies middleware doesn't break request flow.
func TestExtractTraceContextMiddleware(t *testing.T) {
	cfg := createTestConfig()
	cfg.OpenTelemetry.Enabled = true
	cfg.OpenTelemetry.Endpoint = "localhost:4317"
	cfg.OpenTelemetry.SamplingRate = 1.0

	server := NewServer(cfg)

	// Create a test handler to wrap
	handlerCalled := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("test"))
	})

	// Wrap with middleware
	wrappedHandler := server.extractTraceContextMiddleware(testHandler)

	// Make request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rec, req)

	assert.True(t, handlerCalled, "Handler should be called through middleware")
	assert.Equal(t, http.StatusOK, rec.Code, "Response should be 200 OK")
	assert.Equal(t, "test", rec.Body.String(), "Response body should be passed through")
}

// TestExtractTraceContextMiddleware_WithTraceHeader verifies trace context extraction.
func TestExtractTraceContextMiddleware_WithTraceHeader(t *testing.T) {
	cfg := createTestConfig()
	cfg.OpenTelemetry.Enabled = true
	cfg.OpenTelemetry.Endpoint = "localhost:4317"
	cfg.OpenTelemetry.SamplingRate = 1.0

	server := NewServer(cfg)

	var capturedContext context.Context
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedContext = r.Context()
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := server.extractTraceContextMiddleware(testHandler)

	// Create request with W3C trace context header
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	rec := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rec, req)

	assert.NotNil(t, capturedContext, "Context should be captured")
	assert.Equal(t, http.StatusOK, rec.Code, "Response should be 200 OK")
}

// TestExtractTraceContextMiddleware_NoTraceHeader verifies normal operation without trace.
func TestExtractTraceContextMiddleware_NoTraceHeader(t *testing.T) {
	cfg := createTestConfig()
	cfg.OpenTelemetry.Enabled = true
	cfg.OpenTelemetry.Endpoint = "localhost:4317"
	cfg.OpenTelemetry.SamplingRate = 1.0

	server := NewServer(cfg)

	var capturedContext context.Context
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedContext = r.Context()
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := server.extractTraceContextMiddleware(testHandler)

	// Create request without trace header
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rec, req)

	assert.NotNil(t, capturedContext, "Context should be present even without trace")
	assert.Equal(t, http.StatusOK, rec.Code, "Response should be 200 OK")
}

// TestServerErrorPropagation verifies server errors are sent through error channel.
func TestServerErrorPropagation(t *testing.T) {
	// Use a port that's already in use
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	// Parse the port
	_, port, err := net.SplitHostPort(listener.Addr().String())
	require.NoError(t, err)

	// Create a mock NBU server for the collector
	mockNBU := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data": []}`))
	}))
	defer mockNBU.Close()

	nbuHost, nbuPort, err := net.SplitHostPort(mockNBU.Listener.Addr().String())
	require.NoError(t, err)

	// Create config with the occupied port
	cfg := createTestConfig()
	cfg.Server.Host = "127.0.0.1"
	cfg.Server.Port = port // This port is occupied
	cfg.NbuServer.Host = nbuHost
	cfg.NbuServer.Port = nbuPort
	cfg.NbuServer.Scheme = "https"
	cfg.NbuServer.InsecureSkipVerify = true

	server := NewServer(cfg)

	// Start should succeed (starts async)
	err = server.Start()
	require.NoError(t, err, "Start should succeed even with port conflict (async)")

	// Wait for error on channel
	select {
	case serverErr := <-server.ErrorChan():
		assert.Error(t, serverErr, "Should receive error for port conflict")
		assert.Contains(t, serverErr.Error(), "HTTP server error")
	case <-time.After(2 * time.Second):
		t.Fatal("Expected error on error channel, but none received")
	}
}

// TestConfigGetServerAddress verifies server address construction.
func TestConfigGetServerAddress(t *testing.T) {
	cfg := createTestConfig()
	cfg.Server.Host = "0.0.0.0"
	cfg.Server.Port = "2112"

	addr := cfg.GetServerAddress()
	assert.Equal(t, "0.0.0.0:2112", addr)
}

// TestConfigGetNBUBaseURL verifies NBU base URL construction.
func TestConfigGetNBUBaseURL(t *testing.T) {
	cfg := createTestConfig()
	cfg.NbuServer.Scheme = "https"
	cfg.NbuServer.Host = "nbu-master.example.com"
	cfg.NbuServer.Port = "1556"
	cfg.NbuServer.URI = "/netbackup"

	url := cfg.GetNBUBaseURL()
	assert.Equal(t, "https://nbu-master.example.com:1556/netbackup", url)
}

// BenchmarkNewServer measures server creation performance.
func BenchmarkNewServer(b *testing.B) {
	cfg := createTestConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewServer(cfg)
	}
}

// BenchmarkHealthHandler measures health endpoint performance.
func BenchmarkHealthHandler(b *testing.B) {
	cfg := createTestConfig()
	server := NewServer(cfg)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		server.healthHandler(rec, req)
	}
}

// BenchmarkValidateConfig measures config validation performance.
func BenchmarkValidateConfig(b *testing.B) {
	// Get path relative to test file
	wd, err := os.Getwd()
	if err != nil {
		b.Fatal(err)
	}
	configPath := filepath.Join(wd, "testdata", "valid_config.yaml")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = validateConfig(configPath)
	}
}

// TestLogSuppression is a helper to suppress logrus output during tests.
func suppressLogs() func() {
	original := logrus.StandardLogger().Out
	logrus.SetOutput(&bytes.Buffer{})
	return func() {
		logrus.SetOutput(original)
	}
}
