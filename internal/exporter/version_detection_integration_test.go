package exporter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fjacquet/nbu_exporter/internal/testutil"
)

// extractVersionFromAcceptHeader extracts the API version from the Accept header
func extractVersionFromAcceptHeader(acceptHeader string) string {
	if strings.Contains(acceptHeader, testutil.APIVersion130) {
		return "13.0"
	} else if strings.Contains(acceptHeader, testutil.APIVersion120) {
		return "12.0"
	} else if strings.Contains(acceptHeader, testutil.APIVersion30) {
		return "3.0"
	}
	return ""
}

// isVersionSupported checks if the requested version is in the list of supported versions
func isVersionSupported(requestedVersion string, supportedVersions []string) bool {
	for _, v := range supportedVersions {
		if v == requestedVersion {
			return true
		}
	}
	return false
}

// writeVersionResponse writes the appropriate HTTP response based on version support
func writeVersionResponse(w http.ResponseWriter, requestedVersion string, supported bool) {
	if supported {
		response := createMinimalJobsResponse()
		w.Header().Set(testutil.ContentTypeHeader, fmt.Sprintf("application/vnd.netbackup+json;version=%s", requestedVersion))
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
	} else {
		w.WriteHeader(http.StatusNotAcceptable)
		errorResponse := map[string]interface{}{
			"errorCode":    406,
			"errorMessage": fmt.Sprintf("API version %s is not supported", requestedVersion),
		}
		_ = json.NewEncoder(w).Encode(errorResponse)
	}
}

// createVersionDetectionMockServer creates a mock server for version detection testing
func createVersionDetectionMockServer(supportedVersions []string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedVersion := extractVersionFromAcceptHeader(r.Header.Get("Accept"))
		supported := isVersionSupported(requestedVersion, supportedVersions)
		writeVersionResponse(w, requestedVersion, supported)
	}))
}

// validateVersionDetectionResult validates the result of version detection
func validateVersionDetectionResult(t *testing.T, detectedVersion string, err error, expectedVersion string, expectError bool) {
	if expectError {
		if err == nil {
			t.Error("Expected error but got none")
		}
	} else {
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if detectedVersion != expectedVersion {
			t.Errorf("Expected version %s, got %s", expectedVersion, detectedVersion)
		}
	}
}

// TestVersionDetectionWithMockServers tests automatic version detection with mock servers
func TestVersionDetectionWithMockServers(t *testing.T) {
	tests := []struct {
		name              string
		supportedVersions []string // Versions the mock server will accept
		expectedVersion   string   // Version we expect to be detected
		expectError       bool
	}{
		{
			name:              "Detect v13.0 when available",
			supportedVersions: []string{"13.0", "12.0", "3.0"},
			expectedVersion:   "13.0",
			expectError:       false,
		},
		{
			name:              "Fallback to v12.0 when v13.0 not available",
			supportedVersions: []string{"12.0", "3.0"},
			expectedVersion:   "12.0",
			expectError:       false,
		},
		{
			name:              "Fallback to v3.0 when only v3.0 available",
			supportedVersions: []string{"3.0"},
			expectedVersion:   "3.0",
			expectError:       false,
		},
		{
			name:              "Error when no versions supported",
			supportedVersions: []string{},
			expectedVersion:   "",
			expectError:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := createVersionDetectionMockServer(tt.supportedVersions)
			defer server.Close()

			cfg := createTestConfig(server.URL, "") // Empty version to trigger detection
			client := NewNbuClient(cfg)

			baseURL := cfg.GetNBUBaseURL()
			detector := NewAPIVersionDetector(client, baseURL, cfg.NbuServer.APIKey)
			detectedVersion, err := detector.DetectVersion(context.Background())

			validateVersionDetectionResult(t, detectedVersion, err, tt.expectedVersion, tt.expectError)
		})
	}
}

// TestFallbackBehavior tests the version fallback logic in detail
func TestFallbackBehavior(t *testing.T) {
	attemptedVersions := []string{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		acceptHeader := r.Header.Get("Accept")
		requestedVersion := ""
		if strings.Contains(acceptHeader, testutil.APIVersion130) {
			requestedVersion = "13.0"
		} else if strings.Contains(acceptHeader, testutil.APIVersion120) {
			requestedVersion = "12.0"
		} else if strings.Contains(acceptHeader, testutil.APIVersion30) {
			requestedVersion = "3.0"
		}

		attemptedVersions = append(attemptedVersions, requestedVersion)

		// Only v3.0 works
		if requestedVersion == "3.0" {
			response := createMinimalJobsResponse()
			w.Header().Set(testutil.ContentTypeHeader, "application/vnd.netbackup+json;version=3.0")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(response)
		} else {
			w.WriteHeader(http.StatusNotAcceptable)
			errorResponse := map[string]interface{}{
				"errorCode":    406,
				"errorMessage": fmt.Sprintf("API version %s is not supported", requestedVersion),
			}
			_ = json.NewEncoder(w).Encode(errorResponse)
		}
	}))
	defer server.Close()

	cfg := createTestConfig(server.URL, "")
	client := NewNbuClient(cfg)

	baseURL := cfg.GetNBUBaseURL()
	detector := NewAPIVersionDetector(client, baseURL, cfg.NbuServer.APIKey)
	detectedVersion, err := detector.DetectVersion(context.Background())

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if detectedVersion != "3.0" {
		t.Errorf("Expected version 3.0, got %s", detectedVersion)
	}

	// Verify fallback order: 13.0 -> 12.0 -> 3.0
	expectedOrder := []string{"13.0", "12.0", "3.0"}
	if len(attemptedVersions) != len(expectedOrder) {
		t.Errorf("Expected %d version attempts, got %d", len(expectedOrder), len(attemptedVersions))
	}

	for i, expected := range expectedOrder {
		if i < len(attemptedVersions) && attemptedVersions[i] != expected {
			t.Errorf("Attempt %d: expected version %s, got %s", i+1, expected, attemptedVersions[i])
		}
	}
}

// TestConfigurationOverride tests that explicit configuration overrides version detection
func TestConfigurationOverride(t *testing.T) {
	detectionAttempted := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		acceptHeader := r.Header.Get("Accept")

		// Track if version detection was attempted (would try 13.0 first)
		if strings.Contains(acceptHeader, testutil.APIVersion130) {
			detectionAttempted = true
		}

		// Only accept v12.0
		if strings.Contains(acceptHeader, testutil.APIVersion120) {
			response := createMinimalJobsResponse()
			w.Header().Set(testutil.ContentTypeHeader, "application/vnd.netbackup+json;version=12.0")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(response)
		} else {
			w.WriteHeader(http.StatusNotAcceptable)
		}
	}))
	defer server.Close()

	// Configure with explicit version 12.0
	cfg := createTestConfig(server.URL, "12.0")
	client := NewNbuClient(cfg)

	// Make a request - should use configured version without detection
	jobsSize := make(map[string]float64)
	jobsCount := make(map[string]float64)
	jobsStatusCount := make(map[string]float64)

	err := FetchAllJobs(context.Background(), client, jobsSize, jobsCount, jobsStatusCount, "5m")
	if err != nil {
		t.Fatalf("FetchAllJobs failed: %v", err)
	}

	// Verify version detection was NOT attempted (no 13.0 request)
	if detectionAttempted {
		t.Error("Version detection should not be attempted when version is explicitly configured")
	}
}
