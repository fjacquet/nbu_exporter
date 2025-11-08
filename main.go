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

	"github.com/fjacquet/nbu_exporter/internal/exporter"
	"github.com/fjacquet/nbu_exporter/internal/logging"
	"github.com/fjacquet/nbu_exporter/internal/models"
	"github.com/fjacquet/nbu_exporter/internal/utils"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
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
// It manages the lifecycle of the HTTP server, Prometheus registry, and NetBackup collector.
type Server struct {
	cfg      models.Config        // Application configuration
	httpSrv  *http.Server         // HTTP server instance
	registry *prometheus.Registry // Prometheus metrics registry
}

// NewServer creates a new server instance with the provided configuration.
// It initializes a new Prometheus registry for metric collection.
//
// Example:
//
//	cfg := models.Config{...}
//	server := NewServer(cfg)
//	server.Start()
func NewServer(cfg models.Config) *Server {
	return &Server{
		cfg:      cfg,
		registry: prometheus.NewRegistry(),
	}
}

// Start initializes and starts the HTTP server with Prometheus metrics endpoint.
// It registers the NetBackup collector, configures HTTP handlers, and starts
// the server in a goroutine.
//
// The server exposes:
//   - Metrics endpoint at the configured URI (default: /metrics)
//   - Health check endpoint at /health
//
// Returns an error if collector creation or registration fails. The HTTP server runs
// asynchronously and logs fatal errors if startup fails.
func (s *Server) Start() error {
	// Create NetBackup collector with version detection
	collector, err := exporter.NewNbuCollector(s.cfg)
	if err != nil {
		return fmt.Errorf("failed to create collector: %w", err)
	}

	// Register collector with Prometheus
	if err := s.registry.Register(collector); err != nil {
		return fmt.Errorf("failed to register collector: %w", err)
	}

	// Setup HTTP handlers
	mux := http.NewServeMux()
	mux.Handle(s.cfg.Server.URI, promhttp.HandlerFor(s.registry, promhttp.HandlerOpts{}))
	mux.HandleFunc("/health", s.healthHandler)

	// Create HTTP server
	s.httpSrv = &http.Server{
		Addr:              s.cfg.GetServerAddress(),
		Handler:           mux,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	// Start server in goroutine
	go func() {
		log.Infof("Starting %s on %s%s", programName, s.cfg.GetServerAddress(), s.cfg.Server.URI)
		if err := s.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	return nil
}

// Shutdown gracefully shuts down the HTTP server with a timeout.
// It waits for active connections to complete before shutting down.
//
// The shutdown process:
//  1. Stops accepting new connections
//  2. Waits for active requests to complete (up to shutdownTimeout)
//  3. Forces shutdown if timeout is exceeded
//
// Returns an error if shutdown fails or times out.
func (s *Server) Shutdown() error {
	if s.httpSrv == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	log.Info("Shutting down server...")
	if err := s.httpSrv.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	log.Info("Server stopped gracefully")
	return nil
}

// healthHandler provides a simple health check endpoint that returns HTTP 200 OK.
// This endpoint can be used by load balancers and monitoring systems to verify
// the application is running.
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
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

// waitForShutdownSignal blocks until a shutdown signal (SIGINT or SIGTERM) is received.
// This enables graceful shutdown when the application is terminated.
//
// Signals handled:
//   - SIGINT (Ctrl+C)
//   - SIGTERM (kill command)
func waitForShutdownSignal() {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
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

			// Setup logging
			if err := setupLogging(*cfg, debug); err != nil {
				return err
			}

			log.Infof("Starting %s...", programName)
			log.Infof("NBU server: %s", cfg.GetNBUBaseURL())
			log.Infof("Scraping interval: %s", cfg.Server.ScrapingInterval)
			if debug {
				log.Infof("API Key: %s", cfg.MaskAPIKey())
			}

			// Create and start server
			server := NewServer(*cfg)
			if err := server.Start(); err != nil {
				return err
			}

			// Wait for shutdown signal
			waitForShutdownSignal()

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
