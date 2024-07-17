package main

import (
	"time"
)

/**
 * convertTimeToNBUDate converts the given time.Time value to a string in the format "2006-01-02T15:04:05.999Z".
 * The function logs the resulting string using the InfoLogger function.
 *
 * @param t the time.Time value to convert
 * @return the converted string in the format "2006-01-02T15:04:05.999Z"
 */
func convertTimeToNBUDate(t time.Time) string {
	return t.Format(time.RFC3339Nano)
	// 2006-01-02T15:04:05.999Z"
	// var returnString string = fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02d.%03dZ", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond()/1000000)
	// InfoLogger(returnString)
	// return returnString
}
