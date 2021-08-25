package main

import (
	"fmt"
	"time"
)

func convertTimeToNBUDate(t time.Time) string {
	// 2006-01-02T15:04:05.999Z"
	var returnString string = fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02d.%03dZ", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond()/1000000)
	InfoLogger(returnString)
	return returnString
}
