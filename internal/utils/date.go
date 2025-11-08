package utils

import (
	"time"
)

// ConvertTimeToNBUDate converts a time.Time value to RFC3339Nano format
// for use in NetBackup API queries.
func ConvertTimeToNBUDate(t time.Time) string {
	return t.Format(time.RFC3339Nano)
}
