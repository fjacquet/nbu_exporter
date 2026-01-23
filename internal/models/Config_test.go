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

// createConfigWithAPIVersion creates a test config with the specified API version
func createConfigWithAPIVersion(apiVersion string) *Config {
	return &Config{
		Server: struct {
			Port             string `yaml:"port"`
			Host             string `yaml:"host"`
			URI              string `yaml:"uri"`
			ScrapingInterval string `yaml:"scrapingInterval"`
			LogName          string `yaml:"logName"`
		}{
			Port:             "2112",
			Host:             "localhost",
			URI:              testPathMetrics,
			ScrapingInterval: "5m",
			LogName:          testLogName,
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
			URI:        testPathNetBackup,
			Host:       testServerNBUMaster,
			APIKey:     "test-api-key",
			APIVersion: apiVersion,
		},
	}
}

// assertAPIVersionValidation validates the API version validation result
func assertAPIVersionValidation(t *testing.T, config *Config, err error, apiVersion string, wantErr bool, errMsg string) {
	if wantErr {
		if err == nil {
			t.Errorf("Validate() expected error containing %q, got nil", errMsg)
			return
		}
		if errMsg != "" && err.Error() == "" {
			t.Errorf("Validate() error = %v, want error containing %q", err, errMsg)
		}
	} else {
		if err != nil {
			t.Errorf(testErrorValidateUnexpected, err)
		}
		// Verify default was set if empty
		if apiVersion == "" && config.NbuServer.APIVersion != "13.0" {
			t.Errorf("Validate() APIVersion = %v, want default 13.0", config.NbuServer.APIVersion)
		}
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
			errMsg:     testInvalidAPIVersion,
		},
		{
			name:       "invalid format - too many decimals",
			apiVersion: "12.0.1",
			wantErr:    true,
			errMsg:     testInvalidAPIVersion,
		},
		{
			name:       "invalid format - non-numeric",
			apiVersion: "v12.0",
			wantErr:    true,
			errMsg:     testInvalidAPIVersion,
		},
		{
			name:       "invalid format - letters",
			apiVersion: "abc",
			wantErr:    true,
			errMsg:     testInvalidAPIVersion,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := createConfigWithAPIVersion(tt.apiVersion)
			err := config.Validate()
			assertAPIVersionValidation(t, config, err, tt.apiVersion, tt.wantErr, tt.errMsg)
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
  uri: testPathMetrics
  scrapingInterval: "5m"
  logName: testLogName
nbuserver:
  host: testServerNBUMaster
  port: "1556"
  scheme: "https"
  uri: testPathNetBackup
  apiKey: testKeyName
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
  uri: testPathMetrics
  scrapingInterval: "5m"
  logName: testLogName
nbuserver:
  host: testServerNBUMaster
  port: "1556"
  scheme: "https"
  uri: testPathNetBackup
  apiKey: testKeyName
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
  uri: testPathMetrics
  scrapingInterval: "5m"
  logName: testLogName
nbuserver:
  host: testServerNBUMaster
  port: "1556"
  scheme: "https"
  uri: testPathNetBackup
  apiKey: testKeyName
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

// parseAndValidateYAMLConfig parses YAML and validates the config
func parseAndValidateYAMLConfig(t *testing.T, yamlContent string) (Config, error) {
	var config Config
	err := yaml.Unmarshal([]byte(yamlContent), &config)
	if err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}
	return config, config.Validate()
}

// assertBackwardCompatibility validates backward compatibility expectations
func assertBackwardCompatibility(t *testing.T, config Config, err error, expectedAPIVer string, shouldValidate bool) {
	if shouldValidate && err != nil {
		t.Errorf(testErrorValidateUnexpected, err)
	}
	if !shouldValidate && err == nil {
		t.Error("Validate() expected error, got nil")
	}
	if shouldValidate && config.NbuServer.APIVersion != expectedAPIVer {
		t.Errorf("APIVersion = %v, want %v", config.NbuServer.APIVersion, expectedAPIVer)
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
			config, err := parseAndValidateYAMLConfig(t, tt.yaml)
			assertBackwardCompatibility(t, config, err, tt.expectedAPIVer, tt.shouldValidate)
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

// createConfigWithNBUServer creates a config with the specified NBU server settings
func createConfigWithNBUServer(scheme, host, port, uri string) Config {
	return Config{
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
			Scheme: scheme,
			Host:   host,
			Port:   port,
			URI:    uri,
		},
	}
}

func TestConfigGetNBUBaseURL(t *testing.T) {
	tests := []struct {
		name     string
		scheme   string
		host     string
		port     string
		uri      string
		expected string
	}{
		{
			name:     "standard HTTPS URL",
			scheme:   "https",
			host:     "nbu-master.example.com",
			port:     "1556",
			uri:      testPathNetBackup,
			expected: "https://nbu-master.example.com:1556/netbackup",
		},
		{
			name:     "HTTP URL with different port",
			scheme:   "http",
			host:     "localhost",
			port:     "8080",
			uri:      "/api",
			expected: "http://localhost:8080/api",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := createConfigWithNBUServer(tt.scheme, tt.host, tt.port, tt.uri)
			result := config.GetNBUBaseURL()
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

// createConfigWithInterval creates a config with the specified scraping interval
func createConfigWithInterval(interval string) Config {
	return Config{
		Server: struct {
			Port             string `yaml:"port"`
			Host             string `yaml:"host"`
			URI              string `yaml:"uri"`
			ScrapingInterval string `yaml:"scrapingInterval"`
			LogName          string `yaml:"logName"`
		}{
			ScrapingInterval: interval,
		},
	}
}

// assertDurationResult checks if the duration result matches expectations
func assertDurationResult(t *testing.T, result time.Duration, err error, expected time.Duration, expectError bool) {
	if expectError {
		if err == nil {
			t.Error(testErrorExpectedError)
		}
	} else {
		if err != nil {
			t.Errorf(testErrorUnexpected, err)
		}
		if result != expected {
			t.Errorf("GetScrapingDuration() = %v, want %v", result, expected)
		}
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
			config := createConfigWithInterval(tt.interval)
			result, err := config.GetScrapingDuration()
			assertDurationResult(t, result, err, tt.expected, tt.expectError)
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
			Host:   testServerNBUMaster,
			Port:   "1556",
			URI:    testPathNetBackup,
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
			path:        testPathAdminJobs,
			queryParams: map[string]string{},
			contains: []string{
				testSchemeHTTPS + "://" + testServerNBUMaster + ":" + testPort1556 + testPathNetBackup + testPathAdminJobs,
			},
		},
		{
			name: "path with single query param",
			path: "/admin/jobs",
			queryParams: map[string]string{
				"page[limit]": "100",
			},
			contains: []string{
				testSchemeHTTPS + "://" + testServerNBUMaster + ":" + testPort1556 + testPathNetBackup + testPathAdminJobs,
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
				"https://nbu-master:1556/netbackup/storage/storage-units",
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
				"https://nbu-master:1556/netbackup/admin/jobs",
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
				URI:              testPathMetrics,
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
				URI:    testPathNetBackup,
				Host:   testServerNBUMaster,
				APIKey: testKeyName,
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
			name: "valid config",
			modify: func(c *Config) {
				// No modifications needed - testing default valid config
			},
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
			errMsg:    testInvalidServerPort,
		},
		{
			name: "invalid server port - negative",
			modify: func(c *Config) {
				c.Server.Port = "-1"
			},
			wantError: true,
			errMsg:    testInvalidServerPort,
		},
		{
			name: "invalid server port - non-numeric",
			modify: func(c *Config) {
				c.Server.Port = "abc"
			},
			wantError: true,
			errMsg:    testInvalidServerPort,
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
					t.Error(testErrorExpectedError)
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf(testErrorExpectedErrorContaining, tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf(testErrorUnexpected, err)
				}
			}
		})
	}
}

// assertSupportedVersionValidation validates the supported version validation result
func assertSupportedVersionValidation(t *testing.T, err error, wantErr bool, errMsg string) {
	if wantErr {
		if err == nil {
			t.Errorf("Validate() expected error, got nil")
			return
		}
		if errMsg != "" {
			errStr := err.Error()
			if len(errStr) < len(errMsg) || errStr[:len(errMsg)] != errMsg {
				t.Errorf("Validate() error = %v, want error starting with %v", err.Error(), errMsg)
			}
		}
	} else {
		if err != nil {
			t.Errorf(testErrorValidateUnexpected, err)
		}
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
			config := createConfigWithAPIVersion(tt.apiVersion)
			err := config.Validate()
			assertSupportedVersionValidation(t, err, tt.wantErr, tt.errMsg)
		})
	}
}

// createBaseTestConfig creates a base configuration for testing
func createBaseTestConfig() Config {
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
			URI:              testPathMetrics,
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
			URI:    testPathNetBackup,
			Host:   testServerNBUMaster,
			APIKey: testKeyName,
		},
	}
}

// assertValidationResult checks if the validation result matches expectations
func assertValidationResult(t *testing.T, err error, wantError bool, errMsg string) {
	if wantError {
		if err == nil {
			t.Error(testErrorExpectedError)
		} else if errMsg != "" && !strings.Contains(err.Error(), errMsg) {
			t.Errorf(testErrorExpectedErrorContaining, errMsg, err.Error())
		}
	} else {
		if err != nil {
			t.Errorf(testErrorUnexpected, err)
		}
	}
}

func TestConfigValidateOpenTelemetry(t *testing.T) {
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
				c.OpenTelemetry.Endpoint = testOTELEndpoint
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
				c.OpenTelemetry.Endpoint = testOTELEndpoint
				c.OpenTelemetry.SamplingRate = -0.1
			},
			wantError: true,
			errMsg:    "OpenTelemetry sampling rate must be between 0.0 and 1.0",
		},
		{
			name: "sampling rate too high",
			modify: func(c *Config) {
				c.OpenTelemetry.Enabled = true
				c.OpenTelemetry.Endpoint = testOTELEndpoint
				c.OpenTelemetry.SamplingRate = 1.5
			},
			wantError: true,
			errMsg:    "OpenTelemetry sampling rate must be between 0.0 and 1.0",
		},
		{
			name: "sampling rate at lower bound",
			modify: func(c *Config) {
				c.OpenTelemetry.Enabled = true
				c.OpenTelemetry.Endpoint = testOTELEndpoint
				c.OpenTelemetry.SamplingRate = 0.0
			},
			wantError: false,
		},
		{
			name: "sampling rate at upper bound",
			modify: func(c *Config) {
				c.OpenTelemetry.Enabled = true
				c.OpenTelemetry.Endpoint = testOTELEndpoint
				c.OpenTelemetry.SamplingRate = 1.0
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := createBaseTestConfig()
			tt.modify(&config)
			err := config.Validate()
			assertValidationResult(t, err, tt.wantError, tt.errMsg)
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
			endpoint:     testOTELEndpoint,
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
				URI:              testPathMetrics,
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
				URI:    testPathNetBackup,
				Host:   testServerNBUMaster,
				APIKey: testKeyName,
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
			endpoint:  testOTELEndpoint,
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
					t.Error(testErrorExpectedError)
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf(testErrorExpectedErrorContaining, tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf(testErrorUnexpected, err)
				}
			}
		})
	}
}

// assertOTelConfigFields validates OpenTelemetry configuration fields
func assertOTelConfigFields(t *testing.T, config Config, enabled bool, endpoint string, insecure bool, samplingRate float64) {
	if config.OpenTelemetry.Enabled != enabled {
		t.Errorf("ParseYAML() OpenTelemetry.Enabled = %v, want %v", config.OpenTelemetry.Enabled, enabled)
	}
	if config.OpenTelemetry.Endpoint != endpoint {
		t.Errorf("ParseYAML() OpenTelemetry.Endpoint = %v, want %v", config.OpenTelemetry.Endpoint, endpoint)
	}
	if config.OpenTelemetry.Insecure != insecure {
		t.Errorf("ParseYAML() OpenTelemetry.Insecure = %v, want %v", config.OpenTelemetry.Insecure, insecure)
	}
	if config.OpenTelemetry.SamplingRate != samplingRate {
		t.Errorf("ParseYAML() OpenTelemetry.SamplingRate = %v, want %v", config.OpenTelemetry.SamplingRate, samplingRate)
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
  uri: testPathMetrics
  scrapingInterval: "5m"
  logName: testLogName
nbuserver:
  host: testServerNBUMaster
  port: "1556"
  scheme: "https"
  uri: testPathNetBackup
  apiKey: testKeyName
opentelemetry:
  enabled: true
  endpoint: localhost:4317
  insecure: true
  samplingRate: 0.1
`,
			enabled:      true,
			endpoint:     testOTELEndpoint,
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
  uri: testPathMetrics
  scrapingInterval: "5m"
  logName: testLogName
nbuserver:
  host: testServerNBUMaster
  port: "1556"
  scheme: "https"
  uri: testPathNetBackup
  apiKey: testKeyName
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
  uri: testPathMetrics
  scrapingInterval: "5m"
  logName: testLogName
nbuserver:
  host: testServerNBUMaster
  port: "1556"
  scheme: "https"
  uri: testPathNetBackup
  apiKey: testKeyName
opentelemetry:
  enabled: false
  endpoint: localhost:4317
  insecure: false
  samplingRate: 0.5
`,
			enabled:      false,
			endpoint:     testOTELEndpoint,
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
				assertOTelConfigFields(t, config, tt.enabled, tt.endpoint, tt.insecure, tt.samplingRate)
			}
		})
	}
}
