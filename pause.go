package main

import "time"

// Pause sleeps for the duration specified in the server.ScrappingInterval configuration.
// If there is an error parsing the duration, it will panic with the error message.
func Pause() {
	var duration time.Duration
	var errMsg error
	duration, errMsg = time.ParseDuration(Cfg.Server.ScrappingInterval)
	if errMsg != nil {
		PanicLoggerErr(errMsg)
	}

	time.Sleep(duration)
}
