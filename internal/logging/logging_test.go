package logging

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
)

func TestLogInfo(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetFormatter(&log.JSONFormatter{})

	LogInfo("test info message")

	output := buf.String()
	if !strings.Contains(output, "test info message") {
		t.Errorf("Expected log output to contain 'test info message', got: %s", output)
	}
	if !strings.Contains(output, "\"level\":\"info\"") {
		t.Errorf("Expected log level to be 'info', got: %s", output)
	}
	if !strings.Contains(output, "\"job\":") {
		t.Errorf("Expected log to contain 'job' field, got: %s", output)
	}
}

func TestLogError(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetFormatter(&log.JSONFormatter{})

	LogError("test error message")

	output := buf.String()
	if !strings.Contains(output, "test error message") {
		t.Errorf("Expected log output to contain 'test error message', got: %s", output)
	}
	if !strings.Contains(output, "\"level\":\"error\"") {
		t.Errorf("Expected log level to be 'error', got: %s", output)
	}
}

func TestLogPanic(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetFormatter(&log.JSONFormatter{})

	defer func() {
		if r := recover(); r == nil {
			t.Error("LogPanic did not panic as expected")
		}
		output := buf.String()
		if !strings.Contains(output, "test panic error") {
			t.Errorf("Expected log output to contain 'test panic error', got: %s", output)
		}
		if !strings.Contains(output, "\"level\":\"panic\"") {
			t.Errorf("Expected log level to be 'panic', got: %s", output)
		}
	}()

	LogPanic(errors.New("test panic error"))
}

func TestLogPanicMsg(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetFormatter(&log.JSONFormatter{})

	defer func() {
		if r := recover(); r == nil {
			t.Error("LogPanicMsg did not panic as expected")
		}
		output := buf.String()
		if !strings.Contains(output, "test panic message") {
			t.Errorf("Expected log output to contain 'test panic message', got: %s", output)
		}
	}()

	LogPanicMsg("test panic message")
}

func TestPrepareLogs(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		logName     string
		expectError bool
	}{
		{
			name:        "creates new log file",
			logName:     filepath.Join(tmpDir, "test.log"),
			expectError: false,
		},
		{
			name:        "appends to existing log file",
			logName:     filepath.Join(tmpDir, "existing.log"),
			expectError: false,
		},
		{
			name:        "handles nested directory",
			logName:     filepath.Join(tmpDir, "logs", "nested.log"),
			expectError: true, // Directory doesn't exist
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Pre-create file for "existing" test
			if strings.Contains(tt.name, "existing") {
				if err := os.WriteFile(tt.logName, []byte("existing content\n"), 0644); err != nil {
					t.Fatalf("Failed to create existing log file: %v", err)
				}
			}

			err := PrepareLogs(tt.logName)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				// Verify file was created
				if _, statErr := os.Stat(tt.logName); os.IsNotExist(statErr) {
					t.Errorf("Log file was not created: %s", tt.logName)
				}

				// Test that logging works after PrepareLogs
				var buf bytes.Buffer
				log.SetOutput(&buf)
				LogInfo("test after prepare")

				output := buf.String()
				if !strings.Contains(output, "test after prepare") {
					t.Error("Logging did not work after PrepareLogs")
				}
			}
		})
	}
}

func TestPrepareLogs_InvalidPath(t *testing.T) {
	// Test with invalid path (directory that doesn't exist)
	err := PrepareLogs("/nonexistent/directory/test.log")
	if err == nil {
		t.Error("Expected error for invalid path, got nil")
	}
}

func TestPrepareLogs_JSONFormatter(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "json-test.log")

	err := PrepareLogs(logFile)
	if err != nil {
		t.Fatalf("PrepareLogs failed: %v", err)
	}

	// Log a message
	LogInfo("json format test")

	// Read the log file
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	// Verify JSON format
	output := string(content)
	if !strings.Contains(output, "{") || !strings.Contains(output, "}") {
		t.Error("Log output is not in JSON format")
	}
	if !strings.Contains(output, "\"msg\":") {
		t.Error("JSON log missing 'msg' field")
	}
	if !strings.Contains(output, "\"level\":") {
		t.Error("JSON log missing 'level' field")
	}
}
