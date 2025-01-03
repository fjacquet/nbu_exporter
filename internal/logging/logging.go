package logging

import (
	"fmt"
	"io"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
)

var currentTime = time.Now()
var version = currentTime.Format("2006-01-02T15:04:05")

// Program name management
var programName = os.Args[0] + "-" + version

// LogInfo logs the provided message with the programName field.
// This function should be used to log informational messages during program execution.
func LogInfo(msg string) {
	log.WithFields(log.Fields{"job": programName}).Info(msg)
}

// LogPanic logs the provided error and exits the program with a non-zero exit code.
// This function should be used to handle critical errors that prevent the program from continuing.
func LogPanic(err error) {
	log.WithFields(log.Fields{"job": programName}).Panic(err)
}

// LogPanicMsg logs the provided message and exits the program with a non-zero exit code.
// This function should be used to handle critical errors that prevent the program from continuing.
func LogPanicMsg(msg string) {
	log.WithFields(log.Fields{"job": programName}).Panic(msg)
}

// HandleError logs the provided error and exits the program with a non-zero exit code.
// This function should be used to handle critical errors that prevent the program from continuing.
func HandleError(err error) {
	log.WithFields(log.Fields{"job": programName}).Error(err)
	os.Exit(2)
}

// LogError logs the provided error message with the programName field.
// This function should be used to log recoverable errors that do not terminate the program.
func LogError(msg string) {
	log.WithFields(log.Fields{"job": programName}).Error(msg)
}

// PrepareLogs sets up logging.
func PrepareLogs(logName string) error {
	logFile, err := os.OpenFile(logName, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %v", err)
	}
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)
	log.SetFormatter(&log.JSONFormatter{PrettyPrint: true})
	return nil
}
