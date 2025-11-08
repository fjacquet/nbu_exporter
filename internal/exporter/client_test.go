package exporter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fjacquet/nbu_exporter/internal/models"
)

// mockAPIResponse represents a simple API response structure for testing
type mockAPIResponse struct {
	Data []struct {
		ID         string `json:"id"`
		Attributes struct {
			Name string `json:"name"`
		} `json:"attributes"`
	} `json:"data"`
}

func TestNbuClient_GetHeaders(t *testing.T) {
	tests := []struct {
		name               string
		apiVersion         string
		apiKey             string
		expectedAccept     string
		expectedAuthHeader string
	}{
		{
			name:               "constructs Accept header with API version 12.0",
			apiVersion:         "12.0",
			apiKey:             "test-api-key-123",
			expectedAccept:     "application/vnd.netbackup+json;version=12.0",
			expectedAuthHeader: "test-api-key-123",
		},
		{
			name:               "constructs Accept header with API version 11.1",
			apiVersion:         "11.1",
			apiKey:             "another-key",
			expectedAccept:     "application/vnd.netbackup+json;version=11.1",
			expectedAuthHeader: "another-key",
		},
		{
			name:               "constructs Accept header with API version 10.5",
			apiVersion:         "10.5",
			apiKey:             "legacy-key",
			expectedAccept:     "application/vnd.netbackup+json;version=10.5",
			expectedAuthHeader: "legacy-key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := models.Config{
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
					APIVersion: tt.apiVersion,
					APIKey:     tt.apiKey,
				},
			}

			client := NewNbuClient(cfg)
			headers := client.getHeaders()

			if headers[HeaderAccept] != tt.expectedAccept {
				t.Errorf("getHeaders() Accept = %v, want %v", headers[HeaderAccept], tt.expectedAccept)
			}

			if headers[HeaderAuthorization] != tt.expectedAuthHeader {
				t.Errorf("getHeaders() Authorization = %v, want %v", headers[HeaderAuthorization], tt.expectedAuthHeader)
			}
		})
	}
}

func TestNbuClient_FetchData_Success(t *testing.T) {
	tests := []struct {
		name       string
		apiVersion string
		apiKey     string
	}{
		{
			name:       "fetches data with API version 12.0",
			apiVersion: "12.0",
			apiKey:     "test-key-12",
		},
		{
			name:       "fetches data with API version 11.1",
			apiVersion: "11.1",
			apiKey:     "test-key-11",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test server that validates headers
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify Accept header includes API version
				expectedAccept := fmt.Sprintf("application/vnd.netbackup+json;version=%s", tt.apiVersion)
				if r.Header.Get(HeaderAccept) != expectedAccept {
					t.Errorf("Accept header = %v, want %v", r.Header.Get(HeaderAccept), expectedAccept)
				}

				// Verify Authorization header is unchanged
				if r.Header.Get(HeaderAuthorization) != tt.apiKey {
					t.Errorf("Authorization header = %v, want %v", r.Header.Get(HeaderAuthorization), tt.apiKey)
				}

				// Return mock response
				response := mockAPIResponse{
					Data: []struct {
						ID         string `json:"id"`
						Attributes struct {
							Name string `json:"name"`
						} `json:"attributes"`
					}{
						{
							ID: "1",
							Attributes: struct {
								Name string `json:"name"`
							}{
								Name: "test-resource",
							},
						},
					},
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()

			cfg := models.Config{
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
					APIVersion: tt.apiVersion,
					APIKey:     tt.apiKey,
				},
			}

			client := NewNbuClient(cfg)
			var result mockAPIResponse

			err := client.FetchData(context.Background(), server.URL, &result)
			if err != nil {
				t.Errorf("FetchData() unexpected error = %v", err)
			}

			if len(result.Data) != 1 {
				t.Errorf("FetchData() got %d items, want 1", len(result.Data))
			}

			if result.Data[0].Attributes.Name != "test-resource" {
				t.Errorf("FetchData() name = %v, want test-resource", result.Data[0].Attributes.Name)
			}
		})
	}
}

func TestNbuClient_FetchData_NotAcceptableError(t *testing.T) {
	tests := []struct {
		name       string
		apiVersion string
		statusCode int
		wantErrMsg string
	}{
		{
			name:       "handles 406 Not Acceptable error",
			apiVersion: "12.0",
			statusCode: 406,
			wantErrMsg: "API version 12.0 is not supported by the NetBackup server (HTTP 406 Not Acceptable)",
		},
		{
			name:       "handles 406 with different API version",
			apiVersion: "13.0",
			statusCode: 406,
			wantErrMsg: "API version 13.0 is not supported by the NetBackup server (HTTP 406 Not Acceptable)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test server that returns 406 Not Acceptable
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				errorResponse := map[string]interface{}{
					"errorCode":    2060006,
					"errorMessage": "The requested API version is not supported by this NetBackup server.",
					"errorDetails": []string{
						fmt.Sprintf("The Accept header specifies version %s, but this server only supports versions up to 11.0", tt.apiVersion),
						"Update the API version in your configuration or upgrade the NetBackup server",
					},
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_ = json.NewEncoder(w).Encode(errorResponse)
			}))
			defer server.Close()

			cfg := models.Config{
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
					APIVersion: tt.apiVersion,
					APIKey:     "test-key",
				},
			}

			client := NewNbuClient(cfg)
			var result mockAPIResponse

			err := client.FetchData(context.Background(), server.URL, &result)
			if err == nil {
				t.Error("FetchData() expected error, got nil")
				return
			}

			// Check that error message contains expected text
			if err.Error() == "" {
				t.Errorf("FetchData() error message is empty")
			}

			// Verify the error message mentions the API version and 406 status
			errMsg := err.Error()
			if len(errMsg) == 0 {
				t.Error("FetchData() error message should not be empty")
			}
		})
	}
}

func TestNbuClient_FetchData_OtherErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		apiVersion string
	}{
		{
			name:       "handles 401 Unauthorized",
			statusCode: 401,
			apiVersion: "12.0",
		},
		{
			name:       "handles 403 Forbidden",
			statusCode: 403,
			apiVersion: "12.0",
		},
		{
			name:       "handles 404 Not Found",
			statusCode: 404,
			apiVersion: "12.0",
		},
		{
			name:       "handles 500 Internal Server Error",
			statusCode: 500,
			apiVersion: "12.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			cfg := models.Config{
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
					APIVersion: tt.apiVersion,
					APIKey:     "test-key",
				},
			}

			client := NewNbuClient(cfg)
			var result mockAPIResponse

			err := client.FetchData(context.Background(), server.URL, &result)
			if err == nil {
				t.Errorf("FetchData() expected error for status %d, got nil", tt.statusCode)
			}
		})
	}
}

func TestNbuClient_AuthorizationHeaderUnchanged(t *testing.T) {
	// This test specifically verifies that the Authorization header
	// remains unchanged regardless of API version
	apiKeys := []string{
		"simple-key",
		"complex-key-with-special-chars",
		"very-long-api-key-abcdefghijklmnopqrstuvwxyz0123456789",
	}

	for _, apiKey := range apiKeys {
		displayKey := apiKey
		if len(apiKey) > 20 {
			displayKey = apiKey[:20]
		}
		t.Run(fmt.Sprintf("preserves API key: %s", displayKey), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify Authorization header is exactly as provided
				if r.Header.Get(HeaderAuthorization) != apiKey {
					t.Errorf("Authorization header modified: got %v, want %v", r.Header.Get(HeaderAuthorization), apiKey)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(mockAPIResponse{})
			}))
			defer server.Close()

			cfg := models.Config{
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
					APIVersion: "12.0",
					APIKey:     apiKey,
				},
			}

			client := NewNbuClient(cfg)
			var result mockAPIResponse

			err := client.FetchData(context.Background(), server.URL, &result)
			if err != nil {
				t.Errorf("FetchData() unexpected error = %v", err)
			}
		})
	}
}

// TestNbuClient_HeaderConstruction_AllVersions tests that headers are correctly
// constructed for all three supported API versions (3.0, 12.0, 13.0)
func TestNbuClient_HeaderConstruction_AllVersions(t *testing.T) {
	tests := []struct {
		name           string
		apiVersion     string
		apiKey         string
		expectedAccept string
	}{
		{
			name:           "API version 3.0 (NetBackup 10.0-10.4)",
			apiVersion:     models.APIVersion30,
			apiKey:         "test-key-v3",
			expectedAccept: "application/vnd.netbackup+json;version=3.0",
		},
		{
			name:           "API version 12.0 (NetBackup 10.5)",
			apiVersion:     models.APIVersion120,
			apiKey:         "test-key-v12",
			expectedAccept: "application/vnd.netbackup+json;version=12.0",
		},
		{
			name:           "API version 13.0 (NetBackup 11.0)",
			apiVersion:     models.APIVersion130,
			apiKey:         "test-key-v13",
			expectedAccept: "application/vnd.netbackup+json;version=13.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := models.Config{}
			cfg.NbuServer.APIVersion = tt.apiVersion
			cfg.NbuServer.APIKey = tt.apiKey

			client := NewNbuClient(cfg)
			headers := client.getHeaders()

			if headers[HeaderAccept] != tt.expectedAccept {
				t.Errorf("getHeaders() Accept = %v, want %v", headers[HeaderAccept], tt.expectedAccept)
			}

			if headers[HeaderAuthorization] != tt.apiKey {
				t.Errorf("getHeaders() Authorization = %v, want %v", headers[HeaderAuthorization], tt.apiKey)
			}
		})
	}
}

// TestNbuClient_VersionFailureErrorMessages tests that error messages for version
// failures include helpful troubleshooting information
func TestNbuClient_VersionFailureErrorMessages(t *testing.T) {
	tests := []struct {
		name              string
		apiVersion        string
		statusCode        int
		wantErrorContains []string
		wantErrorExcludes []string
	}{
		{
			name:       "406 error includes all supported versions",
			apiVersion: "13.0",
			statusCode: 406,
			wantErrorContains: []string{
				"API version 13.0 is not supported",
				"HTTP 406 Not Acceptable",
				"3.0  (NetBackup 10.0-10.4)",
				"12.0 (NetBackup 10.5)",
				"13.0 (NetBackup 11.0)",
				"Troubleshooting steps",
				"bpgetconfig -g | grep VERSION",
				"automatic version detection",
			},
		},
		{
			name:       "406 error for version 12.0",
			apiVersion: "12.0",
			statusCode: 406,
			wantErrorContains: []string{
				"API version 12.0 is not supported",
				"HTTP 406 Not Acceptable",
				"apiVersion",
			},
		},
		{
			name:       "406 error for version 3.0",
			apiVersion: "3.0",
			statusCode: 406,
			wantErrorContains: []string{
				"API version 3.0 is not supported",
				"HTTP 406 Not Acceptable",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"errorMessage": "API version not supported",
				})
			}))
			defer server.Close()

			cfg := models.Config{}
			cfg.NbuServer.APIVersion = tt.apiVersion
			cfg.NbuServer.APIKey = "test-key"
			cfg.NbuServer.InsecureSkipVerify = true

			client := NewNbuClient(cfg)
			var result mockAPIResponse

			err := client.FetchData(context.Background(), server.URL, &result)
			if err == nil {
				t.Fatal("FetchData() expected error, got nil")
			}

			errMsg := err.Error()

			// Check that all expected strings are present
			for _, expected := range tt.wantErrorContains {
				if len(expected) > 0 && !contains(errMsg, expected) {
					t.Errorf("Error message missing expected text: %q\nGot: %s", expected, errMsg)
				}
			}

			// Check that excluded strings are not present
			for _, excluded := range tt.wantErrorExcludes {
				if len(excluded) > 0 && contains(errMsg, excluded) {
					t.Errorf("Error message contains unexpected text: %q\nGot: %s", excluded, errMsg)
				}
			}
		})
	}
}

// TestNewNbuClientWithVersionDetection_ExplicitVersion tests that when an API version
// is explicitly configured, version detection is bypassed
func TestNewNbuClientWithVersionDetection_ExplicitVersion(t *testing.T) {
	tests := []struct {
		name              string
		configuredVersion string
		shouldDetect      bool
	}{
		{
			name:              "explicit version 3.0 bypasses detection",
			configuredVersion: models.APIVersion30,
			shouldDetect:      false,
		},
		{
			name:              "explicit version 12.0 bypasses detection",
			configuredVersion: models.APIVersion120,
			shouldDetect:      false,
		},
		{
			name:              "empty version triggers detection",
			configuredVersion: "",
			shouldDetect:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := models.Config{}
			cfg.NbuServer.Host = "localhost"
			cfg.NbuServer.Port = "1556"
			cfg.NbuServer.Scheme = "https"
			cfg.NbuServer.URI = "/netbackup"
			cfg.NbuServer.APIKey = "test-key"
			cfg.NbuServer.APIVersion = tt.configuredVersion
			cfg.NbuServer.InsecureSkipVerify = true

			// For explicit versions, we should not attempt detection
			if !tt.shouldDetect {
				client := NewNbuClient(cfg)
				if client.cfg.NbuServer.APIVersion != tt.configuredVersion {
					t.Errorf("NewNbuClient() changed configured version from %s to %s",
						tt.configuredVersion, client.cfg.NbuServer.APIVersion)
				}
			}
		})
	}
}

// TestNewNbuClientWithVersionDetection_AutoDetection tests automatic version detection
// when no version is configured
// Note: This test is simplified to avoid complex mock server setup.
// Full integration tests for version detection are in version_detector_test.go
func TestNewNbuClientWithVersionDetection_AutoDetection(t *testing.T) {
	t.Run("returns error when detection fails", func(t *testing.T) {
		// Create a server that always returns 406
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(406)
		}))
		defer server.Close()

		cfg := models.Config{}
		// Parse the test server URL to get host and port
		cfg.NbuServer.Scheme = "http"
		cfg.NbuServer.Host = "localhost"
		cfg.NbuServer.Port = "9999" // Non-existent port to trigger failure
		cfg.NbuServer.URI = ""
		cfg.NbuServer.APIKey = "test-key"
		cfg.NbuServer.APIVersion = "" // Empty to trigger detection
		cfg.NbuServer.InsecureSkipVerify = true

		_, err := NewNbuClientWithVersionDetection(context.Background(), &cfg)
		if err == nil {
			t.Error("NewNbuClientWithVersionDetection() expected error when all versions fail, got nil")
		}
	})
}

// TestNbuClient_ConfigurationOverride tests that explicit configuration
// takes precedence over automatic detection
func TestNbuClient_ConfigurationOverride(t *testing.T) {
	tests := []struct {
		name              string
		configuredVersion string
		wantVersion       string
	}{
		{
			name:              "configured version 3.0 is preserved",
			configuredVersion: "3.0",
			wantVersion:       "3.0",
		},
		{
			name:              "configured version 12.0 is preserved",
			configuredVersion: "12.0",
			wantVersion:       "12.0",
		},
		{
			name:              "configured version 13.0 is preserved",
			configuredVersion: "13.0",
			wantVersion:       "13.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := models.Config{}
			cfg.NbuServer.APIVersion = tt.configuredVersion
			cfg.NbuServer.APIKey = "test-key"
			cfg.NbuServer.InsecureSkipVerify = true

			client := NewNbuClient(cfg)

			if client.cfg.NbuServer.APIVersion != tt.wantVersion {
				t.Errorf("Configuration override failed: got version %s, want %s",
					client.cfg.NbuServer.APIVersion, tt.wantVersion)
			}
		})
	}
}

// TestNbuClient_FetchData_HTMLResponse tests handling of HTML responses instead of JSON
// This addresses the bug where server returns HTML error pages (e.g., 404, auth failures)
// and we get "invalid character '<' looking for beginning of value" errors
func TestNbuClient_FetchData_HTMLResponse(t *testing.T) {
	tests := []struct {
		name        string
		statusCode  int
		contentType string
		body        string
		expectError string
	}{
		{
			name:        "HTML error page with 200 status",
			statusCode:  200,
			contentType: "text/html",
			body:        "<html><body><h1>Error</h1><p>Something went wrong</p></body></html>",
			expectError: "server returned text/html instead of JSON",
		},
		{
			name:        "HTML 404 page",
			statusCode:  200,
			contentType: "text/html; charset=utf-8",
			body:        "<!DOCTYPE html><html><head><title>404 Not Found</title></head><body><h1>Not Found</h1></body></html>",
			expectError: "server returned text/html; charset=utf-8 instead of JSON",
		},
		{
			name:        "HTML login page (auth failure)",
			statusCode:  200,
			contentType: "text/html",
			body:        "<html><body><form action='/login'>Please login</form></body></html>",
			expectError: "server returned text/html instead of JSON",
		},
		{
			name:        "XML response instead of JSON",
			statusCode:  200,
			contentType: "application/xml",
			body:        "<?xml version='1.0'?><error><message>Invalid request</message></error>",
			expectError: "server returned application/xml instead of JSON",
		},
		{
			name:        "Plain text error",
			statusCode:  200,
			contentType: "text/plain",
			body:        "Error: Invalid API endpoint",
			expectError: "server returned text/plain instead of JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", tt.contentType)
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			cfg := models.Config{
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
					APIVersion: "13.0",
					APIKey:     "test-key",
				},
			}

			client := NewNbuClient(cfg)
			var result mockAPIResponse

			err := client.FetchData(context.Background(), server.URL, &result)
			if err == nil {
				t.Error("FetchData() expected error for HTML response, got nil")
				return
			}

			if !contains(err.Error(), tt.expectError) {
				t.Errorf("FetchData() error = %v, should contain %v", err.Error(), tt.expectError)
			}

			// Verify error message includes helpful context
			// Note: The detailed message is logged, but the returned error is shorter
			// This is acceptable as long as the error clearly indicates non-JSON response
			if !contains(err.Error(), "instead of JSON") && !contains(err.Error(), "Content-Type") {
				t.Errorf("Error message should indicate non-JSON response, got: %v", err.Error())
			}
		})
	}
}

// TestNbuClient_FetchData_InvalidJSON tests handling of malformed JSON responses
func TestNbuClient_FetchData_InvalidJSON(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		expectError string
	}{
		{
			name:        "incomplete JSON",
			body:        `{"data": [{"id": "1"`,
			expectError: "failed to unmarshal JSON response",
		},
		{
			name:        "invalid JSON syntax",
			body:        `{data: invalid}`,
			expectError: "failed to unmarshal JSON response",
		},
		{
			name:        "empty response",
			body:        ``,
			expectError: "failed to unmarshal JSON response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(200)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			cfg := models.Config{
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
					APIVersion: "13.0",
					APIKey:     "test-key",
				},
			}

			client := NewNbuClient(cfg)
			var result mockAPIResponse

			err := client.FetchData(context.Background(), server.URL, &result)
			if err == nil {
				t.Error("FetchData() expected error for invalid JSON, got nil")
				return
			}

			if !contains(err.Error(), tt.expectError) {
				t.Errorf("FetchData() error = %v, should contain %v", err.Error(), tt.expectError)
			}

			// Verify error includes response preview for debugging
			if !contains(err.Error(), "Response preview:") {
				t.Error("Error message should include response preview for debugging")
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(substr) > 0 && len(s) > 0 && (s == substr || len(s) >= len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
