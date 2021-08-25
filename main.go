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

func prepareLogs() {
	logFile, err := os.OpenFile(Cfg.Server.LogName, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		panic(err)
	}
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)
	// log.SetFormatter(&log.JSONFormatter{PrettyPrint: true})
}

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
