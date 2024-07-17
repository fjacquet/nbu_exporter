package main

import (
	"testing"
	"time"
)

// TestConvertTimeToNBUDate tests the convertTimeToNBUDate function, which converts a time.Time value to a string in the format expected by the NBU (National Bank of Ukraine).
// The test cases cover valid time values, zero time values, and nanosecond precision.

func TestConvertTimeToNBUDate(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Time
		expected string
	}{
		{
			name:     "Valid time",
			input:    time.Date(2023, 4, 15, 12, 34, 56, 789000000, time.UTC),
			expected: "2023-04-15T12:34:56.789Z",
		},
		{
			name:     "Zero time",
			input:    time.Time{},
			expected: "0001-01-01T00:00:00Z",
		},
		{
			name:     "Nanosecond precision",
			input:    time.Date(2023, 4, 15, 12, 34, 56, 789123456, time.UTC),
			expected: "2023-04-15T12:34:56.789123456Z",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := convertTimeToNBUDate(tc.input)
			if actual != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, actual)
			}
		})
	}
}

// func TestFormatDate(t *testing.T) {
// 	tests := []struct {
// 		name     string
// 		input    time.Time
// 		expected string
// 	}{
// 		{
// 			name:     "Valid date",
// 			input:    time.Date(2023, time.May, 15, 12, 30, 0, 0, time.UTC),
// 			expected: "2023-05-15 12:30:00 +0000 UTC",
// 		},
// 		{
// 			name:     "Zero date",
// 			input:    time.Time{},
// 			expected: "0001-01-01 00:00:00 +0000 UTC",
// 		},
// 	}

// 	for _, tc := range tests {
// 		t.Run(tc.name, func(t *testing.T) {
// 			actual := FormatDate(tc.input)
// 			if actual != tc.expected {
// 				t.Errorf("Expected %s, got %s", tc.expected, actual)
// 			}
// 		})
// 	}
// }
