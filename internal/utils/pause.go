package utils

import (
	"time"

	"github.com/fjacquet/nbu_exporter/internal/logging"
)

// Pause sleeps for the duration specified in the server.ScrappingInterval configuration.
// If there is an error parsing the duration, it will panic with the error message.
func Pause(interval string) {
	var duration time.Duration
	var errMsg error
	duration, errMsg = time.ParseDuration(interval)
	if errMsg != nil {
		logging.LogPanic(errMsg)
	}

	time.Sleep(duration)
}
