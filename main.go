// NBU Exporter is a Prometheus exporter for Veritas NetBackup that collects
// and exposes backup infrastructure metrics for monitoring and visualization.
//
// The exporter scrapes NetBackup API endpoints to collect:
//   - Storage unit capacity metrics (free/used bytes)
//   - Job statistics (count, bytes transferred, status)
//
// Metrics are exposed via HTTP endpoint for Prometheus scraping.
//
// Usage:
//
//	nbu_exporter --config config.yaml [--debug]
//
// Configuration is provided via YAML file specifying:
//   - Server settings (host, port, metrics URI, scraping interval)
//   - NetBackup server details (host, port, API key, API version)
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fjacquet/nbu_exporter/internal/config"
	"github.com/fjacquet/nbu_exporter/internal/exporter"
	"github.com/fjacquet/nbu_exporter/internal/logging"
	"github.com/fjacquet/nbu_exporter/internal/models"
	"github.com/fjacquet/nbu_exporter/internal/telemetry"
	"github.com/fjacquet/nbu_exporter/internal/utils"
	"github.com/fsnotify/fsnotify"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const (
	programName       = "nbu_exporter"   // Application name
	shutdownTimeout   = 10 * time.Second // Maximum time to wait for graceful shutdown
	readHeaderTimeout = 5 * time.Second  // HTTP server read header timeout
)

var (
	configFile string
	debug      bool
)

// Server encapsulates the HTTP server and its dependencies for serving Prometheus metrics.
// It manages the lifecycle of the HTTP server, Prometheus registry, NetBackup collector,
// and OpenTelemetry telemetry manager.
//
// Configuration Reload:
// The server supports dynamic configuration reload via SIGHUP signal or file watching.
// Use SafeConfig for thread-safe access to configuration during operation.
//
// Error Handling:
// Server errors (such as port binding failures) are communicated through the ErrorChan()
// channel rather than calling log.Fatal. This allows the caller to perform graceful
// shutdown even when the server encounters errors.
//
// Usage:
//
//	safeCfg := models.NewSafeConfig(cfg)
//	server := NewServer(safeCfg, configPath)
//	if err := server.Start(); err != nil {
//	    return err
//	}
//
//	select {
//	case <-shutdownSignal:
//	    // Normal shutdown
//	case err := <-server.ErrorChan():
//	    log.Errorf("Server error: %v", err)
//	}
//
//	server.Shutdown()
type Server struct {
	cfg              *models.SafeConfig     // Thread-safe config wrapper
	configPath       string                 // Path to config file (for reload)
	httpSrv          *http.Server           // HTTP server instance
	registry         *prometheus.Registry   // Prometheus metrics registry
	telemetryManager *telemetry.Manager     // OpenTelemetry telemetry manager (nil if disabled)
	collector        *exporter.NbuCollector // NetBackup collector (for cleanup)
	configWatcher    *fsnotify.Watcher      // File watcher for config reload (for cleanup)
	// serverErrChan receives HTTP server errors. It is buffered (capacity 1)
	// to ensure the goroutine can send an error even if the main select
	// hasn't started listening yet (race between Start() return and select).
	serverErrChan chan error
}

// NewServer creates a new server instance with the provided SafeConfig.
// It initializes a new Prometheus registry for metric collection and creates
// a telemetry manager if OpenTelemetry is enabled in the configuration.
//
// The server uses SafeConfig for thread-safe configuration access, enabling
// dynamic configuration reload without restart.
//
// Example:
//
//	cfg := &models.Config{...}
//	safeCfg := models.NewSafeConfig(cfg)
//	server := NewServer(safeCfg, "/path/to/config.yaml")
//	server.Start()
func NewServer(safeCfg *models.SafeConfig, configPath string) *Server {
	cfg := safeCfg.Get()
	var telemetryMgr *telemetry.Manager

	// Create telemetry manager if OpenTelemetry is enabled
	if cfg.IsOTelEnabled() {
		telemetryMgr = telemetry.NewManager(telemetry.Config{
			Enabled:         cfg.OpenTelemetry.Enabled,
			Endpoint:        cfg.OpenTelemetry.Endpoint,
			Insecure:        cfg.OpenTelemetry.Insecure,
			SamplingRate:    cfg.OpenTelemetry.SamplingRate,
			ServiceName:     "nbu-exporter",
			ServiceVersion:  "2.0.0", // Version with OpenTelemetry support
			NetBackupServer: cfg.NbuServer.Host,
		})
	}

	return &Server{
		cfg:              safeCfg,
		configPath:       configPath,
		registry:         prometheus.NewRegistry(),
		telemetryManager: telemetryMgr,
		serverErrChan:    make(chan error, 1), // Buffered to prevent goroutine leak
	}
}

// Start initializes and starts the HTTP server with Prometheus metrics endpoint.
// It initializes OpenTelemetry if enabled, registers the NetBackup collector,
// configures HTTP handlers, and starts the server in a goroutine.
//
// The server exposes:
//   - Metrics endpoint at the configured URI (default: /metrics)
//   - Health check endpoint at /health
//
// Returns an error if collector creation or registration fails. The HTTP server runs
// asynchronously and logs fatal errors if startup fails.
func (s *Server) Start() error {
	cfg := s.cfg.Get()

	// Initialize OpenTelemetry if enabled
	var tracerProvider trace.TracerProvider
	if s.telemetryManager != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := s.telemetryManager.Initialize(ctx); err != nil {
			// Log warning but continue - telemetry manager handles graceful degradation
			log.Warnf("Failed to initialize OpenTelemetry: %v. Continuing without tracing.", err)
		}

		// Configure W3C Trace Context propagation if telemetry is enabled
		if s.telemetryManager.IsEnabled() {
			// Get TracerProvider for injection
			tracerProvider = s.telemetryManager.TracerProvider()

			otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
				propagation.TraceContext{},
				propagation.Baggage{},
			))
			log.Info("OpenTelemetry trace context propagation configured")
		}
	}

	// Create NetBackup collector with injected TracerProvider
	var collectorOpts []exporter.CollectorOption
	if tracerProvider != nil {
		collectorOpts = append(collectorOpts, exporter.WithCollectorTracerProvider(tracerProvider))
	}

	collector, err := exporter.NewNbuCollector(*cfg, collectorOpts...)
	if err != nil {
		return fmt.Errorf("failed to create collector: %w", err)
	}

	// Store collector reference for shutdown cleanup
	s.collector = collector

	// Register collector with Prometheus
	if err := s.registry.Register(collector); err != nil {
		return fmt.Errorf("failed to register collector: %w", err)
	}

	// Setup HTTP handlers
	mux := http.NewServeMux()

	// Wrap Prometheus handler with trace context extraction if OpenTelemetry is enabled
	prometheusHandler := promhttp.HandlerFor(s.registry, promhttp.HandlerOpts{})
	if s.telemetryManager != nil && s.telemetryManager.IsEnabled() {
		prometheusHandler = s.extractTraceContextMiddleware(prometheusHandler)
	}

	mux.Handle(cfg.Server.URI, prometheusHandler)
	mux.HandleFunc("/health", s.healthHandler)

	// Create HTTP server
	s.httpSrv = &http.Server{
		Addr:              cfg.GetServerAddress(),
		Handler:           mux,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	// Start server in goroutine
	go func() {
		log.Infof("Starting %s on %s%s", programName, cfg.GetServerAddress(), cfg.Server.URI)
		if err := s.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// Send error through channel instead of log.Fatalf
			s.serverErrChan <- fmt.Errorf("HTTP server error: %w", err)
		}
	}()

	return nil
}

// ErrorChan returns the channel for receiving server errors.
// The main function should select on this channel to handle errors gracefully.
func (s *Server) ErrorChan() <-chan error {
	return s.serverErrChan
}

// ReloadConfig reloads configuration from the config file.
// It validates the new configuration before applying and flushes the storage
// cache if the NBU server address changed.
//
// This method is called by:
//   - SIGHUP signal handler (manual reload trigger)
//   - File watcher (automatic reload on config change)
//
// Thread-safety:
// Uses SafeConfig's fail-fast validation pattern - invalid configurations
// are rejected without affecting the running exporter.
//
// Returns an error if the configuration file cannot be read or validation fails.
func (s *Server) ReloadConfig(configPath string) error {
	serverChanged, err := s.cfg.ReloadConfig(configPath)
	if err != nil {
		return err
	}

	// Flush cache if NBU server address changed
	if serverChanged && s.collector != nil {
		if cache := s.collector.GetStorageCache(); cache != nil {
			cache.Flush()
			log.Info("Storage cache flushed due to server address change")
		}
	}

	return nil
}

// SetConfigWatcher stores the config file watcher reference for cleanup.
// Called by main() after setting up the file watcher.
func (s *Server) SetConfigWatcher(w *fsnotify.Watcher) {
	s.configWatcher = w
}

// Shutdown gracefully shuts down the HTTP server and OpenTelemetry with a timeout.
// It waits for active connections to complete and flushes pending telemetry data
// before shutting down.
//
// The shutdown process:
//  1. Shuts down OpenTelemetry TracerProvider (flushes pending spans)
//  2. Stops accepting new HTTP connections
//  3. Waits for active requests to complete (up to shutdownTimeout)
//  4. Forces shutdown if timeout is exceeded
//
// Shutdown gracefully shuts down the server components in the correct order.
//
// Shutdown Order:
//  1. Close config watcher (stop watching for config changes)
//  2. Stop HTTP server (no new scrapes accepted)
//  3. Shutdown OpenTelemetry (flush pending spans)
//  4. Close collector (drains API connections)
//
// Note: Telemetry is shutdown BEFORE client to ensure traces from
// in-flight requests are flushed before connections close.
//
// Returns an error if shutdown fails or times out.
func (s *Server) Shutdown() error {
	var errs []error

	// Step 0: Close config watcher (stop watching for changes)
	if s.configWatcher != nil {
		log.Info("Closing config file watcher...")
		if err := s.configWatcher.Close(); err != nil {
			log.Warnf("Config watcher close warning: %v", err)
			// Non-fatal, continue shutdown
		}
	}

	// Step 1: Shutdown HTTP server first (stops accepting new scrapes)
	if s.httpSrv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		log.Info("Shutting down HTTP server...")
		if err := s.httpSrv.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("HTTP server shutdown: %w", err))
		}
	}

	// Step 2: Shutdown OpenTelemetry (flush pending spans)
	if s.telemetryManager != nil {
		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		log.Info("Shutting down telemetry...")
		if err := s.telemetryManager.Shutdown(ctx); err != nil {
			log.Warnf("Telemetry shutdown warning: %v", err)
			// Don't add to errs - telemetry shutdown warnings are non-fatal
		}
	}

	// Step 3: Close collector (drains API connections)
	if s.collector != nil {
		log.Info("Closing collector connections...")
		if err := s.collector.Close(); err != nil {
			errs = append(errs, fmt.Errorf("collector close: %w", err))
		}
	}

	// Close error channel to signal no more errors will be sent
	close(s.serverErrChan)

	if len(errs) > 0 {
		log.Errorf("Shutdown completed with %d errors", len(errs))
		// Return first error for simplicity
		return errs[0]
	}

	log.Info("Server stopped gracefully")
	return nil
}

// extractTraceContextMiddleware wraps an HTTP handler to extract trace context from incoming requests.
// This enables distributed tracing when the exporter is part of a larger observability pipeline.
// The extracted context is propagated to the Prometheus collector's Collect method.
//
// The middleware:
//   - Extracts W3C Trace Context headers from the incoming request
//   - Creates a new context with the extracted trace information
//   - Passes the context to the wrapped handler
//
// If no trace context is present in the request, the handler operates normally without tracing.
func (s *Server) extractTraceContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract trace context from incoming request headers using the global propagator
		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))

		// Create a new request with the extracted context
		r = r.WithContext(ctx)

		// Call the next handler with the updated request
		next.ServeHTTP(w, r)
	})
}

// healthHandler provides health check with NetBackup connectivity verification.
// Returns 200 OK if NBU API is reachable, 503 Service Unavailable otherwise.
// Used by load balancers and orchestrators (Kubernetes probes).
//
// Behavior:
//   - If collector not initialized (startup phase): returns 200 "OK (starting)"
//   - If NBU API is reachable: returns 200 "OK"
//   - If NBU API is unreachable: returns 503 "UNHEALTHY: NetBackup API unreachable"
//
// The connectivity test uses a lightweight API call with 5-second timeout.
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	// Fast path: if no collector, just return OK (startup phase)
	if s.collector == nil {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, "OK (starting)\n")
		return
	}

	// Test NetBackup connectivity with timeout from request context
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := s.collector.TestConnectivity(ctx); err != nil {
		log.Warnf("Health check failed: %v", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = fmt.Fprintf(w, "UNHEALTHY: NetBackup API unreachable\n")
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintf(w, "OK\n")
}

// validateConfig checks if the configuration file exists, loads it, and validates its contents.
//
// Parameters:
//   - configPath: Path to the YAML configuration file
//
// Returns:
//   - Pointer to validated Config struct
//   - Error if file doesn't exist, cannot be parsed, or validation fails
func validateConfig(configPath string) (*models.Config, error) {
	if !utils.FileExists(configPath) {
		return nil, fmt.Errorf("config file not found: %s", configPath)
	}

	var cfg models.Config
	if err := utils.ReadFile(&cfg, configPath); err != nil {
		return nil, err
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// setupLogging initializes the logging system with the configured log file.
// If debug mode is enabled, sets the log level to DEBUG for verbose output.
//
// Parameters:
//   - cfg: Application configuration containing log file path
//   - debugMode: If true, enables DEBUG level logging
//
// Returns an error if log file initialization fails.
func setupLogging(cfg models.Config, debugMode bool) error {
	if err := logging.PrepareLogs(cfg.Server.LogName); err != nil {
		return fmt.Errorf("failed to initialize logging: %w", err)
	}

	if debugMode {
		log.SetLevel(log.DebugLevel)
		log.Debug("Debug mode enabled")
	}

	return nil
}

// waitForShutdown blocks until either a shutdown signal is received
// or a server error occurs through the error channel.
//
// Signals handled:
//   - SIGINT (Ctrl+C)
//   - SIGTERM (kill command)
//
// Returns an error if the server encountered a fatal error, nil for normal signal shutdown.
func waitForShutdown(serverErr <-chan error) error {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-stop:
		log.Infof("Received signal %v, initiating graceful shutdown...", sig)
		return nil
	case err := <-serverErr:
		if err != nil {
			return err
		}
		return nil
	}
}

func main() {
	rootCmd := &cobra.Command{
		Use:   programName,
		Short: "Prometheus exporter for Veritas NetBackup",
		Long:  "NBU Exporter collects metrics from NetBackup API and exposes them in Prometheus format",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Validate and load configuration
			cfg, err := validateConfig(configFile)
			if err != nil {
				return err
			}

			// Wrap in SafeConfig for thread-safe access
			safeCfg := models.NewSafeConfig(cfg)

			// Setup logging
			if err := setupLogging(*safeCfg.Get(), debug); err != nil {
				return err
			}

			log.Infof("Starting %s...", programName)
			log.Infof("NBU server: %s", safeCfg.Get().GetNBUBaseURL())
			log.Infof("Scraping interval: %s", safeCfg.Get().Server.ScrapingInterval)
			if debug {
				log.Infof("API Key: %s", safeCfg.Get().MaskAPIKey())
			}

			// Create and start server
			server := NewServer(safeCfg, configFile)
			if err := server.Start(); err != nil {
				return err
			}

			// Setup config reload handlers
			config.SetupSIGHUPHandler(configFile, server.ReloadConfig)

			watcher, err := config.WatchConfigFile(configFile, server.ReloadConfig)
			if err != nil {
				log.Warnf("File watcher setup failed: %v. SIGHUP reload still available.", err)
			} else {
				server.SetConfigWatcher(watcher)
			}

			// Wait for shutdown signal or server error
			if err := waitForShutdown(server.ErrorChan()); err != nil {
				log.Errorf("Server error: %v", err)
				// Continue to graceful shutdown
			}

			// Graceful shutdown
			return server.Shutdown()
		},
	}

	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "Path to configuration file (required)")
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "Enable debug mode")
	_ = rootCmd.MarkPersistentFlagRequired("config")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
