package models

import (
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v2"
)

func TestConfigSetDefaults(t *testing.T) {
	tests := []struct {
		name           string
		config         Config
		expectedAPIVer string
	}{
		{
			name: "sets default API version when empty",
			config: Config{
				NbuServer: struct {
					Port               string `yaml:"port"`
					Scheme             string `yaml:"scheme"`
					URI                string `yaml:"uri"`
					Domain             string `yaml:"domain"`
					DomainType         string `yaml:"domainType"`
					Host               string `yaml:"host"`
					APIKey             string `yaml:"apiKey"`
					APIVersion         string `yaml:"apiVersion"`
					ContentType        string `yaml:"contentType"`
					InsecureSkipVerify bool   `yaml:"insecureSkipVerify"`
				}{
					APIVersion: "",
				},
			},
			expectedAPIVer: "13.0",
		},
		{
			name: "preserves existing API version",
			config: Config{
				NbuServer: struct {
					Port               string `yaml:"port"`
					Scheme             string `yaml:"scheme"`
					URI                string `yaml:"uri"`
					Domain             string `yaml:"domain"`
					DomainType         string `yaml:"domainType"`
					Host               string `yaml:"host"`
					APIKey             string `yaml:"apiKey"`
					APIVersion         string `yaml:"apiVersion"`
					ContentType        string `yaml:"contentType"`
					InsecureSkipVerify bool   `yaml:"insecureSkipVerify"`
				}{
					APIVersion: "11.1",
				},
			},
			expectedAPIVer: "11.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.config.SetDefaults()
			if tt.config.NbuServer.APIVersion != tt.expectedAPIVer {
				t.Errorf("SetDefaults() APIVersion = %v, want %v", tt.config.NbuServer.APIVersion, tt.expectedAPIVer)
			}
		})
	}
}

func TestConfigValidateAPIVersion(t *testing.T) {
	tests := []struct {
		name       string
		apiVersion string
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "valid API version 13.0",
			apiVersion: "13.0",
			wantErr:    false,
		},
		{
			name:       "valid API version 12.0",
			apiVersion: "12.0",
			wantErr:    false,
		},
		{
			name:       "valid API version 3.0",
			apiVersion: "3.0",
			wantErr:    false,
		},
		{
			name:       "unsupported API version 11.1",
			apiVersion: "11.1",
			wantErr:    true,
			errMsg:     "unsupported API version",
		},
		{
			name:       "unsupported API version 10.5",
			apiVersion: "10.5",
			wantErr:    true,
			errMsg:     "unsupported API version",
		},
		{
			name:       "empty API version gets default",
			apiVersion: "",
			wantErr:    false,
		},
		{
			name:       "invalid format - no decimal",
			apiVersion: "12",
			wantErr:    true,
			errMsg:     "invalid API version format",
		},
		{
			name:       "invalid format - too many decimals",
			apiVersion: "12.0.1",
			wantErr:    true,
			errMsg:     "invalid API version format",
		},
		{
			name:       "invalid format - non-numeric",
			apiVersion: "v12.0",
			wantErr:    true,
			errMsg:     "invalid API version format",
		},
		{
			name:       "invalid format - letters",
			apiVersion: "abc",
			wantErr:    true,
			errMsg:     "invalid API version format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Server: struct {
					Port             string `yaml:"port"`
					Host             string `yaml:"host"`
					URI              string `yaml:"uri"`
					ScrapingInterval string `yaml:"scrapingInterval"`
					LogName          string `yaml:"logName"`
				}{
					Port:             "2112",
					Host:             "localhost",
					URI:              "/metrics",
					ScrapingInterval: "5m",
					LogName:          "test.log",
				},
				NbuServer: struct {
					Port               string `yaml:"port"`
					Scheme             string `yaml:"scheme"`
					URI                string `yaml:"uri"`
					Domain             string `yaml:"domain"`
					DomainType         string `yaml:"domainType"`
					Host               string `yaml:"host"`
					APIKey             string `yaml:"apiKey"`
					APIVersion         string `yaml:"apiVersion"`
					ContentType        string `yaml:"contentType"`
					InsecureSkipVerify bool   `yaml:"insecureSkipVerify"`
				}{
					Port:       "1556",
					Scheme:     "https",
					URI:        "/netbackup",
					Host:       "nbu-master",
					APIKey:     "test-api-key",
					APIVersion: tt.apiVersion,
				},
			}

			err := config.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error containing %q, got nil", tt.errMsg)
					return
				}
				if tt.errMsg != "" && err.Error() == "" {
					t.Errorf("Validate() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v", err)
				}
				// Verify default was set if empty
				if tt.apiVersion == "" && config.NbuServer.APIVersion != "13.0" {
					t.Errorf("Validate() APIVersion = %v, want default 13.0", config.NbuServer.APIVersion)
				}
			}
		})
	}
}

func TestConfigParseYAMLAPIVersion(t *testing.T) {
	tests := []struct {
		name           string
		yaml           string
		expectedAPIVer string
		wantErr        bool
	}{
		{
			name: "parses API version from YAML",
			yaml: `
server:
  host: "localhost"
  port: "2112"
  uri: "/metrics"
  scrapingInterval: "5m"
  logName: "test.log"
nbuserver:
  host: "nbu-master"
  port: "1556"
  scheme: "https"
  uri: "/netbackup"
  apiKey: "test-key"
  apiVersion: "13.0"
`,
			expectedAPIVer: "13.0",
			wantErr:        false,
		},
		{
			name: "handles missing API version field",
			yaml: `
server:
  host: "localhost"
  port: "2112"
  uri: "/metrics"
  scrapingInterval: "5m"
  logName: "test.log"
nbuserver:
  host: "nbu-master"
  port: "1556"
  scheme: "https"
  uri: "/netbackup"
  apiKey: "test-key"
`,
			expectedAPIVer: "",
			wantErr:        false,
		},
		{
			name: "parses default version 12.0",
			yaml: `
server:
  host: "localhost"
  port: "2112"
  uri: "/metrics"
  scrapingInterval: "5m"
  logName: "test.log"
nbuserver:
  host: "nbu-master"
  port: "1556"
  scheme: "https"
  uri: "/netbackup"
  apiKey: "test-key"
  apiVersion: "12.0"
`,
			expectedAPIVer: "12.0",
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var config Config
			err := yaml.Unmarshal([]byte(tt.yaml), &config)
			if (err != nil) != tt.wantErr {
				t.Errorf("yaml.Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && config.NbuServer.APIVersion != tt.expectedAPIVer {
				t.Errorf("ParseYAML() APIVersion = %v, want %v", config.NbuServer.APIVersion, tt.expectedAPIVer)
			}
		})
	}
}

func TestConfigBackwardCompatibility(t *testing.T) {
	tests := []struct {
		name           string
		yaml           string
		expectedAPIVer string
		shouldValidate bool
	}{
		{
			name: "config without API version field validates with default",
			yaml: `
server:
  host: "localhost"
  port: "2112"
  uri: "/metrics"
  scrapingInterval: "5m"
  logName: "test.log"
nbuserver:
  host: "nbu-master"
  port: "1556"
  scheme: "https"
  uri: "/netbackup"
  apiKey: "test-key"
  contentType: "application/json"
`,
			expectedAPIVer: "13.0",
			shouldValidate: true,
		},
		{
			name: "legacy config structure still works",
			yaml: `
server:
  host: "localhost"
  port: "2112"
  uri: "/metrics"
  scrapingInterval: "1h"
  logName: "nbu.log"
nbuserver:
  host: "master.example.com"
  port: "1556"
  scheme: "https"
  uri: "/netbackup"
  domain: "example.com"
  domainType: "NT"
  apiKey: "legacy-api-key"
  contentType: "application/json"
  insecureSkipVerify: false
`,
			expectedAPIVer: "13.0",
			shouldValidate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var config Config
			err := yaml.Unmarshal([]byte(tt.yaml), &config)
			if err != nil {
				t.Fatalf("yaml.Unmarshal() error = %v", err)
			}

			// Validate the config
			err = config.Validate()
			if tt.shouldValidate && err != nil {
				t.Errorf("Validate() unexpected error = %v", err)
			}
			if !tt.shouldValidate && err == nil {
				t.Error("Validate() expected error, got nil")
			}

			// Check that default API version was set
			if tt.shouldValidate && config.NbuServer.APIVersion != tt.expectedAPIVer {
				t.Errorf("APIVersion = %v, want %v", config.NbuServer.APIVersion, tt.expectedAPIVer)
			}
		})
	}
}

func TestSupportedAPIVersions(t *testing.T) {
	// Test that the supported versions list contains expected versions
	expectedVersions := []string{"13.0", "12.0", "3.0"}

	if len(SupportedAPIVersions) != len(expectedVersions) {
		t.Errorf("SupportedAPIVersions length = %d, want %d", len(SupportedAPIVersions), len(expectedVersions))
	}

	for i, expected := range expectedVersions {
		if SupportedAPIVersions[i] != expected {
			t.Errorf("SupportedAPIVersions[%d] = %s, want %s", i, SupportedAPIVersions[i], expected)
		}
	}
}

func TestAPIVersionConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{
			name:     "APIVersion30 constant",
			constant: APIVersion30,
			expected: "3.0",
		},
		{
			name:     "APIVersion120 constant",
			constant: APIVersion120,
			expected: "12.0",
		},
		{
			name:     "APIVersion130 constant",
			constant: APIVersion130,
			expected: "13.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("%s = %s, want %s", tt.name, tt.constant, tt.expected)
			}
		})
	}
}

func TestConfigGetNBUBaseURL(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected string
	}{
		{
			name: "standard HTTPS URL",
			config: Config{
				NbuServer: struct {
					Port               string `yaml:"port"`
					Scheme             string `yaml:"scheme"`
					URI                string `yaml:"uri"`
					Domain             string `yaml:"domain"`
					DomainType         string `yaml:"domainType"`
					Host               string `yaml:"host"`
					APIKey             string `yaml:"apiKey"`
					APIVersion         string `yaml:"apiVersion"`
					ContentType        string `yaml:"contentType"`
					InsecureSkipVerify bool   `yaml:"insecureSkipVerify"`
				}{
					Scheme: "https",
					Host:   "nbu-master.example.com",
					Port:   "1556",
					URI:    "/netbackup",
				},
			},
			expected: "https://nbu-master.example.com:1556/netbackup",
		},
		{
			name: "HTTP URL with different port",
			config: Config{
				NbuServer: struct {
					Port               string `yaml:"port"`
					Scheme             string `yaml:"scheme"`
					URI                string `yaml:"uri"`
					Domain             string `yaml:"domain"`
					DomainType         string `yaml:"domainType"`
					Host               string `yaml:"host"`
					APIKey             string `yaml:"apiKey"`
					APIVersion         string `yaml:"apiVersion"`
					ContentType        string `yaml:"contentType"`
					InsecureSkipVerify bool   `yaml:"insecureSkipVerify"`
				}{
					Scheme: "http",
					Host:   "localhost",
					Port:   "8080",
					URI:    "/api",
				},
			},
			expected: "http://localhost:8080/api",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetNBUBaseURL()
			if result != tt.expected {
				t.Errorf("GetNBUBaseURL() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConfigGetServerAddress(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected string
	}{
		{
			name: "standard server address",
			config: Config{
				Server: struct {
					Port             string `yaml:"port"`
					Host             string `yaml:"host"`
					URI              string `yaml:"uri"`
					ScrapingInterval string `yaml:"scrapingInterval"`
					LogName          string `yaml:"logName"`
				}{
					Host: "0.0.0.0",
					Port: "2112",
				},
			},
			expected: "0.0.0.0:2112",
		},
		{
			name: "localhost with custom port",
			config: Config{
				Server: struct {
					Port             string `yaml:"port"`
					Host             string `yaml:"host"`
					URI              string `yaml:"uri"`
					ScrapingInterval string `yaml:"scrapingInterval"`
					LogName          string `yaml:"logName"`
				}{
					Host: "localhost",
					Port: "9090",
				},
			},
			expected: "localhost:9090",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetServerAddress()
			if result != tt.expected {
				t.Errorf("GetServerAddress() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConfigGetScrapingDuration(t *testing.T) {
	tests := []struct {
		name        string
		interval    string
		expected    time.Duration
		expectError bool
	}{
		{
			name:        "5 minutes",
			interval:    "5m",
			expected:    5 * time.Minute,
			expectError: false,
		},
		{
			name:        "1 hour",
			interval:    "1h",
			expected:    1 * time.Hour,
			expectError: false,
		},
		{
			name:        "30 seconds",
			interval:    "30s",
			expected:    30 * time.Second,
			expectError: false,
		},
		{
			name:        "invalid format",
			interval:    "invalid",
			expected:    0,
			expectError: true,
		},
		{
			name:        "empty string",
			interval:    "",
			expected:    0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{
				Server: struct {
					Port             string `yaml:"port"`
					Host             string `yaml:"host"`
					URI              string `yaml:"uri"`
					ScrapingInterval string `yaml:"scrapingInterval"`
					LogName          string `yaml:"logName"`
				}{
					ScrapingInterval: tt.interval,
				},
			}

			result, err := config.GetScrapingDuration()

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("GetScrapingDuration() = %v, want %v", result, tt.expected)
				}
			}
		})
	}
}

func TestConfigMaskAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		apiKey   string
		expected string
	}{
		{
			name:     "standard API key",
			apiKey:   "abcd1234efgh5678ijkl",
			expected: "abcd****ijkl",
		},
		{
			name:     "short API key",
			apiKey:   "short",
			expected: "****",
		},
		{
			name:     "exactly 8 characters",
			apiKey:   "12345678",
			expected: "****",
		},
		{
			name:     "9 characters",
			apiKey:   "123456789",
			expected: "1234****6789",
		},
		{
			name:     "empty string",
			apiKey:   "",
			expected: "****",
		},
		{
			name:     "very long key",
			apiKey:   "abcdefghijklmnopqrstuvwxyz0123456789",
			expected: "abcd****6789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{
				NbuServer: struct {
					Port               string `yaml:"port"`
					Scheme             string `yaml:"scheme"`
					URI                string `yaml:"uri"`
					Domain             string `yaml:"domain"`
					DomainType         string `yaml:"domainType"`
					Host               string `yaml:"host"`
					APIKey             string `yaml:"apiKey"`
					APIVersion         string `yaml:"apiVersion"`
					ContentType        string `yaml:"contentType"`
					InsecureSkipVerify bool   `yaml:"insecureSkipVerify"`
				}{
					APIKey: tt.apiKey,
				},
			}

			result := config.MaskAPIKey()
			if result != tt.expected {
				t.Errorf("MaskAPIKey() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConfigBuildURL(t *testing.T) {
	config := Config{
		NbuServer: struct {
			Port               string `yaml:"port"`
			Scheme             string `yaml:"scheme"`
			URI                string `yaml:"uri"`
			Domain             string `yaml:"domain"`
			DomainType         string `yaml:"domainType"`
			Host               string `yaml:"host"`
			APIKey             string `yaml:"apiKey"`
			APIVersion         string `yaml:"apiVersion"`
			ContentType        string `yaml:"contentType"`
			InsecureSkipVerify bool   `yaml:"insecureSkipVerify"`
		}{
			Scheme: "https",
			Host:   "nbu-master",
			Port:   "1556",
			URI:    "/netbackup",
		},
	}

	tests := []struct {
		name        string
		path        string
		queryParams map[string]string
		contains    []string
	}{
		{
			name:        "simple path without query params",
			path:        "/admin/jobs",
			queryParams: map[string]string{},
			contains: []string{
				"https://nbu-master:1556/admin/jobs",
			},
		},
		{
			name: "path with single query param",
			path: "/admin/jobs",
			queryParams: map[string]string{
				"page[limit]": "100",
			},
			contains: []string{
				"https://nbu-master:1556/admin/jobs",
				"page%5Blimit%5D=100",
			},
		},
		{
			name: "path with multiple query params",
			path: "/storage/storage-units",
			queryParams: map[string]string{
				"page[limit]":  "50",
				"page[offset]": "0",
				"filter[type]": "DISK",
			},
			contains: []string{
				"https://nbu-master:1556/storage/storage-units",
				"page%5Blimit%5D=50",
				"page%5Boffset%5D=0",
				"filter%5Btype%5D=DISK",
			},
		},
		{
			name: "path with special characters in params",
			path: "/admin/jobs",
			queryParams: map[string]string{
				"filter[startTime]": "2024-11-08T10:00:00Z",
			},
			contains: []string{
				"https://nbu-master:1556/admin/jobs",
				"filter%5BstartTime%5D=2024-11-08T10%3A00%3A00Z",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := config.BuildURL(tt.path, tt.queryParams)

			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("BuildURL() result does not contain %q\nGot: %s", expected, result)
				}
			}
		})
	}
}

func TestConfigValidateServerFields(t *testing.T) {
	baseConfig := func() Config {
		return Config{
			Server: struct {
				Port             string `yaml:"port"`
				Host             string `yaml:"host"`
				URI              string `yaml:"uri"`
				ScrapingInterval string `yaml:"scrapingInterval"`
				LogName          string `yaml:"logName"`
			}{
				Port:             "2112",
				Host:             "localhost",
				URI:              "/metrics",
				ScrapingInterval: "5m",
			},
			NbuServer: struct {
				Port               string `yaml:"port"`
				Scheme             string `yaml:"scheme"`
				URI                string `yaml:"uri"`
				Domain             string `yaml:"domain"`
				DomainType         string `yaml:"domainType"`
				Host               string `yaml:"host"`
				APIKey             string `yaml:"apiKey"`
				APIVersion         string `yaml:"apiVersion"`
				ContentType        string `yaml:"contentType"`
				InsecureSkipVerify bool   `yaml:"insecureSkipVerify"`
			}{
				Port:   "1556",
				Scheme: "https",
				URI:    "/netbackup",
				Host:   "nbu-master",
				APIKey: "test-key",
			},
		}
	}

	tests := []struct {
		name      string
		modify    func(*Config)
		wantError bool
		errMsg    string
	}{
		{
			name:      "valid config",
			modify:    func(c *Config) {},
			wantError: false,
		},
		{
			name: "missing server port",
			modify: func(c *Config) {
				c.Server.Port = ""
			},
			wantError: true,
			errMsg:    "server port is required",
		},
		{
			name: "invalid server port - too high",
			modify: func(c *Config) {
				c.Server.Port = "99999"
			},
			wantError: true,
			errMsg:    "invalid server port",
		},
		{
			name: "invalid server port - negative",
			modify: func(c *Config) {
				c.Server.Port = "-1"
			},
			wantError: true,
			errMsg:    "invalid server port",
		},
		{
			name: "invalid server port - non-numeric",
			modify: func(c *Config) {
				c.Server.Port = "abc"
			},
			wantError: true,
			errMsg:    "invalid server port",
		},
		{
			name: "missing server host",
			modify: func(c *Config) {
				c.Server.Host = ""
			},
			wantError: true,
			errMsg:    "server host is required",
		},
		{
			name: "missing server URI",
			modify: func(c *Config) {
				c.Server.URI = ""
			},
			wantError: true,
			errMsg:    "server URI is required",
		},
		{
			name: "invalid scraping interval",
			modify: func(c *Config) {
				c.Server.ScrapingInterval = "invalid"
			},
			wantError: true,
			errMsg:    "invalid scraping interval",
		},
		{
			name: "missing NBU host",
			modify: func(c *Config) {
				c.NbuServer.Host = ""
			},
			wantError: true,
			errMsg:    "NBU server host is required",
		},
		{
			name: "missing NBU port",
			modify: func(c *Config) {
				c.NbuServer.Port = ""
			},
			wantError: true,
			errMsg:    "NBU server port is required",
		},
		{
			name: "invalid NBU scheme",
			modify: func(c *Config) {
				c.NbuServer.Scheme = "ftp"
			},
			wantError: true,
			errMsg:    "invalid NBU server scheme",
		},
		{
			name: "missing API key",
			modify: func(c *Config) {
				c.NbuServer.APIKey = ""
			},
			wantError: true,
			errMsg:    "NBU server API key is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := baseConfig()
			tt.modify(&config)

			err := config.Validate()

			if tt.wantError {
				if err == nil {
					t.Error("Expected error, got nil")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestConfigValidateSupportedVersions(t *testing.T) {
	tests := []struct {
		name       string
		apiVersion string
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "supported version 13.0",
			apiVersion: "13.0",
			wantErr:    false,
		},
		{
			name:       "supported version 12.0",
			apiVersion: "12.0",
			wantErr:    false,
		},
		{
			name:       "supported version 3.0",
			apiVersion: "3.0",
			wantErr:    false,
		},
		{
			name:       "unsupported version 14.0",
			apiVersion: "14.0",
			wantErr:    true,
			errMsg:     "unsupported API version: 14.0",
		},
		{
			name:       "unsupported version 2.0",
			apiVersion: "2.0",
			wantErr:    true,
			errMsg:     "unsupported API version: 2.0",
		},
		{
			name:       "unsupported version 11.0",
			apiVersion: "11.0",
			wantErr:    true,
			errMsg:     "unsupported API version: 11.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Server: struct {
					Port             string `yaml:"port"`
					Host             string `yaml:"host"`
					URI              string `yaml:"uri"`
					ScrapingInterval string `yaml:"scrapingInterval"`
					LogName          string `yaml:"logName"`
				}{
					Port:             "2112",
					Host:             "localhost",
					URI:              "/metrics",
					ScrapingInterval: "5m",
					LogName:          "test.log",
				},
				NbuServer: struct {
					Port               string `yaml:"port"`
					Scheme             string `yaml:"scheme"`
					URI                string `yaml:"uri"`
					Domain             string `yaml:"domain"`
					DomainType         string `yaml:"domainType"`
					Host               string `yaml:"host"`
					APIKey             string `yaml:"apiKey"`
					APIVersion         string `yaml:"apiVersion"`
					ContentType        string `yaml:"contentType"`
					InsecureSkipVerify bool   `yaml:"insecureSkipVerify"`
				}{
					Port:       "1556",
					Scheme:     "https",
					URI:        "/netbackup",
					Host:       "nbu-master",
					APIKey:     "test-api-key",
					APIVersion: tt.apiVersion,
				},
			}

			err := config.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error, got nil")
					return
				}
				// Check if error message contains the expected substring
				if tt.errMsg != "" {
					errStr := err.Error()
					if len(errStr) < len(tt.errMsg) || errStr[:len(tt.errMsg)] != tt.errMsg {
						t.Errorf("Validate() error = %v, want error starting with %v", err.Error(), tt.errMsg)
					}
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestConfigValidateOpenTelemetry(t *testing.T) {
	baseConfig := func() Config {
		return Config{
			Server: struct {
				Port             string `yaml:"port"`
				Host             string `yaml:"host"`
				URI              string `yaml:"uri"`
				ScrapingInterval string `yaml:"scrapingInterval"`
				LogName          string `yaml:"logName"`
			}{
				Port:             "2112",
				Host:             "localhost",
				URI:              "/metrics",
				ScrapingInterval: "5m",
			},
			NbuServer: struct {
				Port               string `yaml:"port"`
				Scheme             string `yaml:"scheme"`
				URI                string `yaml:"uri"`
				Domain             string `yaml:"domain"`
				DomainType         string `yaml:"domainType"`
				Host               string `yaml:"host"`
				APIKey             string `yaml:"apiKey"`
				APIVersion         string `yaml:"apiVersion"`
				ContentType        string `yaml:"contentType"`
				InsecureSkipVerify bool   `yaml:"insecureSkipVerify"`
			}{
				Port:   "1556",
				Scheme: "https",
				URI:    "/netbackup",
				Host:   "nbu-master",
				APIKey: "test-key",
			},
		}
	}

	tests := []struct {
		name      string
		modify    func(*Config)
		wantError bool
		errMsg    string
	}{
		{
			name: "valid OpenTelemetry config",
			modify: func(c *Config) {
				c.OpenTelemetry.Enabled = true
				c.OpenTelemetry.Endpoint = "localhost:4317"
				c.OpenTelemetry.Insecure = true
				c.OpenTelemetry.SamplingRate = 0.1
			},
			wantError: false,
		},
		{
			name: "OpenTelemetry disabled - no validation",
			modify: func(c *Config) {
				c.OpenTelemetry.Enabled = false
				c.OpenTelemetry.Endpoint = ""
				c.OpenTelemetry.SamplingRate = 2.0
			},
			wantError: false,
		},
		{
			name: "missing endpoint when enabled",
			modify: func(c *Config) {
				c.OpenTelemetry.Enabled = true
				c.OpenTelemetry.Endpoint = ""
			},
			wantError: true,
			errMsg:    "OpenTelemetry endpoint is required when enabled",
		},
		{
			name: "sampling rate too low",
			modify: func(c *Config) {
				c.OpenTelemetry.Enabled = true
				c.OpenTelemetry.Endpoint = "localhost:4317"
				c.OpenTelemetry.SamplingRate = -0.1
			},
			wantError: true,
			errMsg:    "OpenTelemetry sampling rate must be between 0.0 and 1.0",
		},
		{
			name: "sampling rate too high",
			modify: func(c *Config) {
				c.OpenTelemetry.Enabled = true
				c.OpenTelemetry.Endpoint = "localhost:4317"
				c.OpenTelemetry.SamplingRate = 1.5
			},
			wantError: true,
			errMsg:    "OpenTelemetry sampling rate must be between 0.0 and 1.0",
		},
		{
			name: "sampling rate at lower bound",
			modify: func(c *Config) {
				c.OpenTelemetry.Enabled = true
				c.OpenTelemetry.Endpoint = "localhost:4317"
				c.OpenTelemetry.SamplingRate = 0.0
			},
			wantError: false,
		},
		{
			name: "sampling rate at upper bound",
			modify: func(c *Config) {
				c.OpenTelemetry.Enabled = true
				c.OpenTelemetry.Endpoint = "localhost:4317"
				c.OpenTelemetry.SamplingRate = 1.0
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := baseConfig()
			tt.modify(&config)

			err := config.Validate()

			if tt.wantError {
				if err == nil {
					t.Error("Expected error, got nil")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestConfigIsOTelEnabled(t *testing.T) {
	tests := []struct {
		name     string
		enabled  bool
		expected bool
	}{
		{
			name:     "OpenTelemetry enabled",
			enabled:  true,
			expected: true,
		},
		{
			name:     "OpenTelemetry disabled",
			enabled:  false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{
				OpenTelemetry: struct {
					Enabled      bool    `yaml:"enabled"`
					Endpoint     string  `yaml:"endpoint"`
					Insecure     bool    `yaml:"insecure"`
					SamplingRate float64 `yaml:"samplingRate"`
				}{
					Enabled: tt.enabled,
				},
			}

			result := config.IsOTelEnabled()
			if result != tt.expected {
				t.Errorf("IsOTelEnabled() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConfigGetOTelConfig(t *testing.T) {
	tests := []struct {
		name         string
		enabled      bool
		endpoint     string
		insecure     bool
		samplingRate float64
	}{
		{
			name:         "full OpenTelemetry config",
			enabled:      true,
			endpoint:     "localhost:4317",
			insecure:     true,
			samplingRate: 0.1,
		},
		{
			name:         "disabled OpenTelemetry",
			enabled:      false,
			endpoint:     "",
			insecure:     false,
			samplingRate: 0.0,
		},
		{
			name:         "secure endpoint with full sampling",
			enabled:      true,
			endpoint:     "otel-collector.example.com:4317",
			insecure:     false,
			samplingRate: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{
				OpenTelemetry: struct {
					Enabled      bool    `yaml:"enabled"`
					Endpoint     string  `yaml:"endpoint"`
					Insecure     bool    `yaml:"insecure"`
					SamplingRate float64 `yaml:"samplingRate"`
				}{
					Enabled:      tt.enabled,
					Endpoint:     tt.endpoint,
					Insecure:     tt.insecure,
					SamplingRate: tt.samplingRate,
				},
			}

			result := config.GetOTelConfig()

			if result.Enabled != tt.enabled {
				t.Errorf("GetOTelConfig().Enabled = %v, want %v", result.Enabled, tt.enabled)
			}
			if result.Endpoint != tt.endpoint {
				t.Errorf("GetOTelConfig().Endpoint = %v, want %v", result.Endpoint, tt.endpoint)
			}
			if result.Insecure != tt.insecure {
				t.Errorf("GetOTelConfig().Insecure = %v, want %v", result.Insecure, tt.insecure)
			}
			if result.SamplingRate != tt.samplingRate {
				t.Errorf("GetOTelConfig().SamplingRate = %v, want %v", result.SamplingRate, tt.samplingRate)
			}
		})
	}
}

func TestConfigValidateOTelEndpoint(t *testing.T) {
	baseConfig := func() Config {
		return Config{
			Server: struct {
				Port             string `yaml:"port"`
				Host             string `yaml:"host"`
				URI              string `yaml:"uri"`
				ScrapingInterval string `yaml:"scrapingInterval"`
				LogName          string `yaml:"logName"`
			}{
				Port:             "2112",
				Host:             "localhost",
				URI:              "/metrics",
				ScrapingInterval: "5m",
			},
			NbuServer: struct {
				Port               string `yaml:"port"`
				Scheme             string `yaml:"scheme"`
				URI                string `yaml:"uri"`
				Domain             string `yaml:"domain"`
				DomainType         string `yaml:"domainType"`
				Host               string `yaml:"host"`
				APIKey             string `yaml:"apiKey"`
				APIVersion         string `yaml:"apiVersion"`
				ContentType        string `yaml:"contentType"`
				InsecureSkipVerify bool   `yaml:"insecureSkipVerify"`
			}{
				Port:   "1556",
				Scheme: "https",
				URI:    "/netbackup",
				Host:   "nbu-master",
				APIKey: "test-key",
			},
		}
	}

	tests := []struct {
		name      string
		endpoint  string
		wantError bool
		errMsg    string
	}{
		{
			name:      "valid endpoint localhost:4317",
			endpoint:  "localhost:4317",
			wantError: false,
		},
		{
			name:      "valid endpoint with IP address",
			endpoint:  "192.168.1.100:4317",
			wantError: false,
		},
		{
			name:      "valid endpoint with hostname",
			endpoint:  "otel-collector.example.com:4317",
			wantError: false,
		},
		{
			name:      "valid endpoint with IPv6 address",
			endpoint:  "[::1]:4317",
			wantError: false,
		},
		{
			name:      "valid endpoint with port 1",
			endpoint:  "localhost:1",
			wantError: false,
		},
		{
			name:      "valid endpoint with port 65535",
			endpoint:  "localhost:65535",
			wantError: false,
		},
		{
			name:      "empty endpoint",
			endpoint:  "",
			wantError: true,
			errMsg:    "OpenTelemetry endpoint is required when enabled",
		},
		{
			name:      "missing port",
			endpoint:  "localhost",
			wantError: true,
			errMsg:    "invalid OpenTelemetry endpoint format",
		},
		{
			name:      "missing host",
			endpoint:  ":4317",
			wantError: true,
			errMsg:    "invalid OpenTelemetry endpoint format",
		},
		{
			name:      "invalid port - non-numeric",
			endpoint:  "localhost:abc",
			wantError: true,
			errMsg:    "invalid OpenTelemetry endpoint port",
		},
		{
			name:      "invalid port - zero",
			endpoint:  "localhost:0",
			wantError: true,
			errMsg:    "invalid OpenTelemetry endpoint port: 0 (must be between 1 and 65535)",
		},
		{
			name:      "invalid port - negative",
			endpoint:  "localhost:-1",
			wantError: true,
			errMsg:    "invalid OpenTelemetry endpoint port",
		},
		{
			name:      "invalid port - too high",
			endpoint:  "localhost:65536",
			wantError: true,
			errMsg:    "invalid OpenTelemetry endpoint port: 65536 (must be between 1 and 65535)",
		},
		{
			name:      "invalid port - way too high",
			endpoint:  "localhost:99999",
			wantError: true,
			errMsg:    "invalid OpenTelemetry endpoint port: 99999 (must be between 1 and 65535)",
		},
		{
			name:      "multiple colons without brackets",
			endpoint:  "host:with:colons:4317",
			wantError: false, // Takes last colon as separator
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := baseConfig()
			config.OpenTelemetry.Enabled = true
			config.OpenTelemetry.Endpoint = tt.endpoint

			err := config.Validate()

			if tt.wantError {
				if err == nil {
					t.Error("Expected error, got nil")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestConfigParseYAMLOpenTelemetry(t *testing.T) {
	tests := []struct {
		name         string
		yaml         string
		enabled      bool
		endpoint     string
		insecure     bool
		samplingRate float64
		wantErr      bool
	}{
		{
			name: "parses OpenTelemetry config from YAML",
			yaml: `
server:
  host: "localhost"
  port: "2112"
  uri: "/metrics"
  scrapingInterval: "5m"
  logName: "test.log"
nbuserver:
  host: "nbu-master"
  port: "1556"
  scheme: "https"
  uri: "/netbackup"
  apiKey: "test-key"
opentelemetry:
  enabled: true
  endpoint: "localhost:4317"
  insecure: true
  samplingRate: 0.1
`,
			enabled:      true,
			endpoint:     "localhost:4317",
			insecure:     true,
			samplingRate: 0.1,
			wantErr:      false,
		},
		{
			name: "handles missing OpenTelemetry section",
			yaml: `
server:
  host: "localhost"
  port: "2112"
  uri: "/metrics"
  scrapingInterval: "5m"
  logName: "test.log"
nbuserver:
  host: "nbu-master"
  port: "1556"
  scheme: "https"
  uri: "/netbackup"
  apiKey: "test-key"
`,
			enabled:      false,
			endpoint:     "",
			insecure:     false,
			samplingRate: 0.0,
			wantErr:      false,
		},
		{
			name: "parses disabled OpenTelemetry",
			yaml: `
server:
  host: "localhost"
  port: "2112"
  uri: "/metrics"
  scrapingInterval: "5m"
  logName: "test.log"
nbuserver:
  host: "nbu-master"
  port: "1556"
  scheme: "https"
  uri: "/netbackup"
  apiKey: "test-key"
opentelemetry:
  enabled: false
  endpoint: "localhost:4317"
  insecure: false
  samplingRate: 0.5
`,
			enabled:      false,
			endpoint:     "localhost:4317",
			insecure:     false,
			samplingRate: 0.5,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var config Config
			err := yaml.Unmarshal([]byte(tt.yaml), &config)
			if (err != nil) != tt.wantErr {
				t.Errorf("yaml.Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if config.OpenTelemetry.Enabled != tt.enabled {
					t.Errorf("ParseYAML() OpenTelemetry.Enabled = %v, want %v", config.OpenTelemetry.Enabled, tt.enabled)
				}
				if config.OpenTelemetry.Endpoint != tt.endpoint {
					t.Errorf("ParseYAML() OpenTelemetry.Endpoint = %v, want %v", config.OpenTelemetry.Endpoint, tt.endpoint)
				}
				if config.OpenTelemetry.Insecure != tt.insecure {
					t.Errorf("ParseYAML() OpenTelemetry.Insecure = %v, want %v", config.OpenTelemetry.Insecure, tt.insecure)
				}
				if config.OpenTelemetry.SamplingRate != tt.samplingRate {
					t.Errorf("ParseYAML() OpenTelemetry.SamplingRate = %v, want %v", config.OpenTelemetry.SamplingRate, tt.samplingRate)
				}
			}
		})
	}
}
