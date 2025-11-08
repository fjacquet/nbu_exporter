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
				json.NewEncoder(w).Encode(response)
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
				json.NewEncoder(w).Encode(errorResponse)
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
		t.Run(fmt.Sprintf("preserves API key: %s", apiKey[:min(len(apiKey), 20)]), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify Authorization header is exactly as provided
				if r.Header.Get(HeaderAuthorization) != apiKey {
					t.Errorf("Authorization header modified: got %v, want %v", r.Header.Get(HeaderAuthorization), apiKey)
				}

				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(mockAPIResponse{})
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
