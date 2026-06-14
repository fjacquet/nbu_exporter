package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fjacquet/nbu_exporter/internal/models"
)

func TestFileExists(t *testing.T) {
	// Create a temporary file for testing
	tmpDir := t.TempDir()
	existingFile := filepath.Join(tmpDir, "existing.txt")
	if err := os.WriteFile(existingFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		{
			name:     "existing file returns true",
			filename: existingFile,
			expected: true,
		},
		{
			name:     "non-existing file returns false",
			filename: filepath.Join(tmpDir, "nonexistent.txt"),
			expected: false,
		},
		{
			name:     "directory returns true",
			filename: tmpDir,
			expected: true,
		},
		{
			name:     "empty path returns false",
			filename: "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FileExists(tt.filename)
			if result != tt.expected {
				t.Errorf("FileExists(%q) = %v, want %v", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestReadFile(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		fileContent string
		fileName    string
		expectError bool
		validate    func(*testing.T, *models.Config)
	}{
		{
			name: "valid config file",
			fileContent: `
server:
  host: "localhost"
  port: "9440"
  uri: "/metrics"
  scrapingInterval: "5m"
  logName: "test.log"
nbuserver:
  host: "nbu-master"
  port: "1556"
  scheme: "https"
  uri: "/netbackup"
  apiKey: "test-api-key"
  apiVersion: "13.0"
`,
			fileName:    "valid-config.yaml",
			expectError: false,
			validate: func(t *testing.T, cfg *models.Config) {
				if cfg.Server.Host != "localhost" {
					t.Errorf("Expected server host 'localhost', got '%s'", cfg.Server.Host)
				}
				if cfg.NbuServer.APIVersion != "13.0" {
					t.Errorf("Expected API version '13.0', got '%s'", cfg.NbuServer.APIVersion)
				}
			},
		},
		{
			name: "config with optional fields",
			fileContent: `
server:
  host: "0.0.0.0"
  port: "9090"
  uri: "/prometheus"
  scrapingInterval: "10m"
  logName: "nbu.log"
nbuserver:
  host: "backup.example.com"
  port: "1556"
  scheme: "https"
  uri: "/netbackup"
  domain: "example.com"
  domainType: "NT"
  apiKey: "secret-key"
  apiVersion: "12.0"
  contentType: "application/json"
  insecureSkipVerify: true
`,
			fileName:    "full-config.yaml",
			expectError: false,
			validate: func(t *testing.T, cfg *models.Config) {
				if cfg.NbuServer.Domain != "example.com" {
					t.Errorf("Expected domain 'example.com', got '%s'", cfg.NbuServer.Domain)
				}
				if !cfg.NbuServer.InsecureSkipVerify {
					t.Error("Expected insecureSkipVerify to be true")
				}
			},
		},
		{
			name:        "invalid YAML syntax",
			fileContent: "invalid: yaml: content: [unclosed",
			fileName:    "invalid.yaml",
			expectError: true,
		},
		{
			name:        "empty file",
			fileContent: "",
			fileName:    "empty.yaml",
			expectError: true, // Empty YAML file causes EOF error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file
			filePath := filepath.Join(tmpDir, tt.fileName)
			if err := os.WriteFile(filePath, []byte(tt.fileContent), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			// Test ReadFile
			var cfg models.Config
			err := ReadFile(&cfg, filePath)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if tt.validate != nil {
					tt.validate(t, &cfg)
				}
			}
		})
	}
}

func TestReadFileNonExistentFile(t *testing.T) {
	var cfg models.Config
	err := ReadFile(&cfg, "/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestReadFileInvalidPermissions(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "noperm.yaml")

	// Create file with no read permissions
	if err := os.WriteFile(filePath, []byte("test: data"), 0000); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer func() { _ = os.Chmod(filePath, 0644) }() // Cleanup

	var cfg models.Config
	err := ReadFile(&cfg, filePath)
	if err == nil {
		t.Error("Expected error for file with no read permissions, got nil")
	}
}

func TestResolveSecrets(t *testing.T) {
	t.Run("expands host and apiKey from environment", func(t *testing.T) {
		t.Setenv("TEST_NBU_HOST", "resolved-host.example.com")
		t.Setenv("TEST_NBU_KEY", "resolved-api-key")

		cfg := &models.Config{}
		cfg.NbuServer.Host = "${TEST_NBU_HOST}"
		cfg.NbuServer.APIKey = "${TEST_NBU_KEY}"

		if err := ResolveSecrets(cfg); err != nil {
			t.Fatalf("ResolveSecrets() unexpected error: %v", err)
		}
		if cfg.NbuServer.Host != "resolved-host.example.com" {
			t.Errorf("Host = %q, want %q", cfg.NbuServer.Host, "resolved-host.example.com")
		}
		if cfg.NbuServer.APIKey != "resolved-api-key" {
			t.Errorf("APIKey = %q, want %q", cfg.NbuServer.APIKey, "resolved-api-key")
		}
	})

	t.Run("literal values pass through unchanged", func(t *testing.T) {
		cfg := &models.Config{}
		cfg.NbuServer.Host = "literal-host"
		cfg.NbuServer.APIKey = "literal-key"

		if err := ResolveSecrets(cfg); err != nil {
			t.Fatalf("ResolveSecrets() unexpected error: %v", err)
		}
		if cfg.NbuServer.Host != "literal-host" {
			t.Errorf("Host = %q, want %q", cfg.NbuServer.Host, "literal-host")
		}
		if cfg.NbuServer.APIKey != "literal-key" {
			t.Errorf("APIKey = %q, want %q", cfg.NbuServer.APIKey, "literal-key")
		}
	})

	t.Run("unset host variable returns error with field context", func(t *testing.T) {
		cfg := &models.Config{}
		cfg.NbuServer.Host = "${UNSET_HOST_VAR_XYZ}"
		cfg.NbuServer.APIKey = "literal-key"

		err := ResolveSecrets(cfg)
		if err == nil {
			t.Fatal("ResolveSecrets() expected error, got nil")
		}
		if !strings.Contains(err.Error(), "nbuserver.host") {
			t.Errorf("error %q should mention field nbuserver.host", err.Error())
		}
		if !strings.Contains(err.Error(), "UNSET_HOST_VAR_XYZ") {
			t.Errorf("error %q should name the missing var", err.Error())
		}
	})

	t.Run("unset apiKey variable returns error with field context", func(t *testing.T) {
		t.Setenv("TEST_NBU_HOST2", "some-host")
		cfg := &models.Config{}
		cfg.NbuServer.Host = "${TEST_NBU_HOST2}"
		cfg.NbuServer.APIKey = "${UNSET_KEY_VAR_ABC}"

		err := ResolveSecrets(cfg)
		if err == nil {
			t.Fatal("ResolveSecrets() expected error, got nil")
		}
		if !strings.Contains(err.Error(), "nbuserver.apiKey") {
			t.Errorf("error %q should mention field nbuserver.apiKey", err.Error())
		}
		if !strings.Contains(err.Error(), "UNSET_KEY_VAR_ABC") {
			t.Errorf("error %q should name the missing var", err.Error())
		}
	})

	t.Run("ReadFile expands env refs in host and apiKey", func(t *testing.T) {
		t.Setenv("TEST_NBU_HOST3", "env-host.example.com")
		t.Setenv("TEST_NBU_KEY3", "env-api-key")

		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "env-config.yaml")
		content := `
server:
  host: "localhost"
  port: "9440"
  uri: "/metrics"
  scrapingInterval: "5m"
  logName: "test.log"
nbuserver:
  host: "${TEST_NBU_HOST3}"
  port: "1556"
  scheme: "https"
  uri: "/netbackup"
  apiKey: "${TEST_NBU_KEY3}"
  apiVersion: "13.0"
`
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		var cfg models.Config
		if err := ReadFile(&cfg, filePath); err != nil {
			t.Fatalf("ReadFile() unexpected error: %v", err)
		}
		if cfg.NbuServer.Host != "env-host.example.com" {
			t.Errorf("Host = %q, want %q", cfg.NbuServer.Host, "env-host.example.com")
		}
		if cfg.NbuServer.APIKey != "env-api-key" {
			t.Errorf("APIKey = %q, want %q", cfg.NbuServer.APIKey, "env-api-key")
		}
	})
}
