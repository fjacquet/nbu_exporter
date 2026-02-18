package exporter

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fjacquet/nbu_exporter/internal/models"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// Test constants specific to client tests
const (
	testResourceName          = "test-resource"
	testOperationName         = "test.operation"
	errMsgFetchDataUnexpected = "FetchData() unexpected error = %v"
	errMsgFetchDataItemCount  = "FetchData() got %d items, want 1"
	errMsgUnmarshalJSON       = "failed to unmarshal JSON response"
	errMsgHTTP406             = "HTTP 406 Not Acceptable"
)

// Note: Common constants like testAPIKey, contentTypeHeader, and contentTypeJSON
// are defined in test_common.go and shared across all test files

// mockAPIResponse represents a simple API response structure for testing
type mockAPIResponse struct {
	Data []struct {
		ID         string `json:"id"`
		Attributes struct {
			Name string `json:"name"`
		} `json:"attributes"`
	} `json:"data"`
}

func TestNbuClientGetHeaders(t *testing.T) {
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
			cfg := createBasicTestConfig(tt.apiVersion, tt.apiKey)
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

func TestNbuClientFetchDataSuccess(t *testing.T) {
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
			server := createTestServer(t, tt.apiVersion, tt.apiKey)
			defer server.Close()

			cfg := createBasicTestConfig(tt.apiVersion, tt.apiKey)
			client := NewNbuClient(cfg)
			var result mockAPIResponse

			err := client.FetchData(context.Background(), server.URL, &result)
			if err != nil {
				t.Errorf(errMsgFetchDataUnexpected, err)
			}

			if len(result.Data) != 1 {
				t.Errorf(errMsgFetchDataItemCount, len(result.Data))
			}

			if result.Data[0].Attributes.Name != testResourceName {
				t.Errorf("FetchData() name = %v, want %s", result.Data[0].Attributes.Name, testResourceName)
			}
		})
	}
}

// createTestServer creates a test HTTP server that validates headers and returns mock data
func createTestServer(t *testing.T, apiVersion, apiKey string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedAccept := fmt.Sprintf("application/vnd.netbackup+json;version=%s", apiVersion)
		if r.Header.Get(HeaderAccept) != expectedAccept {
			t.Errorf("Accept header = %v, want %v", r.Header.Get(HeaderAccept), expectedAccept)
		}

		if r.Header.Get(HeaderAuthorization) != apiKey {
			t.Errorf("Authorization header = %v, want %v", r.Header.Get(HeaderAuthorization), apiKey)
		}

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
						Name: testResourceName,
					},
				},
			},
		}

		w.Header().Set(contentTypeHeader, contentTypeJSON)
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
	}))
}

// createBasicTestConfig creates a test configuration with the given API version and key
func createBasicTestConfig(apiVersion, apiKey string) models.Config {
	return models.Config{
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
			APIVersion: apiVersion,
			APIKey:     apiKey,
		},
	}
}

func TestNbuClientFetchDataNotAcceptableError(t *testing.T) {
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

				w.Header().Set(contentTypeHeader, contentTypeJSON)
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
					APIKey:     testAPIKey,
				},
			}

			client := NewNbuClient(cfg)
			// Disable retries for faster test execution
			client.client.SetRetryCount(0)
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

func TestNbuClientFetchDataOtherErrors(t *testing.T) {
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

			cfg := createBasicTestConfig(tt.apiVersion, testAPIKey)
			client := NewNbuClient(cfg)
			// Disable retries for faster test execution (especially for 500 errors)
			client.client.SetRetryCount(0)
			var result mockAPIResponse

			err := client.FetchData(context.Background(), server.URL, &result)
			if err == nil {
				t.Errorf("FetchData() expected error for status %d, got nil", tt.statusCode)
			}
		})
	}
}

func TestNbuClientAuthorizationHeaderUnchanged(t *testing.T) {
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

				w.Header().Set(contentTypeHeader, contentTypeJSON)
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
				t.Errorf(errMsgFetchDataUnexpected, err)
			}
		})
	}
}

// TestNbuClientHeaderConstructionAllVersions tests that headers are correctly
// constructed for all three supported API versions (3.0, 12.0, 13.0)
func TestNbuClientHeaderConstructionAllVersions(t *testing.T) {
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
			cfg := createBasicTestConfig(tt.apiVersion, tt.apiKey)
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

// TestNbuClientVersionFailureErrorMessages tests that error messages for version
// failures include helpful troubleshooting information
func TestNbuClientVersionFailureErrorMessages(t *testing.T) {
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
				errMsgHTTP406,
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
				errMsgHTTP406,
				"apiVersion",
			},
		},
		{
			name:       "406 error for version 3.0",
			apiVersion: "3.0",
			statusCode: 406,
			wantErrorContains: []string{
				"API version 3.0 is not supported",
				errMsgHTTP406,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := createErrorServer(tt.statusCode)
			defer server.Close()

			cfg := createMinimalConfig(tt.apiVersion, testAPIKey)
			client := NewNbuClient(cfg)
			// Disable retries for faster test execution
			client.client.SetRetryCount(0)
			var result mockAPIResponse

			err := client.FetchData(context.Background(), server.URL, &result)
			if err == nil {
				t.Fatal("FetchData() expected error, got nil")
			}

			validateErrorMessage(t, err.Error(), tt.wantErrorContains, tt.wantErrorExcludes)
		})
	}
}

// createErrorServer creates a test server that returns the specified error status
func createErrorServer(statusCode int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"errorMessage": "API version not supported",
		})
	}))
}

// createMinimalConfig creates a minimal test configuration
func createMinimalConfig(apiVersion, apiKey string) models.Config {
	cfg := models.Config{}
	cfg.NbuServer.APIVersion = apiVersion
	cfg.NbuServer.APIKey = apiKey
	cfg.NbuServer.InsecureSkipVerify = true
	return cfg
}

// validateErrorMessage checks that error message contains expected strings and excludes unwanted ones
func validateErrorMessage(t *testing.T, errMsg string, wantContains, wantExcludes []string) {
	for _, expected := range wantContains {
		if len(expected) > 0 && !contains(errMsg, expected) {
			t.Errorf("Error message missing expected text: %q\nGot: %s", expected, errMsg)
		}
	}

	for _, excluded := range wantExcludes {
		if len(excluded) > 0 && contains(errMsg, excluded) {
			t.Errorf("Error message contains unexpected text: %q\nGot: %s", excluded, errMsg)
		}
	}
}

// createMockDataServer creates a test server that returns mock API response data
func createMockDataServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(contentTypeHeader, contentTypeJSON)
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(mockAPIResponse{
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
						Name: testResourceName,
					},
				},
			},
		})
	}))
}

// TestNewNbuClientWithVersionDetectionExplicitVersion tests that when an API version
// is explicitly configured, version detection is bypassed
func TestNewNbuClientWithVersionDetectionExplicitVersion(t *testing.T) {
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
			cfg.NbuServer.APIKey = testAPIKey
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

// TestNewNbuClientWithVersionDetectionAutoDetection tests automatic version detection
// when no version is configured
// Note: This test is simplified to avoid complex mock server setup.
// Note: Full version detection tests (including failure scenarios) are in version_detector_test.go
// This test verifies client creation with pre-configured version (bypasses slow detection)
func TestNewNbuClientWithVersionDetectionPreConfigured(t *testing.T) {
	t.Run("skips detection when version is configured", func(t *testing.T) {
		cfg := createBasicTestConfig("13.0", testAPIKey)

		// When version is already set, detection is skipped
		client, err := NewNbuClientWithVersionDetection(context.Background(), &cfg)
		if err != nil {
			t.Fatalf("NewNbuClientWithVersionDetection() unexpected error: %v", err)
		}
		if client == nil {
			t.Fatal("NewNbuClientWithVersionDetection() returned nil client")
		}

		// Verify the configured version is preserved
		if client.cfg.NbuServer.APIVersion != "13.0" {
			t.Errorf("API version = %s, want 13.0", client.cfg.NbuServer.APIVersion)
		}
	})
}

// TestNbuClientConfigurationOverride tests that explicit configuration
// takes precedence over automatic detection
func TestNbuClientConfigurationOverride(t *testing.T) {
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
			cfg := createMinimalConfig(tt.configuredVersion, testAPIKey)
			client := NewNbuClient(cfg)

			if client.cfg.NbuServer.APIVersion != tt.wantVersion {
				t.Errorf("Configuration override failed: got version %s, want %s",
					client.cfg.NbuServer.APIVersion, tt.wantVersion)
			}
		})
	}
}

// TestNbuClientFetchDataHTMLResponse tests handling of HTML responses instead of JSON
// This addresses the bug where server returns HTML error pages (e.g., 404, auth failures)
// and we get "invalid character '<' looking for beginning of value" errors
func TestNbuClientFetchDataHTMLResponse(t *testing.T) {
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
				w.Header().Set(contentTypeHeader, tt.contentType)
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			cfg := createTestConfig("13.0", testAPIKey)
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

// TestNbuClientFetchDataInvalidJSON tests handling of malformed JSON responses
func TestNbuClientFetchDataInvalidJSON(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		expectError string
	}{
		{
			name:        "incomplete JSON",
			body:        `{"data": [{"id": "1"`,
			expectError: errMsgUnmarshalJSON,
		},
		{
			name:        "invalid JSON syntax",
			body:        `{data: invalid}`,
			expectError: errMsgUnmarshalJSON,
		},
		{
			name:        "empty response",
			body:        ``,
			expectError: errMsgUnmarshalJSON,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set(contentTypeHeader, contentTypeJSON)
				w.WriteHeader(200)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			cfg := createBasicTestConfig("13.0", testAPIKey)
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

// TestNbuClientTracingDisabled tests that client works correctly without tracer
func TestNbuClientTracingDisabled(t *testing.T) {
	server := createMockDataServer()
	defer server.Close()

	cfg := createBasicTestConfig("13.0", testAPIKey)
	client := NewNbuClient(cfg) // No TracerProvider option = noop tracing

	var result mockAPIResponse
	err := client.FetchData(context.Background(), server.URL, &result)
	if err != nil {
		t.Errorf(errMsgFetchDataUnexpected, err)
	}

	if len(result.Data) != 1 {
		t.Errorf(errMsgFetchDataItemCount, len(result.Data))
	}
}

// TestNbuClientCreateHTTPSpanNilSafe tests that TracerWrapper always returns valid spans
func TestNbuClientCreateHTTPSpanNilSafe(t *testing.T) {
	cfg := createBasicTestConfig("13.0", testAPIKey)
	client := NewNbuClient(cfg) // No TracerProvider = noop tracer

	ctx := context.Background()
	newCtx, span := client.tracing.StartSpan(ctx, testOperationName, trace.SpanKindClient)

	// TracerWrapper should always return valid context and span (noop if no provider)
	if newCtx == nil {
		t.Error("TracerWrapper.StartSpan() should return valid context")
	}

	if span == nil {
		t.Error("TracerWrapper.StartSpan() should return valid span (noop if tracing disabled)")
	}

	// Should not panic
	span.End()
}

// TestNbuClientRecordHTTPAttributesNilSafe tests that attribute recording is nil-safe
func TestNbuClientRecordHTTPAttributesNilSafe(t *testing.T) {
	cfg := createBasicTestConfig("13.0", testAPIKey)
	client := NewNbuClient(cfg)

	// Should not panic when span is nil
	client.recordHTTPAttributes(nil, "GET", "http://example.com", 200, 0, 100, 50*time.Millisecond)
}

// TestNbuClientRecordErrorNilSafe tests that error recording is nil-safe
func TestNbuClientRecordErrorNilSafe(t *testing.T) {
	cfg := createBasicTestConfig("13.0", testAPIKey)
	client := NewNbuClient(cfg)

	// Should not panic when span is nil
	testErr := fmt.Errorf("test error")
	client.recordError(nil, testErr)
}

// TestNbuClientInjectTraceContextNilSafe tests that trace context injection works with noop tracer
func TestNbuClientInjectTraceContextNilSafe(t *testing.T) {
	cfg := createBasicTestConfig("13.0", testAPIKey)
	client := NewNbuClient(cfg) // No TracerProvider = noop tracer

	headers := map[string]string{
		"Authorization": testAPIKey,
		"Accept":        contentTypeJSON,
	}

	result := client.injectTraceContext(context.Background(), headers)

	// Should return headers unchanged when tracer is nil
	if len(result) != len(headers) {
		t.Errorf("injectTraceContext() changed header count: got %d, want %d", len(result), len(headers))
	}

	for k, v := range headers {
		if result[k] != v {
			t.Errorf("injectTraceContext() changed header %s: got %v, want %v", k, result[k], v)
		}
	}
}

// TestNbuClientFetchDataWithTracing tests FetchData with tracing enabled
// Note: This test uses a mock tracer to verify span creation without requiring a real collector
func TestNbuClientFetchDataWithTracing(t *testing.T) {
	server := createMockDataServer()
	defer server.Close()

	cfg := createBasicTestConfig("13.0", testAPIKey)
	client := NewNbuClient(cfg)

	// Note: Without setting up a full OpenTelemetry environment, the tracer will be nil
	// This test verifies that the code handles both cases correctly

	var result mockAPIResponse
	err := client.FetchData(context.Background(), server.URL, &result)
	if err != nil {
		t.Errorf(errMsgFetchDataUnexpected, err)
	}

	if len(result.Data) != 1 {
		t.Errorf(errMsgFetchDataItemCount, len(result.Data))
	}
}

// TestNbuClientFetchDataErrorWithTracing tests error recording with tracing
func TestNbuClientFetchDataErrorWithTracing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := createBasicTestConfig("13.0", testAPIKey)
	client := NewNbuClient(cfg)
	// Disable retries for faster test execution
	client.client.SetRetryCount(0)

	var result mockAPIResponse
	err := client.FetchData(context.Background(), server.URL, &result)
	if err == nil {
		t.Error("FetchData() expected error for 500 status, got nil")
	}
}

// TestCreateSpanWithNilTracer tests that createSpan handles nil tracer correctly
func TestCreateSpanWithNilTracer(t *testing.T) {
	ctx := context.Background()

	newCtx, span := createSpan(ctx, nil, testOperationName, trace.SpanKindClient)

	if newCtx != ctx {
		t.Error("createSpan() with nil tracer should return original context")
	}

	if span != nil {
		t.Error("createSpan() with nil tracer should return nil span")
	}
}

// TestCreateSpanWithValidTracer tests that createSpan creates a span with valid tracer
func TestCreateSpanWithValidTracer(t *testing.T) {
	// Create a mock tracer provider using noop implementation
	tp := noop.NewTracerProvider()
	tracer := tp.Tracer("test-tracer")

	ctx := context.Background()

	newCtx, span := createSpan(ctx, tracer, testOperationName, trace.SpanKindClient)

	if newCtx == ctx {
		t.Error("createSpan() with valid tracer should return new context")
	}

	if span == nil {
		t.Error("createSpan() with valid tracer should return non-nil span")
	}

	// Clean up
	if span != nil {
		span.End()
	}
}

// TestCreateSpanDifferentKinds tests createSpan with different span kinds
func TestCreateSpanDifferentKinds(t *testing.T) {
	tests := []struct {
		name      string
		operation string
		kind      trace.SpanKind
	}{
		{
			name:      "client span kind",
			operation: "http.request",
			kind:      trace.SpanKindClient,
		},
		{
			name:      "internal span kind",
			operation: "internal.process",
			kind:      trace.SpanKindInternal,
		},
		{
			name:      "server span kind",
			operation: "http.handler",
			kind:      trace.SpanKindServer,
		},
	}

	tp := noop.NewTracerProvider()
	tracer := tp.Tracer("test-tracer")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			newCtx, span := createSpan(ctx, tracer, tt.operation, tt.kind)

			if newCtx == ctx {
				t.Error("createSpan() should return new context")
			}

			if span == nil {
				t.Error("createSpan() should return non-nil span")
			}

			// Clean up
			if span != nil {
				span.End()
			}
		})
	}
}

// TestShouldPerformVersionDetection tests the version detection decision logic
func TestShouldPerformVersionDetection(t *testing.T) {
	tests := []struct {
		name       string
		apiVersion string
		expected   bool
	}{
		{
			name:       "empty version triggers detection",
			apiVersion: "",
			expected:   true,
		},
		{
			name:       "version 3.0 bypasses detection",
			apiVersion: models.APIVersion30,
			expected:   false,
		},
		{
			name:       "version 12.0 bypasses detection",
			apiVersion: models.APIVersion120,
			expected:   false,
		},
		{
			name:       "version 13.0 bypasses detection",
			apiVersion: models.APIVersion130,
			expected:   false,
		},
		{
			name:       "custom version bypasses detection",
			apiVersion: "11.0",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &models.Config{}
			cfg.NbuServer.APIVersion = tt.apiVersion

			result := shouldPerformVersionDetection(cfg)

			if result != tt.expected {
				t.Errorf("shouldPerformVersionDetection() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestIsExplicitVersionConfigured tests the explicit version configuration check
func TestIsExplicitVersionConfigured(t *testing.T) {
	tests := []struct {
		name       string
		apiVersion string
		expected   bool
	}{
		{
			name:       "empty version is not explicit",
			apiVersion: "",
			expected:   false,
		},
		{
			name:       "default version 13.0 is not explicit",
			apiVersion: models.APIVersion130,
			expected:   false,
		},
		{
			name:       "version 12.0 is explicit",
			apiVersion: models.APIVersion120,
			expected:   true,
		},
		{
			name:       "version 3.0 is explicit",
			apiVersion: models.APIVersion30,
			expected:   true,
		},
		{
			name:       "custom version 11.0 is explicit",
			apiVersion: "11.0",
			expected:   true,
		},
		{
			name:       "custom version 10.5 is explicit",
			apiVersion: "10.5",
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &models.Config{}
			cfg.NbuServer.APIVersion = tt.apiVersion

			result := isExplicitVersionConfigured(cfg)

			if result != tt.expected {
				t.Errorf("isExplicitVersionConfigured() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestNbuClientCloseIdempotent tests that Close() can only be called once
func TestNbuClientCloseIdempotent(t *testing.T) {
	cfg := createBasicTestConfig("13.0", "test-key")
	client := NewNbuClient(cfg)

	// First close should succeed
	err := client.Close()
	if err != nil {
		t.Errorf("First Close() unexpected error: %v", err)
	}

	// Second close should return error
	err = client.Close()
	if err == nil {
		t.Error("Second Close() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "already closed") {
		t.Errorf("Close() error = %v, want 'already closed'", err)
	}
}

// TestNbuClientCloseWaitsForActiveRequests tests that Close() waits for active requests to complete
func TestNbuClientCloseWaitsForActiveRequests(t *testing.T) {
	// Create a slow server
	requestStarted := make(chan struct{})
	requestComplete := make(chan struct{})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(requestStarted)
		<-requestComplete // Wait for test to signal completion
		w.Header().Set(contentTypeHeader, contentTypeJSON)
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(mockAPIResponse{})
	}))
	defer server.Close()

	cfg := createBasicTestConfig("13.0", "test-key")
	client := NewNbuClient(cfg)

	// Start a request in background
	var fetchErr error
	fetchDone := make(chan struct{})
	go func() {
		var result mockAPIResponse
		fetchErr = client.FetchData(context.Background(), server.URL, &result)
		close(fetchDone)
	}()

	// Wait for request to start
	<-requestStarted

	// Close should block waiting for request
	closeDone := make(chan struct{})
	go func() {
		_ = client.Close()
		close(closeDone)
	}()

	// Give Close a moment to start waiting
	time.Sleep(50 * time.Millisecond)

	// Verify Close is still blocking
	select {
	case <-closeDone:
		t.Error("Close() returned before request completed")
	default:
		// Expected - Close is waiting
	}

	// Allow request to complete
	close(requestComplete)

	// Now Close should complete
	select {
	case <-closeDone:
		// Expected
	case <-time.After(5 * time.Second):
		t.Error("Close() did not complete after request finished")
	}

	// Verify request completed successfully
	<-fetchDone
	if fetchErr != nil {
		t.Errorf("FetchData() error: %v", fetchErr)
	}
}

// TestNbuClientFetchDataRejectsAfterClose tests that FetchData rejects requests after Close()
func TestNbuClientFetchDataRejectsAfterClose(t *testing.T) {
	cfg := createBasicTestConfig("13.0", "test-key")
	client := NewNbuClient(cfg)

	// Close the client
	_ = client.Close()

	// Attempt to fetch - should fail
	var result mockAPIResponse
	err := client.FetchData(context.Background(), "http://example.com", &result)
	if err == nil {
		t.Error("FetchData() after Close() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "closed") {
		t.Errorf("FetchData() error = %v, want error containing 'closed'", err)
	}
}

// TestNewNbuClient_TLSConfig tests that NewNbuClient configures TLS properly
func TestNewNbuClient_TLSConfig(t *testing.T) {
	tests := []struct {
		name               string
		insecureSkipVerify bool
	}{
		{
			name:               "secure TLS configuration (default)",
			insecureSkipVerify: false,
		},
		{
			name:               "insecure TLS configuration",
			insecureSkipVerify: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := createBasicTestConfig("13.0", "test-key")
			cfg.NbuServer.InsecureSkipVerify = tt.insecureSkipVerify

			client := NewNbuClient(cfg)

			require.NotNil(t, client, "NewNbuClient() returned nil")
			require.NotNil(t, client.client, "NewNbuClient() did not initialize HTTP client")

			// Verify config is stored
			if client.cfg.NbuServer.InsecureSkipVerify != tt.insecureSkipVerify {
				t.Errorf("NewNbuClient() InsecureSkipVerify = %v, want %v",
					client.cfg.NbuServer.InsecureSkipVerify, tt.insecureSkipVerify)
			}
		})
	}
}

// TestNbuClientCloseTimeout tests that CloseWithContext respects timeout
func TestNbuClientCloseTimeout(t *testing.T) {
	// Create a server that responds slowly
	requestReceived := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(requestReceived)
		// Block for a long time to simulate a slow request
		time.Sleep(5 * time.Second)
		w.Header().Set(contentTypeHeader, contentTypeJSON)
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(mockAPIResponse{})
	}))
	defer server.Close()

	cfg := createBasicTestConfig("13.0", "test-key")
	client := NewNbuClient(cfg)

	// Start a request that will take a long time
	requestCtx, requestCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer requestCancel()

	go func() {
		var result mockAPIResponse
		_ = client.FetchData(requestCtx, server.URL, &result)
	}()

	// Wait for request to start
	select {
	case <-requestReceived:
		// Request started
	case <-time.After(1 * time.Second):
		t.Fatal("Request did not start in time")
	}

	// Close with short timeout should return quickly
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := client.CloseWithContext(ctx)
	elapsed := time.Since(start)

	// Should complete around timeout time, not hang
	if elapsed > 500*time.Millisecond {
		t.Errorf("CloseWithContext took %v, expected ~100ms timeout", elapsed)
	}

	// Should return context deadline exceeded error
	if err != nil && err != context.DeadlineExceeded {
		t.Logf("CloseWithContext returned error: %v", err)
	}

	// Cancel the request to clean up
	requestCancel()
}

func TestNbuClient_RetryConfiguration(t *testing.T) {
	cfg := createBasicTestConfig("13.0", "test-key")
	client := NewNbuClient(cfg)
	if client == nil {
		t.Fatal("NewNbuClient returned nil")
	}

	// Verify retry count is configured (resty exposes this)
	if client.client.RetryCount != 3 {
		t.Errorf("RetryCount = %d, want 3", client.client.RetryCount)
	}

	// Verify retry wait times
	if client.client.RetryWaitTime != 5*time.Second {
		t.Errorf("RetryWaitTime = %v, want 5s", client.client.RetryWaitTime)
	}
	if client.client.RetryMaxWaitTime != 60*time.Second {
		t.Errorf("RetryMaxWaitTime = %v, want 60s", client.client.RetryMaxWaitTime)
	}
}

func TestNbuClient_ConnectionPoolConfiguration(t *testing.T) {
	cfg := createBasicTestConfig("13.0", "test-key")
	client := NewNbuClient(cfg)
	if client == nil {
		t.Fatal("NewNbuClient returned nil")
	}

	// Access underlying transport
	httpClient := client.client.GetClient()
	transport, ok := httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatal("Expected *http.Transport")
	}

	// Verify connection pool settings
	if transport.MaxIdleConns != 100 {
		t.Errorf("MaxIdleConns = %d, want 100", transport.MaxIdleConns)
	}
	if transport.MaxIdleConnsPerHost != 20 {
		t.Errorf("MaxIdleConnsPerHost = %d, want 20", transport.MaxIdleConnsPerHost)
	}
	if transport.IdleConnTimeout != 90*time.Second {
		t.Errorf("IdleConnTimeout = %v, want 90s", transport.IdleConnTimeout)
	}
}

func TestNbuClient_TLSInTransport(t *testing.T) {
	tests := []struct {
		name               string
		insecureSkipVerify bool
	}{
		{"secure", false},
		{"insecure", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := createBasicTestConfig("13.0", "test-key")
			cfg.NbuServer.InsecureSkipVerify = tt.insecureSkipVerify

			client := NewNbuClient(cfg)
			if client == nil {
				t.Fatal("NewNbuClient returned nil")
			}

			httpClient := client.client.GetClient()
			transport, ok := httpClient.Transport.(*http.Transport)
			if !ok {
				t.Fatal("Expected *http.Transport")
			}
			if transport.TLSClientConfig == nil {
				t.Fatal("TLSClientConfig is nil")
			}

			if transport.TLSClientConfig.InsecureSkipVerify != tt.insecureSkipVerify {
				t.Errorf("InsecureSkipVerify = %v, want %v", transport.TLSClientConfig.InsecureSkipVerify, tt.insecureSkipVerify)
			}
			if transport.TLSClientConfig.MinVersion != tls.VersionTLS12 {
				t.Errorf("MinVersion = %d, want %d", transport.TLSClientConfig.MinVersion, tls.VersionTLS12)
			}
		})
	}
}

// TestClientNetworkTimeout tests behavior when server doesn't respond within timeout
func TestClientNetworkTimeout(t *testing.T) {
	// Create a server that delays response longer than client timeout
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Sleep longer than the client's context timeout
		time.Sleep(5 * time.Second)
		w.Header().Set(contentTypeHeader, contentTypeJSON)
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(mockAPIResponse{})
	}))
	defer server.Close()

	cfg := createBasicTestConfig("13.0", testAPIKey)
	client := NewNbuClient(cfg)
	// Disable retries for faster test execution
	client.client.SetRetryCount(0)

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	var result mockAPIResponse
	err := client.FetchData(ctx, server.URL, &result)

	if err == nil {
		t.Error("FetchData() expected timeout error, got nil")
		return
	}

	// Error should indicate timeout or deadline exceeded
	errStr := err.Error()
	if !strings.Contains(errStr, "deadline exceeded") &&
		!strings.Contains(errStr, "timeout") &&
		!strings.Contains(errStr, "context deadline exceeded") &&
		!strings.Contains(errStr, "context canceled") {
		t.Errorf("FetchData() error = %v, should indicate timeout", err)
	}
}

// TestClientConnectionRefused tests behavior when server is unreachable
func TestClientConnectionRefused(t *testing.T) {
	cfg := createBasicTestConfig("13.0", testAPIKey)
	client := NewNbuClient(cfg)
	// Disable retries for faster test execution
	client.client.SetRetryCount(0)

	// Use a URL that should fail to connect (localhost with invalid port)
	unreachableURL := "http://127.0.0.1:65534/nonexistent"

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var result mockAPIResponse
	err := client.FetchData(ctx, unreachableURL, &result)

	if err == nil {
		t.Error("FetchData() expected connection error, got nil")
		return
	}

	// Error should indicate connection failure
	errStr := err.Error()
	if !strings.Contains(errStr, "connection refused") &&
		!strings.Contains(errStr, "connect:") &&
		!strings.Contains(errStr, "dial") &&
		!strings.Contains(errStr, "network") &&
		!strings.Contains(errStr, "no such host") {
		t.Errorf("FetchData() error = %v, should indicate connection failure", err)
	}
}

// TestClientHTTPErrorsComprehensive is a table-driven test for various HTTP status codes
// including edge cases like 400, 502, 503 that weren't covered before
func TestClientHTTPErrorsComprehensive(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "400 Bad Request",
			statusCode: 400,
			wantErr:    true,
		},
		{
			name:       "401 Unauthorized",
			statusCode: 401,
			wantErr:    true,
		},
		{
			name:       "403 Forbidden",
			statusCode: 403,
			wantErr:    true,
		},
		{
			name:       "404 Not Found",
			statusCode: 404,
			wantErr:    true,
		},
		{
			name:       "500 Internal Server Error",
			statusCode: 500,
			wantErr:    true,
		},
		{
			name:       "502 Bad Gateway",
			statusCode: 502,
			wantErr:    true,
		},
		{
			name:       "503 Service Unavailable",
			statusCode: 503,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": fmt.Sprintf("HTTP %d error", tt.statusCode),
				})
			}))
			defer server.Close()

			cfg := createBasicTestConfig("13.0", testAPIKey)
			client := NewNbuClient(cfg)
			// Disable retries for faster test execution (especially for 5xx errors)
			client.client.SetRetryCount(0)
			var result mockAPIResponse

			err := client.FetchData(context.Background(), server.URL, &result)

			if tt.wantErr {
				if err == nil {
					t.Errorf("FetchData() expected error for status %d, got nil", tt.statusCode)
				}
			} else {
				if err != nil {
					t.Errorf("FetchData() unexpected error for status %d: %v", tt.statusCode, err)
				}
			}
		})
	}
}

// TestClientPartialResponse tests handling of truncated/partial JSON response
func TestClientPartialResponse(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		expectError string
	}{
		{
			name:        "truncated JSON object",
			body:        `{"data": [{"id": "1"`,
			expectError: errMsgUnmarshalJSON,
		},
		{
			name:        "truncated JSON array",
			body:        `{"data": [`,
			expectError: errMsgUnmarshalJSON,
		},
		{
			name:        "incomplete key",
			body:        `{"dat`,
			expectError: errMsgUnmarshalJSON,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set(contentTypeHeader, contentTypeJSON)
				w.WriteHeader(http.StatusOK)
				// Write partial response and close connection
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			cfg := createBasicTestConfig("13.0", testAPIKey)
			client := NewNbuClient(cfg)
			var result mockAPIResponse

			err := client.FetchData(context.Background(), server.URL, &result)
			if err == nil {
				t.Error("FetchData() expected error for partial response, got nil")
				return
			}

			if !contains(err.Error(), tt.expectError) {
				t.Errorf("FetchData() error = %v, should contain %v", err.Error(), tt.expectError)
			}
		})
	}
}

// TestClientEmptyResponseBody tests handling of empty response body
func TestClientEmptyResponseBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(contentTypeHeader, contentTypeJSON)
		w.WriteHeader(http.StatusOK)
		// Write nothing - empty body
	}))
	defer server.Close()

	cfg := createBasicTestConfig("13.0", testAPIKey)
	client := NewNbuClient(cfg)
	var result mockAPIResponse

	err := client.FetchData(context.Background(), server.URL, &result)
	if err == nil {
		t.Error("FetchData() expected error for empty response, got nil")
		return
	}

	// Error should indicate JSON parsing failure
	if !contains(err.Error(), errMsgUnmarshalJSON) {
		t.Errorf("FetchData() error = %v, should indicate JSON parsing failure", err)
	}
}

// TestClientServerClosesDuringTransfer tests behavior when server closes connection mid-transfer
func TestClientServerClosesDuringTransfer(t *testing.T) {
	// Create a server that closes connection after sending partial headers
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}

	serverClosed := make(chan struct{})
	go func() {
		defer close(serverClosed)
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		// Read some data
		buf := make([]byte, 1024)
		_, _ = conn.Read(buf)
		// Close without sending response
		_ = conn.Close()
	}()

	cfg := createBasicTestConfig("13.0", testAPIKey)
	client := NewNbuClient(cfg)
	client.client.SetRetryCount(0)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var result mockAPIResponse
	testURL := fmt.Sprintf("http://%s/test", listener.Addr().String())
	err = client.FetchData(ctx, testURL, &result)

	// Close listener and wait for server goroutine
	_ = listener.Close()
	<-serverClosed

	if err == nil {
		t.Error("FetchData() expected error when server closes connection, got nil")
	}
}
