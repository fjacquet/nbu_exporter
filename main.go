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
	programName       = "nbu_exporter"
	shutdownTimeout   = 10 * time.Second
	readHeaderTimeout = 5 * time.Second
)

var (
	configFile string
	debug      bool
)

// Server encapsulates the HTTP server and its dependencies.
type Server struct {
	cfg      models.Config
	httpSrv  *http.Server
	registry *prometheus.Registry
}

// NewServer creates a new server instance with the provided configuration.
func NewServer(cfg models.Config) *Server {
	return &Server{
		cfg:      cfg,
		registry: prometheus.NewRegistry(),
	}
}

// Start initializes and starts the HTTP server.
func (s *Server) Start() error {
	// Register NetBackup collector
	collector := exporter.NewNbuCollector(s.cfg)
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

// Shutdown gracefully shuts down the HTTP server.
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

// healthHandler provides a simple health check endpoint.
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "OK\n")
}

// validateConfig checks if the configuration file exists and is valid.
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

// setupLogging initializes the logging system.
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

// waitForShutdownSignal blocks until a shutdown signal is received.
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
	rootCmd.MarkPersistentFlagRequired("config")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
