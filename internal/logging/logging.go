// Package logging provides centralized logging functionality using logrus.
// It configures structured logging with JSON formatting and provides
// convenience functions for different log levels.
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

// programName is used as a field in all log entries for identification
var programName = os.Args[0] + "-" + version

// LogInfo logs an informational message with the programName field.
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

// PrepareLogs initializes the logging system with the specified log file.
// It configures logging to write to both stdout and the log file with JSON formatting.
//
// If logName is empty, "stdout", or "-", logging is sent to stdout only and no
// file is opened. This is the recommended mode for containers (logs are captured
// by the runtime, e.g. `docker logs`) and avoids requiring a writable log
// directory. Otherwise logs are written to both stdout and the named file; the
// file's parent directory must already exist.
//
// Parameters:
//   - logName: Path to the log file, or "" / "stdout" / "-" for stdout only.
//
// Returns an error if the log file cannot be opened or created.
func PrepareLogs(logName string) error {
	log.SetFormatter(&log.JSONFormatter{PrettyPrint: true})

	if logName == "" || logName == "stdout" || logName == "-" {
		log.SetOutput(os.Stdout)
		return nil
	}

	logFile, err := os.OpenFile(logName, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %v", err)
	}
	log.SetOutput(io.MultiWriter(os.Stdout, logFile))
	return nil
}
