package main

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
)

// InfoLogger logs the provided message with the programName field.
// This function should be used to log informational messages during program execution.
func InfoLogger(msg string) {
	log.WithFields(log.Fields{"job": programName}).Info(msg)
}

// PanicLoggerErr logs the provided error and exits the program with a non-zero exit code.
// This function should be used to handle critical errors that prevent the program from continuing.
func PanicLoggerErr(errMgr error) {
	log.WithFields(log.Fields{"job": programName}).Panicln(errMgr)
}

// PanicLoggerStr logs the provided message and exits the program with a non-zero exit code.
// This function should be used to handle critical errors that prevent the program from continuing.
func PanicLoggerStr(msg string) {
	log.WithFields(log.Fields{"job": programName}).Panicln(msg)
}

// ProcessError logs the provided error and exits the program with a non-zero exit code.
// This function should be used to handle critical errors that prevent the program from continuing.
func ProcessError(err error) {
	fmt.Println(err)
	os.Exit(2)
}
