package models

import (
	"testing"

	"gopkg.in/yaml.v2"
)

func TestConfig_SetDefaults(t *testing.T) {
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

func TestConfig_Validate_APIVersion(t *testing.T) {
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

func TestConfig_ParseYAML_APIVersion(t *testing.T) {
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

func TestConfig_BackwardCompatibility(t *testing.T) {
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

func TestConfig_Validate_SupportedVersions(t *testing.T) {
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
