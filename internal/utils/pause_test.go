package utils

import (
	"testing"
	"time"
)

func TestPause(t *testing.T) {
	tests := []struct {
		name     string
		interval string
		minTime  time.Duration
		maxTime  time.Duration
	}{
		{
			name:     "pause for 100 milliseconds",
			interval: "100ms",
			minTime:  90 * time.Millisecond,
			maxTime:  150 * time.Millisecond,
		},
		{
			name:     "pause for 1 second",
			interval: "1s",
			minTime:  900 * time.Millisecond,
			maxTime:  1200 * time.Millisecond,
		},
		{
			name:     "pause for 500 microseconds",
			interval: "500us",
			minTime:  0,
			maxTime:  10 * time.Millisecond,
		},
		{
			name:     "pause for zero duration",
			interval: "0s",
			minTime:  0,
			maxTime:  10 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			Pause(tt.interval)
			elapsed := time.Since(start)

			if elapsed < tt.minTime {
				t.Errorf("Pause(%q) took %v, expected at least %v", tt.interval, elapsed, tt.minTime)
			}
			if elapsed > tt.maxTime {
				t.Errorf("Pause(%q) took %v, expected at most %v", tt.interval, elapsed, tt.maxTime)
			}
		})
	}
}

func TestPause_InvalidDuration(t *testing.T) {
	tests := []struct {
		name     string
		interval string
	}{
		{
			name:     "invalid format",
			interval: "invalid",
		},
		{
			name:     "empty string",
			interval: "",
		},
		{
			name:     "missing unit",
			interval: "100",
		},
		{
			name:     "invalid unit",
			interval: "100x",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("Pause(%q) did not panic as expected", tt.interval)
				}
			}()
			Pause(tt.interval)
		})
	}
}
