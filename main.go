package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/fjacquet/nbu_exporter/internal/exporter"
	"github.com/fjacquet/nbu_exporter/internal/logging"
	"github.com/fjacquet/nbu_exporter/internal/models"
	"github.com/fjacquet/nbu_exporter/internal/utils"
	"github.com/go-resty/resty/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	ConfigFile  string
	Cfg         models.Config
	Client      *resty.Client
	programName string
	Debug       bool
	nbuRoot     string
)

// checkParams validates the command-line arguments and configuration file.
func checkParams() error {
	if !utils.FileExists(ConfigFile) {
		return fmt.Errorf("cannot find file %s", ConfigFile)
	}
	return nil
}

// startHTTPServer starts the HTTP server and handles graceful shutdown.
func startHTTPServer() {
	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", Cfg.Server.Host, Cfg.Server.Port),
		Handler: http.DefaultServeMux,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	log.Infof("Starting exporter on %s:%s%s", Cfg.Server.Host, Cfg.Server.Port, Cfg.Server.URI)

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Info("Shutting down server...")
	if err := server.Shutdown(nil); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Info("Server exiting")
}

func main() {
	var rootCmd = &cobra.Command{
		Use:   "nbu_exporter",
		Short: "NBU Exporter for Prometheus",
		Run: func(cmd *cobra.Command, args []string) {
			if err := checkParams(); err != nil {
				log.Fatal(err)
			}

			utils.ReadFile(&Cfg, ConfigFile)
			nbuRoot = fmt.Sprintf("%s://%s:%s%s", Cfg.NbuServer.Scheme, Cfg.NbuServer.Host, Cfg.NbuServer.Port, Cfg.NbuServer.URI)

			if err := logging.PrepareLogs(Cfg.Server.LogName); err != nil {
				log.Fatal(err)
			}

			log.Infof("Log name is: %s", Cfg.Server.LogName)
			log.Infof("Starting %s...", programName)
			log.Infof("ScrappingInterval: %s", Cfg.Server.ScrappingInterval)

			if Debug {
				log.Infof("NBU server is on %s", nbuRoot)
			}

			// Register worker
			nbu := exporter.NewNbuCollector(Cfg)
			prometheus.MustRegister(nbu)

			// HTTP server startup
			http.Handle(Cfg.Server.URI, promhttp.Handler())
			startHTTPServer()
		},
	}

	rootCmd.PersistentFlags().StringVarP(&ConfigFile, "config", "c", "", "Path to configuration file")
	rootCmd.PersistentFlags().BoolVarP(&Debug, "debug", "d", false, "Enable debug mode")
	rootCmd.MarkPersistentFlagRequired("config")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
