package main

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
)

func InfoLogger(msg string) {
	log.WithFields(log.Fields{"job": programName}).Info(msg)
}

func PanicLoggerErr(errMgr error) {
	log.WithFields(log.Fields{"job": programName}).Panicln(errMgr)
}

func PanicLoggerStr(msg string) {
	log.WithFields(log.Fields{"job": programName}).Panicln(msg)
}

func ProcessError(err error) {
	fmt.Println(err)
	os.Exit(2)
}
