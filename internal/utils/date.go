package utils

import (
	"time"
)

/**
 * ConvertTimeToNBUDate converts the given time.Time value to a string in the format "2006-01-02T15:04:05.999Z".
 * The function logs the resulting string using the InfoLogger function.
 *
 * @param t the time.Time value to convert
 * @return the converted string in the format "2006-01-02T15:04:05.999Z"
 */
func ConvertTimeToNBUDate(t time.Time) string {
	return t.Format(time.RFC3339Nano)
	// 2006-01-02T15:04:05.999Z"

}
