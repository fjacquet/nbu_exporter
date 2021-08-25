package main

import "time"

func Pause() {
	var duration time.Duration
	var errMsg error
	duration, errMsg = time.ParseDuration(Cfg.Server.ScrappingInterval)
	if errMsg != nil {
		PanicLoggerErr(errMsg)
	}

	time.Sleep(duration)
}
