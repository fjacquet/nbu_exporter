package telemetry

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// Test constants to avoid string duplication
const (
	testEndpoint               = "localhost:4317"
	testServiceName            = "nbu-exporter"
	testServiceVersion         = "1.0.0"
	testServiceVersionTest     = "1.0.0-test"
	testNetBackupServer        = "nbu-master"
	testNetBackupServerTest    = "nbu-test"
	errMsgInitializeUnexpected = "Initialize() unexpected error = %v"
	errMsgShutdownUnexpected   = "Shutdown() unexpected error = %v"
)

// TestNewManager tests the creation of a new telemetry manager
func TestNewManager(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{
			name: "creates manager with enabled config",
			config: Config{
				Enabled:         true,
				Endpoint:        testEndpoint,
				Insecure:        true,
				SamplingRate:    0.1,
				ServiceName:     testServiceName,
				ServiceVersion:  testServiceVersion,
				NetBackupServer: testNetBackupServer,
			},
		},
		{
			name: "creates manager with disabled config",
			config: Config{
				Enabled: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewManager(tt.config)

			if manager == nil {
				t.Fatal("NewManager() returned nil")
			}

			if manager.enabled != tt.config.Enabled {
				t.Errorf("NewManager() enabled = %v, want %v", manager.enabled, tt.config.Enabled)
			}

			if manager.config.Endpoint != tt.config.Endpoint {
				t.Errorf("NewManager() endpoint = %v, want %v", manager.config.Endpoint, tt.config.Endpoint)
			}
		})
	}
}

// TestManager_Initialize_Disabled tests initialization when telemetry is disabled
func TestManagerInitializeDisabled(t *testing.T) {
	config := Config{
		Enabled: false,
	}

	manager := NewManager(config)
	ctx := context.Background()

	err := manager.Initialize(ctx)
	if err != nil {
		t.Errorf(errMsgInitializeUnexpected, err)
	}

	if manager.tracerProvider != nil {
		t.Error("Initialize() should not create tracer provider when disabled")
	}

	if manager.IsEnabled() {
		t.Error("IsEnabled() should return false when disabled")
	}
}

// TestManager_Initialize_InvalidEndpoint tests initialization with invalid endpoint
// Note: OTLP exporter creation succeeds even with invalid endpoints - it only fails
// when actually trying to send data. This test verifies that initialization completes
// without errors, and the manager can be used (spans will be dropped if collector is unavailable).
// This is an example of graceful degradation - the system continues to operate even when
// the telemetry backend is unavailable, allowing the exporter to function without tracing.
func TestManagerInitializeInvalidEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
	}{
		{
			name:     "empty endpoint",
			endpoint: "",
		},
		{
			name:     "invalid endpoint format",
			endpoint: "not-a-valid-endpoint",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{
				Enabled:        true,
				Endpoint:       tt.endpoint,
				Insecure:       true,
				SamplingRate:   1.0,
				ServiceName:    testServiceName,
				ServiceVersion: testServiceVersion,
			}

			manager := NewManager(config)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			// Initialize should not return error (exporter creation succeeds even with invalid endpoints)
			// Graceful degradation: telemetry failures don't prevent application startup
			err := manager.Initialize(ctx)
			if err != nil {
				t.Errorf(errMsgInitializeUnexpected, err)
			}

			// Manager will be enabled even with invalid endpoint
			// Spans will be dropped if collector is unavailable
			if !manager.IsEnabled() {
				t.Error("IsEnabled() should return true after initialization (even with invalid endpoint)")
			}

			// Clean up - explicitly check shutdown error for proper resource cleanup
			if manager.IsEnabled() {
				shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer shutdownCancel()
				if err := manager.Shutdown(shutdownCtx); err != nil {
					t.Logf("Shutdown returned error (expected with invalid endpoint): %v", err)
				}
			}
		})
	}
}

// TestManager_Initialize_ValidConfig tests successful initialization
// Note: This test requires a real OTLP collector or will gracefully fail.
// Graceful degradation: if no collector is available, the manager initializes but
// may disable itself, allowing the application to continue without tracing.
func TestManagerInitializeValidConfig(t *testing.T) {
	config := Config{
		Enabled:         true,
		Endpoint:        testEndpoint,
		Insecure:        true,
		SamplingRate:    1.0,
		ServiceName:     testServiceName + "-test",
		ServiceVersion:  testServiceVersionTest,
		NetBackupServer: testNetBackupServerTest,
	}

	manager := NewManager(config)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := manager.Initialize(ctx)
	if err != nil {
		t.Errorf(errMsgInitializeUnexpected, err)
	}

	// If initialization succeeded, tracer provider should be set
	// If it failed (no collector available), manager should be disabled
	if manager.IsEnabled() && manager.tracerProvider == nil {
		t.Error("Initialize() enabled but tracer provider is nil")
	}

	// Clean up - explicitly check shutdown error for proper resource cleanup
	if manager.IsEnabled() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := manager.Shutdown(shutdownCtx); err != nil {
			t.Errorf(errMsgShutdownUnexpected, err)
		}
	}
}

// TestManager_Shutdown_NotInitialized tests shutdown when not initialized
func TestManagerShutdownNotInitialized(t *testing.T) {
	config := Config{
		Enabled: false,
	}

	manager := NewManager(config)
	ctx := context.Background()

	err := manager.Shutdown(ctx)
	if err != nil {
		t.Errorf(errMsgShutdownUnexpected, err)
	}
}

// TestManager_Shutdown_WithTimeout tests shutdown with context timeout
// Graceful degradation: initialization errors are ignored to allow testing shutdown behavior
// even when the telemetry backend is unavailable.
func TestManagerShutdownWithTimeout(t *testing.T) {
	config := Config{
		Enabled:        true,
		Endpoint:       testEndpoint,
		Insecure:       true,
		SamplingRate:   1.0,
		ServiceName:    testServiceName + "-test",
		ServiceVersion: testServiceVersionTest,
	}

	manager := NewManager(config)
	initCtx, initCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer initCancel()

	// Graceful degradation: ignore initialization errors (collector may not be available)
	if err := manager.Initialize(initCtx); err != nil {
		t.Logf("Initialize returned error (expected if no collector available): %v", err)
	}

	// Only test shutdown if initialization succeeded
	if manager.IsEnabled() {
		// Create a context with timeout for shutdown
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		err := manager.Shutdown(shutdownCtx)
		if err != nil {
			t.Errorf(errMsgShutdownUnexpected, err)
		}
	}
}

// TestManager_IsEnabled tests the IsEnabled method
// Graceful degradation: initialization errors are ignored as the test focuses on
// the enabled state behavior, not initialization success.
func TestManagerIsEnabled(t *testing.T) {
	tests := []struct {
		name           string
		config         Config
		expectedBefore bool
		initFails      bool
		expectedAfter  bool
	}{
		{
			name: "enabled in config",
			config: Config{
				Enabled:        true,
				Endpoint:       testEndpoint,
				Insecure:       true,
				SamplingRate:   1.0,
				ServiceName:    "test",
				ServiceVersion: testServiceVersion,
			},
			expectedBefore: true,
			// After init, may be disabled if no collector available (graceful degradation)
		},
		{
			name: "disabled in config",
			config: Config{
				Enabled: false,
			},
			expectedBefore: false,
			expectedAfter:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewManager(tt.config)

			// Check initial state
			if manager.IsEnabled() != tt.expectedBefore {
				t.Errorf("IsEnabled() before init = %v, want %v", manager.IsEnabled(), tt.expectedBefore)
			}

			// Initialize - gracefully handle errors (collector may not be available)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if err := manager.Initialize(ctx); err != nil {
				t.Logf("Initialize returned error (expected if no collector available): %v", err)
			}

			// After initialization, check if state matches expectation
			// Note: If config is enabled but init fails, IsEnabled() should return false
			if !tt.config.Enabled && manager.IsEnabled() {
				t.Error("IsEnabled() should return false when disabled in config")
			}

			// Clean up - explicitly check shutdown error
			if manager.IsEnabled() {
				shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer shutdownCancel()
				if err := manager.Shutdown(shutdownCtx); err != nil {
					t.Logf("Shutdown returned error: %v", err)
				}
			}
		})
	}
}

// TestManager_CreateSampler tests sampler creation logic
func TestManagerCreateSampler(t *testing.T) {
	tests := []struct {
		name         string
		samplingRate float64
		description  string
	}{
		{
			name:         "always sample when rate is 1.0",
			samplingRate: 1.0,
			description:  "Should use AlwaysSample",
		},
		{
			name:         "ratio-based sampling when rate is 0.1",
			samplingRate: 0.1,
			description:  "Should use TraceIDRatioBased",
		},
		{
			name:         "ratio-based sampling when rate is 0.5",
			samplingRate: 0.5,
			description:  "Should use TraceIDRatioBased",
		},
		{
			name:         "no sampling when rate is 0.0",
			samplingRate: 0.0,
			description:  "Should use TraceIDRatioBased with 0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{
				Enabled:      true,
				SamplingRate: tt.samplingRate,
			}

			manager := NewManager(config)
			sampler := manager.createSampler()

			if sampler == nil {
				t.Error("createSampler() returned nil")
			}

			// We can't easily test the sampler type without reflection,
			// but we can verify it was created
		})
	}
}

// TestManager_CreateResource tests resource creation
func TestManagerCreateResource(t *testing.T) {
	tests := []struct {
		name            string
		serviceName     string
		serviceVersion  string
		netBackupServer string
	}{
		{
			name:            "creates resource with all attributes",
			serviceName:     testServiceName,
			serviceVersion:  testServiceVersion,
			netBackupServer: testNetBackupServer,
		},
		{
			name:            "creates resource without NetBackup server",
			serviceName:     testServiceName,
			serviceVersion:  testServiceVersion,
			netBackupServer: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{
				ServiceName:     tt.serviceName,
				ServiceVersion:  tt.serviceVersion,
				NetBackupServer: tt.netBackupServer,
			}

			manager := NewManager(config)
			resource, err := manager.createResource()

			// Explicitly check for errors - resource creation should always succeed
			if err != nil {
				t.Errorf("createResource() unexpected error = %v", err)
			}

			if resource == nil {
				t.Error("createResource() returned nil resource")
			}

			// Verify resource attributes are set
			attrs := resource.Attributes()
			if len(attrs) == 0 {
				t.Error("createResource() returned resource with no attributes")
			}
		})
	}
}

// TestManager_DisabledMode tests that disabled mode works correctly
func TestManagerDisabledMode(t *testing.T) {
	config := Config{
		Enabled: false,
	}

	manager := NewManager(config)
	ctx := context.Background()

	// Initialize should succeed
	err := manager.Initialize(ctx)
	if err != nil {
		t.Errorf(errMsgInitializeUnexpected, err)
	}

	// IsEnabled should return false
	if manager.IsEnabled() {
		t.Error("IsEnabled() should return false in disabled mode")
	}

	// Shutdown should succeed
	err = manager.Shutdown(ctx)
	if err != nil {
		t.Errorf(errMsgShutdownUnexpected, err)
	}

	// TracerProvider should be nil
	if manager.tracerProvider != nil {
		t.Error("tracerProvider should be nil in disabled mode")
	}
}

// TestManagerTracerProvider tests TracerProvider() method
func TestManagerTracerProvider(t *testing.T) {
	t.Run("returns nil when disabled", func(t *testing.T) {
		config := Config{
			Enabled: false,
		}

		manager := NewManager(config)
		ctx := context.Background()

		// Initialize succeeds for disabled config
		err := manager.Initialize(ctx)
		if err != nil {
			t.Errorf(errMsgInitializeUnexpected, err)
		}

		// TracerProvider should return nil when disabled
		tp := manager.TracerProvider()
		if tp != nil {
			t.Error("TracerProvider() should return nil when disabled")
		}
	})

	t.Run("returns provider when enabled and initialized", func(t *testing.T) {
		config := Config{
			Enabled:        true,
			Endpoint:       testEndpoint,
			Insecure:       true,
			SamplingRate:   1.0,
			ServiceName:    testServiceName,
			ServiceVersion: testServiceVersion,
		}

		manager := NewManager(config)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		err := manager.Initialize(ctx)
		if err != nil {
			t.Errorf(errMsgInitializeUnexpected, err)
		}

		// If initialization succeeded, TracerProvider should return non-nil
		if manager.IsEnabled() {
			tp := manager.TracerProvider()
			if tp == nil {
				t.Error("TracerProvider() should return non-nil when enabled and initialized")
			}

			// Clean up
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer shutdownCancel()
			if err := manager.Shutdown(shutdownCtx); err != nil {
				t.Logf("Shutdown returned error: %v", err)
			}
		}
	})
}

// TestManagerTracerProviderBeforeInit tests TracerProvider before Initialize is called
func TestManagerTracerProviderBeforeInit(t *testing.T) {
	config := Config{
		Enabled:        true,
		Endpoint:       testEndpoint,
		Insecure:       true,
		SamplingRate:   1.0,
		ServiceName:    testServiceName,
		ServiceVersion: testServiceVersion,
	}

	manager := NewManager(config)

	// TracerProvider should return nil before initialization
	tp := manager.TracerProvider()
	if tp != nil {
		t.Error("TracerProvider() should return nil before Initialize() is called")
	}
}

// TestManagerSamplingRateEdgeCases tests sampling rate edge cases
func TestManagerSamplingRateEdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		samplingRate float64
		description  string
	}{
		{
			name:         "rate greater than 1.0 uses AlwaysSample",
			samplingRate: 1.5,
			description:  "Should use AlwaysSample for rates >= 1.0",
		},
		{
			name:         "rate equal to 1.0 uses AlwaysSample",
			samplingRate: 1.0,
			description:  "Should use AlwaysSample for rate == 1.0",
		},
		{
			name:         "rate of 0.99 uses TraceIDRatioBased",
			samplingRate: 0.99,
			description:  "Should use TraceIDRatioBased for rate < 1.0",
		},
		{
			name:         "rate of 0 uses TraceIDRatioBased with 0",
			samplingRate: 0.0,
			description:  "Should use TraceIDRatioBased(0) - effectively never sample",
		},
		{
			name:         "negative rate uses TraceIDRatioBased",
			samplingRate: -0.5,
			description:  "Should use TraceIDRatioBased for negative rates",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{
				Enabled:      true,
				SamplingRate: tt.samplingRate,
			}

			manager := NewManager(config)
			sampler := manager.createSampler()

			if sampler == nil {
				t.Error("createSampler() returned nil")
			}

			// Verify sampler is created - we can check the Description to verify type
			desc := sampler.Description()
			if desc == "" {
				t.Error("sampler.Description() returned empty string")
			}

			// For rates >= 1.0, should use AlwaysOnSampler
			if tt.samplingRate >= 1.0 {
				if desc != "AlwaysOnSampler" {
					t.Errorf("expected AlwaysOnSampler for rate %.2f, got %s", tt.samplingRate, desc)
				}
			}
		})
	}
}

// TestManagerDoubleInitialize tests calling Initialize twice
func TestManagerDoubleInitialize(t *testing.T) {
	config := Config{
		Enabled:        true,
		Endpoint:       testEndpoint,
		Insecure:       true,
		SamplingRate:   1.0,
		ServiceName:    testServiceName,
		ServiceVersion: testServiceVersion,
	}

	manager := NewManager(config)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// First initialization
	err := manager.Initialize(ctx)
	if err != nil {
		t.Errorf(errMsgInitializeUnexpected, err)
	}

	// Second initialization - should not panic
	err = manager.Initialize(ctx)
	if err != nil {
		t.Errorf("Second Initialize() unexpected error = %v", err)
	}

	// Clean up
	if manager.IsEnabled() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := manager.Shutdown(shutdownCtx); err != nil {
			t.Logf("Shutdown returned error: %v", err)
		}
	}
}

// TestManagerDoubleShutdown tests calling Shutdown twice
func TestManagerDoubleShutdown(t *testing.T) {
	config := Config{
		Enabled:        true,
		Endpoint:       testEndpoint,
		Insecure:       true,
		SamplingRate:   1.0,
		ServiceName:    testServiceName,
		ServiceVersion: testServiceVersion,
	}

	manager := NewManager(config)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := manager.Initialize(ctx)
	if err != nil {
		t.Errorf(errMsgInitializeUnexpected, err)
	}

	// First shutdown
	shutdownCtx1, shutdownCancel1 := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel1()
	err = manager.Shutdown(shutdownCtx1)
	if err != nil {
		t.Logf("First Shutdown returned error (may be expected): %v", err)
	}

	// Second shutdown - should not panic and handle gracefully
	shutdownCtx2, shutdownCancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel2()
	err = manager.Shutdown(shutdownCtx2)
	// Second shutdown should return nil (already shutdown) or handle gracefully
	if err != nil {
		t.Logf("Second Shutdown returned error (may be expected): %v", err)
	}
}

// TestManagerInitializeContextCancelled tests Initialize with an already cancelled context
func TestManagerInitializeContextCancelled(t *testing.T) {
	config := Config{
		Enabled:        true,
		Endpoint:       testEndpoint,
		Insecure:       true,
		SamplingRate:   1.0,
		ServiceName:    testServiceName,
		ServiceVersion: testServiceVersion,
	}

	manager := NewManager(config)

	// Create an already cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Initialize with cancelled context
	// Note: OTLP exporter creation may succeed even with cancelled context
	// (gRPC connection is async), but we verify the function handles it gracefully
	err := manager.Initialize(ctx)
	// Should not return error (graceful degradation) - exporter creation is async
	if err != nil {
		t.Errorf("Initialize() with cancelled context should not return error: %v", err)
	}

	// Clean up if initialized
	if manager.IsEnabled() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		_ = manager.Shutdown(shutdownCtx)
	}
}

// TestManagerShutdownContextTimeout tests Shutdown with a very short timeout
func TestManagerShutdownContextTimeout(t *testing.T) {
	config := Config{
		Enabled:        true,
		Endpoint:       testEndpoint,
		Insecure:       true,
		SamplingRate:   1.0,
		ServiceName:    testServiceName,
		ServiceVersion: testServiceVersion,
	}

	manager := NewManager(config)
	initCtx, initCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer initCancel()

	err := manager.Initialize(initCtx)
	if err != nil {
		t.Errorf(errMsgInitializeUnexpected, err)
	}

	if manager.IsEnabled() {
		// Create a very short timeout context for shutdown
		// Note: actual timeout behavior depends on TracerProvider implementation
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer shutdownCancel()

		// Shutdown may or may not error depending on timing
		// The key is that it doesn't panic
		err = manager.Shutdown(shutdownCtx)
		// We accept both success and timeout error
		if err != nil {
			t.Logf("Shutdown with short timeout returned error (expected): %v", err)
		}
	}
}

// TestManagerCreateResourceHostnameError tests resource creation when hostname retrieval fails
// Note: We can't easily simulate os.Hostname() failure, but we verify fallback behavior
func TestManagerCreateResourceHostnameError(t *testing.T) {
	config := Config{
		Enabled:         true,
		ServiceName:     testServiceName,
		ServiceVersion:  testServiceVersion,
		NetBackupServer: testNetBackupServer,
	}

	manager := NewManager(config)
	resource, err := manager.createResource()

	if err != nil {
		t.Errorf("createResource() unexpected error = %v", err)
	}

	if resource == nil {
		t.Error("createResource() returned nil resource")
	}

	// Verify resource has host.name attribute (either actual hostname or "unknown")
	attrs := resource.Attributes()
	hostNameFound := false
	for _, attr := range attrs {
		if attr.Key == "host.name" {
			hostNameFound = true
			// Value should not be empty
			if attr.Value.AsString() == "" {
				t.Error("host.name attribute should not be empty")
			}
		}
	}
	if !hostNameFound {
		t.Error("createResource() should include host.name attribute")
	}
}

// TestManagerCreateResourceWithoutNetBackupServer tests resource creation without NetBackup server
func TestManagerCreateResourceWithoutNetBackupServer(t *testing.T) {
	config := Config{
		Enabled:         true,
		ServiceName:     testServiceName,
		ServiceVersion:  testServiceVersion,
		NetBackupServer: "", // Empty NetBackup server
	}

	manager := NewManager(config)
	resource, err := manager.createResource()

	if err != nil {
		t.Errorf("createResource() unexpected error = %v", err)
	}

	if resource == nil {
		t.Error("createResource() returned nil resource")
	}

	// Verify resource does NOT have peer.service attribute
	attrs := resource.Attributes()
	for _, attr := range attrs {
		if attr.Key == "peer.service" {
			t.Error("createResource() should not include peer.service when NetBackupServer is empty")
		}
	}
}

// TestManagerConfigFields tests Config struct field validation
func TestManagerConfigFields(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{
			name: "full config",
			config: Config{
				Enabled:         true,
				Endpoint:        testEndpoint,
				Insecure:        true,
				SamplingRate:    0.5,
				ServiceName:     testServiceName,
				ServiceVersion:  testServiceVersion,
				NetBackupServer: testNetBackupServer,
			},
		},
		{
			name: "minimal enabled config",
			config: Config{
				Enabled:        true,
				Endpoint:       testEndpoint,
				Insecure:       true,
				SamplingRate:   1.0,
				ServiceName:    "minimal",
				ServiceVersion: "0.0.1",
			},
		},
		{
			name: "secure config (Insecure=false)",
			config: Config{
				Enabled:        true,
				Endpoint:       "secure-endpoint:4317",
				Insecure:       false,
				SamplingRate:   0.1,
				ServiceName:    "secure-service",
				ServiceVersion: "1.0.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewManager(tt.config)

			// Verify config is stored correctly
			if manager.config.Enabled != tt.config.Enabled {
				t.Errorf("config.Enabled = %v, want %v", manager.config.Enabled, tt.config.Enabled)
			}
			if manager.config.Endpoint != tt.config.Endpoint {
				t.Errorf("config.Endpoint = %v, want %v", manager.config.Endpoint, tt.config.Endpoint)
			}
			if manager.config.Insecure != tt.config.Insecure {
				t.Errorf("config.Insecure = %v, want %v", manager.config.Insecure, tt.config.Insecure)
			}
			if manager.config.SamplingRate != tt.config.SamplingRate {
				t.Errorf("config.SamplingRate = %v, want %v", manager.config.SamplingRate, tt.config.SamplingRate)
			}
			if manager.config.ServiceName != tt.config.ServiceName {
				t.Errorf("config.ServiceName = %v, want %v", manager.config.ServiceName, tt.config.ServiceName)
			}
			if manager.config.ServiceVersion != tt.config.ServiceVersion {
				t.Errorf("config.ServiceVersion = %v, want %v", manager.config.ServiceVersion, tt.config.ServiceVersion)
			}
			if manager.config.NetBackupServer != tt.config.NetBackupServer {
				t.Errorf("config.NetBackupServer = %v, want %v", manager.config.NetBackupServer, tt.config.NetBackupServer)
			}
		})
	}
}

// TestManagerHighSamplingRate tests sampling rates greater than 1.0
func TestManagerHighSamplingRate(t *testing.T) {
	rates := []float64{1.5, 2.0, 10.0, 100.0}

	for _, rate := range rates {
		t.Run("rate_"+fmt.Sprintf("%.1f", rate), func(t *testing.T) {
			config := Config{
				Enabled:      true,
				SamplingRate: rate,
			}

			manager := NewManager(config)
			sampler := manager.createSampler()

			if sampler == nil {
				t.Error("createSampler() returned nil")
			}

			// High rates should use AlwaysOnSampler
			desc := sampler.Description()
			if desc != "AlwaysOnSampler" {
				t.Errorf("expected AlwaysOnSampler for rate %.1f, got %s", rate, desc)
			}
		})
	}
}

// TestManagerInitializeSecureMode tests initialization with secure TLS mode
func TestManagerInitializeSecureMode(t *testing.T) {
	config := Config{
		Enabled:        true,
		Endpoint:       testEndpoint,
		Insecure:       false, // Secure mode
		SamplingRate:   1.0,
		ServiceName:    testServiceName,
		ServiceVersion: testServiceVersion,
	}

	manager := NewManager(config)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Initialize should not panic even with secure mode (though it may fail without valid TLS)
	err := manager.Initialize(ctx)
	// We accept both success and failure - the key is no panic
	if err != nil {
		t.Logf("Initialize with secure mode returned error (may be expected without valid TLS): %v", err)
	}

	// Clean up
	if manager.IsEnabled() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		_ = manager.Shutdown(shutdownCtx)
	}
}

// TestManagerZeroValueConfig tests manager with zero value Config
func TestManagerZeroValueConfig(t *testing.T) {
	config := Config{} // Zero value config

	manager := NewManager(config)

	// Should not panic
	if manager == nil {
		t.Fatal("NewManager() returned nil for zero value config")
	}

	// Should be disabled by default
	if manager.IsEnabled() {
		t.Error("Manager with zero value config should be disabled")
	}

	// TracerProvider should return nil
	tp := manager.TracerProvider()
	if tp != nil {
		t.Error("TracerProvider() should return nil for disabled manager")
	}

	// Initialize and Shutdown should succeed without errors
	ctx := context.Background()
	if err := manager.Initialize(ctx); err != nil {
		t.Errorf("Initialize() with zero value config should not error: %v", err)
	}
	if err := manager.Shutdown(ctx); err != nil {
		t.Errorf("Shutdown() with zero value config should not error: %v", err)
	}
}

// TestManagerIsEnabledAfterInitFailure tests that IsEnabled returns false after init failure
func TestManagerIsEnabledAfterInitFailure(t *testing.T) {
	// This test verifies graceful degradation behavior
	config := Config{
		Enabled:        true,
		Endpoint:       testEndpoint,
		Insecure:       true,
		SamplingRate:   1.0,
		ServiceName:    testServiceName,
		ServiceVersion: testServiceVersion,
	}

	manager := NewManager(config)

	// Before init, enabled should reflect config
	if !manager.enabled {
		t.Error("Manager should have enabled=true from config before init")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Initialize (may succeed or fail depending on environment)
	err := manager.Initialize(ctx)
	if err != nil {
		t.Logf("Initialize returned error: %v", err)
	}

	// If init failed and manager is disabled, verify consistent state
	if !manager.IsEnabled() {
		tp := manager.TracerProvider()
		if tp != nil {
			t.Error("TracerProvider() should return nil when manager is disabled after init failure")
		}
	}

	// Clean up
	if manager.IsEnabled() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		_ = manager.Shutdown(shutdownCtx)
	}
}

// TestManagerGracefulDegradationExporterError simulates exporter error handling
// by testing the manager's state after simulating an initialization failure
func TestManagerGracefulDegradationExporterError(t *testing.T) {
	// Create a manager and manually set the state that would occur after exporter failure
	config := Config{
		Enabled:        true,
		Endpoint:       testEndpoint,
		Insecure:       true,
		SamplingRate:   1.0,
		ServiceName:    testServiceName,
		ServiceVersion: testServiceVersion,
	}

	manager := NewManager(config)

	// Verify initial state
	if !manager.enabled {
		t.Error("Manager should start with enabled=true from config")
	}

	// Simulate what happens after exporter creation failure:
	// The manager sets enabled = false and returns nil (no error)
	// This is the graceful degradation behavior

	// We can test that when enabled is false after "init failure",
	// all subsequent operations work correctly
	manager.enabled = false // Simulate init failure

	// Verify IsEnabled returns false
	if manager.IsEnabled() {
		t.Error("IsEnabled() should return false after simulated init failure")
	}

	// Verify TracerProvider returns nil
	tp := manager.TracerProvider()
	if tp != nil {
		t.Error("TracerProvider() should return nil when disabled after init failure")
	}

	// Verify Shutdown succeeds (no-op for disabled manager)
	err := manager.Shutdown(context.Background())
	if err != nil {
		t.Errorf("Shutdown() should succeed for disabled manager: %v", err)
	}
}

// TestManagerGracefulDegradationResourceError tests behavior when resource creation would fail
func TestManagerGracefulDegradationResourceError(t *testing.T) {
	config := Config{
		Enabled:        true,
		Endpoint:       testEndpoint,
		Insecure:       true,
		SamplingRate:   1.0,
		ServiceName:    testServiceName,
		ServiceVersion: testServiceVersion,
	}

	manager := NewManager(config)

	// Test createResource directly with various configs
	// This ensures the method works correctly for all scenarios

	// Test 1: Normal resource creation
	res, err := manager.createResource()
	if err != nil {
		t.Errorf("createResource() failed unexpectedly: %v", err)
	}
	if res == nil {
		t.Error("createResource() returned nil resource")
	}

	// Test 2: Resource with empty service name (should still work)
	manager.config.ServiceName = ""
	res, err = manager.createResource()
	if err != nil {
		t.Errorf("createResource() with empty ServiceName failed: %v", err)
	}
	if res == nil {
		t.Error("createResource() with empty ServiceName returned nil")
	}

	// Test 3: Resource with empty service version
	manager.config.ServiceVersion = ""
	res, err = manager.createResource()
	if err != nil {
		t.Errorf("createResource() with empty ServiceVersion failed: %v", err)
	}
	if res == nil {
		t.Error("createResource() with empty ServiceVersion returned nil")
	}
}

// TestManagerCreateExporterInsecureVsSecure tests both TLS paths
func TestManagerCreateExporterInsecureVsSecure(t *testing.T) {
	tests := []struct {
		name     string
		insecure bool
	}{
		{"insecure mode", true},
		{"secure mode", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{
				Enabled:        true,
				Endpoint:       testEndpoint,
				Insecure:       tt.insecure,
				SamplingRate:   1.0,
				ServiceName:    testServiceName,
				ServiceVersion: testServiceVersion,
			}

			manager := NewManager(config)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			// createExporter should succeed in both modes
			// (gRPC connection is async, so TLS errors happen later)
			exporter, err := manager.createExporter(ctx)

			// Accept both success and failure - the important thing is no panic
			if err != nil {
				t.Logf("createExporter() returned error (may be expected): %v", err)
			} else if exporter == nil {
				t.Error("createExporter() returned nil exporter without error")
			}
		})
	}
}

// TestManagerMultipleInitializeShutdownCycles tests repeated init/shutdown
func TestManagerMultipleInitializeShutdownCycles(t *testing.T) {
	config := Config{
		Enabled:        true,
		Endpoint:       testEndpoint,
		Insecure:       true,
		SamplingRate:   1.0,
		ServiceName:    testServiceName,
		ServiceVersion: testServiceVersion,
	}

	manager := NewManager(config)

	for i := 0; i < 3; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)

		err := manager.Initialize(ctx)
		if err != nil {
			t.Errorf("Initialize() cycle %d failed: %v", i, err)
		}

		if manager.IsEnabled() {
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
			err = manager.Shutdown(shutdownCtx)
			if err != nil {
				t.Logf("Shutdown() cycle %d returned error: %v", i, err)
			}
			shutdownCancel()
		}

		cancel()
	}
}

// TestManagerSamplerDescription verifies sampler description strings
func TestManagerSamplerDescription(t *testing.T) {
	tests := []struct {
		name         string
		samplingRate float64
		wantDesc     string
	}{
		{"always on at 1.0", 1.0, "AlwaysOnSampler"},
		{"always on at 2.0", 2.0, "AlwaysOnSampler"},
		{"ratio based at 0.5", 0.5, "TraceIDRatioBased{0.5}"},
		{"ratio based at 0.1", 0.1, "TraceIDRatioBased{0.1}"},
		{"ratio based at 0.0", 0.0, "TraceIDRatioBased{0}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{
				SamplingRate: tt.samplingRate,
			}
			manager := NewManager(config)
			sampler := manager.createSampler()

			if sampler.Description() != tt.wantDesc {
				t.Errorf("createSampler() description = %v, want %v", sampler.Description(), tt.wantDesc)
			}
		})
	}
}
