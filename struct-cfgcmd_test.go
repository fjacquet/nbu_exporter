package main

import (
	"testing"
)

// TestConfigCommand_Path tests the behavior of the Path field of the ConfigCommand struct.
// It checks that the Path field is set correctly for both an empty path and a valid path.
func TestConfigCommand_Path(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "empty path",
			path:     "",
			expected: "",
		},
		{
			name:     "valid path",
			path:     "/path/to/config.yml",
			expected: "/path/to/config.yml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &ConfigCommand{
				Path: tt.path,
			}

			if cmd.Path != tt.expected {
				t.Errorf("unexpected path, got %s, want %s", cmd.Path, tt.expected)
			}
		})
	}
}
