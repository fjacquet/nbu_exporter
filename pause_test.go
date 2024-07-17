package main

import (
	"testing"
	"time"
)

func TestPause(t *testing.T) {
	// Save the original configuration
	originalCfg := Cfg
	defer func() {
		Cfg = originalCfg
	}()

	t.Run("valid duration", func(t *testing.T) {
		Cfg.Server.ScrappingInterval = "1s"
		start := time.Now()
		Pause()
		elapsed := time.Since(start)
		if elapsed < 1*time.Second || elapsed > 2*time.Second {
			t.Errorf("Pause() did not sleep for the expected duration. Expected 1s, got %v", elapsed)
		}
	})

	// t.Run("invalid duration", func(t *testing.T) {
	// 	Cfg.Server.ScrappingInterval = "invalid"
	// 	defer func() {
	// 		if r := recover(); r == nil {
	// 			t.Errorf("Pause() did not panic with an invalid duration")
	// 		}
	// 	}()
	// 	Pause()
	// })
}
