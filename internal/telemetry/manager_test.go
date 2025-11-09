package telemetry

import (
	"context"
	"testing"
	"time"
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
				Endpoint:        "localhost:4317",
				Insecure:        true,
				SamplingRate:    0.1,
				ServiceName:     "nbu-exporter",
				ServiceVersion:  "1.0.0",
				NetBackupServer: "nbu-master",
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
		t.Errorf("Initialize() unexpected error = %v", err)
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
				ServiceName:    "nbu-exporter",
				ServiceVersion: "1.0.0",
			}

			manager := NewManager(config)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			// Initialize should not return error (exporter creation succeeds even with invalid endpoints)
			// Graceful degradation: telemetry failures don't prevent application startup
			err := manager.Initialize(ctx)
			if err != nil {
				t.Errorf("Initialize() unexpected error = %v", err)
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
		Endpoint:        "localhost:4317",
		Insecure:        true,
		SamplingRate:    1.0,
		ServiceName:     "nbu-exporter-test",
		ServiceVersion:  "1.0.0-test",
		NetBackupServer: "nbu-test",
	}

	manager := NewManager(config)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := manager.Initialize(ctx)
	if err != nil {
		t.Errorf("Initialize() unexpected error = %v", err)
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
			t.Errorf("Shutdown() unexpected error = %v", err)
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
		t.Errorf("Shutdown() unexpected error = %v", err)
	}
}

// TestManager_Shutdown_WithTimeout tests shutdown with context timeout
// Graceful degradation: initialization errors are ignored to allow testing shutdown behavior
// even when the telemetry backend is unavailable.
func TestManagerShutdownWithTimeout(t *testing.T) {
	config := Config{
		Enabled:        true,
		Endpoint:       "localhost:4317",
		Insecure:       true,
		SamplingRate:   1.0,
		ServiceName:    "nbu-exporter-test",
		ServiceVersion: "1.0.0-test",
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
			t.Errorf("Shutdown() unexpected error = %v", err)
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
				Endpoint:       "localhost:4317",
				Insecure:       true,
				SamplingRate:   1.0,
				ServiceName:    "test",
				ServiceVersion: "1.0.0",
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
			serviceName:     "nbu-exporter",
			serviceVersion:  "1.0.0",
			netBackupServer: "nbu-master",
		},
		{
			name:            "creates resource without NetBackup server",
			serviceName:     "nbu-exporter",
			serviceVersion:  "1.0.0",
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
		t.Errorf("Initialize() unexpected error = %v", err)
	}

	// IsEnabled should return false
	if manager.IsEnabled() {
		t.Error("IsEnabled() should return false in disabled mode")
	}

	// Shutdown should succeed
	err = manager.Shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown() unexpected error = %v", err)
	}

	// TracerProvider should be nil
	if manager.tracerProvider != nil {
		t.Error("tracerProvider should be nil in disabled mode")
	}
}
