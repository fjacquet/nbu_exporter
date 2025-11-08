package utils

import (
	"testing"
	"time"
)

func TestConvertTimeToNBUDate(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Time
		expected string
	}{
		{
			name:     "converts standard time to RFC3339Nano",
			input:    time.Date(2024, 11, 8, 10, 30, 45, 123456789, time.UTC),
			expected: "2024-11-08T10:30:45.123456789Z",
		},
		{
			name:     "converts time with zero nanoseconds",
			input:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			expected: "2024-01-01T00:00:00Z",
		},
		{
			name:     "converts time with timezone",
			input:    time.Date(2024, 6, 15, 14, 30, 0, 0, time.FixedZone("EST", -5*3600)),
			expected: "2024-06-15T14:30:00-05:00",
		},
		{
			name:     "converts time at midnight",
			input:    time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
			expected: "2025-12-31T00:00:00Z",
		},
		{
			name:     "converts time at end of day",
			input:    time.Date(2024, 3, 15, 23, 59, 59, 999999999, time.UTC),
			expected: "2024-03-15T23:59:59.999999999Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertTimeToNBUDate(tt.input)
			if result != tt.expected {
				t.Errorf("ConvertTimeToNBUDate() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConvertTimeToNBUDate_RoundTrip(t *testing.T) {
	// Test that we can convert to NBU format and parse it back
	original := time.Date(2024, 11, 8, 15, 30, 45, 123456789, time.UTC)
	nbuFormat := ConvertTimeToNBUDate(original)

	parsed, err := time.Parse(time.RFC3339Nano, nbuFormat)
	if err != nil {
		t.Fatalf("Failed to parse NBU date format: %v", err)
	}

	if !parsed.Equal(original) {
		t.Errorf("Round trip failed: original %v, parsed %v", original, parsed)
	}
}
