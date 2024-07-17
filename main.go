package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/alecthomas/kong"
	"github.com/go-resty/resty/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

// ConfigFile is the path to the configuration file.
// Cfg is the configuration struct.
// Client is the resty client.
// programName is the name of the program, including the version.
// cli is the command-line interface struct, with a Debug flag and a Config command.
// nbuRoot is the root directory for the NBU (Netbackup) configuration.
var (
	ConfigFile  string
	Cfg         Config
	Client      *resty.Client
	programName string
	cli         struct {
		Debug  bool          `help:"Enable debug mode."`
		Config ConfigCommand `cmd help:"Path to configuration file."`
	}
	nbuRoot string
)

// checkParams validates the command-line arguments and configuration file.
// If the number of arguments is less than 2, it prints an error message and exits the program.
// It then parses the command-line arguments using the Kong library and runs the context.
// If there is an error during the parsing or running, it prints an error message and exits the program.
// Finally, it checks if the configuration file exists, and if not, it prints an error message and exits the program.
func checkParams() {
	if len(os.Args) < 2 {
		fmt.Println("Invalid call, pleases try " + os.Args[0] + " --help ")
		os.Exit(1)
	}

	// command line management
	ctx := kong.Parse(&cli)
	err := ctx.Run(&context{Debug: cli.Debug})
	ctx.FatalIfErrorf(err)

	if !fileExists(ConfigFile) {
		fmt.Println("can not find file " + ConfigFile)
		os.Exit(1)
	}

}

// prepareLogs sets up logging by creating a log file, configuring a multi-writer to write to both the log file and stdout, and setting the log output to the multi-writer.
// The log file is created with the name specified in the Cfg.Server.LogName configuration, and has read-write permissions set to 0644.
// If there is an error opening the log file, the function will panic.
func prepareLogs() {
	logFile, err := os.OpenFile(Cfg.Server.LogName, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		panic(err)
	}
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)
	// log.SetFormatter(&log.JSONFormatter{PrettyPrint: true})
}

// main is the entry point of the application. It sets up the program name, checks the command-line parameters,
// reads the configuration file, sets up logging, registers the NBU collector with Prometheus, and starts the HTTP server
// to expose the Prometheus metrics.
//
// The program name is constructed using the current time in the format "2006-01-02T15:04:05".
//
// The checkParams function validates the command-line arguments and configuration file. If the number of arguments is less
// than 2, it prints an error message and exits the program. It then parses the command-line arguments using the Kong library
// and runs the context. If there is an error during the parsing or running, it prints an error message and exits the program.
// Finally, it checks if the configuration file exists, and if not, it prints an error message and exits the program.
//
// The prepareLogs function sets up logging by creating a log file, configuring a multi-writer to write to both the log file
// and stdout, and setting the log output to the multi-writer. The log file is created with the name specified in the
// Cfg.Server.LogName configuration, and has read-write permissions set to 0644. If there is an error opening the log file,
// the function will panic.
//
// The program then registers the NBU collector with Prometheus and starts the HTTP server to expose the Prometheus metrics.
// The server listens on the host and port specified in the Cfg.Server.Host and Cfg.Server.Port configuration, and serves the
// metrics at the URI specified in the Cfg.Server.URI configuration.
func main() {

	currentTime := time.Now()
	var version string = currentTime.Format("2006-01-02T15:04:05")

	// program name management
	programName = os.Args[0] + "-" + version

	checkParams()

	ReadFile(&Cfg, ConfigFile)

	prepareLogs()

	// log creation

	InfoLogger("logName is: " + Cfg.Server.LogName)
	InfoLogger("Starting " + programName + "...")
	InfoLogger("ScrappingInterval:" + Cfg.Server.ScrappingInterval)

	nbuRoot = Cfg.NbuServer.Scheme + "://" + Cfg.NbuServer.Host + ":" + Cfg.NbuServer.Port + Cfg.NbuServer.URI
	if cli.Debug {
		InfoLogger("nbu server is on " + nbuRoot)
	}

	// register worker
	nbu := newNbuCollector()
	prometheus.MustRegister(nbu)

	// http server startup
	http.Handle(Cfg.Server.URI, promhttp.Handler())
	if cli.Debug {
		InfoLogger("starting exporter on " + Cfg.Server.Host + ":" + Cfg.Server.Port + Cfg.Server.URI)
	}
	http.ListenAndServe(Cfg.Server.Host+":"+Cfg.Server.Port, nil)

}
