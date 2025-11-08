package exporter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fjacquet/nbu_exporter/internal/models"
)

// mockJobsResponse represents a minimal jobs API response for testing
type mockJobsResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

// createVersionTestServer creates a test HTTP server that responds to version detection requests
func createVersionTestServer(acceptedVersion string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		acceptHeader := r.Header.Get(HeaderAccept)
		expectedAccept := "application/vnd.netbackup+json;version=" + acceptedVersion

		if acceptHeader == expectedAccept {
			// Version matches - return success
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(mockJobsResponse{
				Data: []struct {
					ID string `json:"id"`
				}{},
			})
		} else {
			// Version doesn't match - return 406
			w.WriteHeader(http.StatusNotAcceptable)
		}
	}))
}

// setupVersionDetectorTest creates a configured client and detector for testing
func setupVersionDetectorTest(serverURL, acceptedVersion string) (*NbuClient, *APIVersionDetector, models.Config) {
	cfg := createTestConfig(serverURL, acceptedVersion)
	cfg.NbuServer.APIKey = "test-key"
	cfg.NbuServer.APIVersion = "" // Will be detected
	cfg.NbuServer.URI = ""        // No base URI for test server
	client := NewNbuClient(cfg)
	detector := NewAPIVersionDetector(client, &cfg)
	return client, detector, cfg
}

func TestAPIVersionDetector_DetectVersion_Success(t *testing.T) {
	tests := []struct {
		name            string
		serverVersion   string // Version the mock server will accept
		expectedVersion string // Version we expect to detect
	}{
		{
			name:            "detects API version 13.0 (NetBackup 11.0)",
			serverVersion:   models.APIVersion130,
			expectedVersion: models.APIVersion130,
		},
		{
			name:            "detects API version 12.0 (NetBackup 10.5)",
			serverVersion:   models.APIVersion120,
			expectedVersion: models.APIVersion120,
		},
		{
			name:            "detects API version 3.0 (NetBackup 10.0)",
			serverVersion:   models.APIVersion30,
			expectedVersion: models.APIVersion30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := createVersionTestServer(tt.serverVersion)
			defer server.Close()

			_, detector, _ := setupVersionDetectorTest(server.URL, tt.serverVersion)

			version, err := detector.DetectVersion(context.Background())
			if err != nil {
				t.Errorf("DetectVersion() unexpected error = %v", err)
			}

			if version != tt.expectedVersion {
				t.Errorf("DetectVersion() = %v, want %v", version, tt.expectedVersion)
			}
		})
	}
}

func TestAPIVersionDetector_DetectVersion_Fallback(t *testing.T) {
	tests := []struct {
		name            string
		serverVersion   string // Version the mock server will accept
		expectedVersion string // Version we expect to detect after fallback
		description     string
	}{
		{
			name:            "falls back from 13.0 to 12.0",
			serverVersion:   models.APIVersion120,
			expectedVersion: models.APIVersion120,
			description:     "Server doesn't support 13.0, falls back to 12.0",
		},
		{
			name:            "falls back from 13.0 to 12.0 to 3.0",
			serverVersion:   models.APIVersion30,
			expectedVersion: models.APIVersion30,
			description:     "Server only supports 3.0, falls back through all versions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := createVersionTestServer(tt.serverVersion)
			defer server.Close()

			_, detector, _ := setupVersionDetectorTest(server.URL, tt.serverVersion)

			version, err := detector.DetectVersion(context.Background())
			if err != nil {
				t.Errorf("DetectVersion() unexpected error = %v", err)
			}

			if version != tt.expectedVersion {
				t.Errorf("DetectVersion() = %v, want %v", version, tt.expectedVersion)
			}
		})
	}
}

func TestAPIVersionDetector_DetectVersion_AllVersionsFail(t *testing.T) {
	// Create a test server that rejects all versions
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotAcceptable)
	}))
	defer server.Close()

	_, detector, _ := setupVersionDetectorTest(server.URL, models.APIVersion130)

	version, err := detector.DetectVersion(context.Background())
	if err == nil {
		t.Error("DetectVersion() expected error when all versions fail, got nil")
	}

	if version != "" {
		t.Errorf("DetectVersion() = %v, want empty string on error", version)
	}

	// Verify error message contains troubleshooting information
	errMsg := err.Error()
	if len(errMsg) == 0 {
		t.Error("DetectVersion() error message should not be empty")
	}
}

func TestAPIVersionDetector_DetectVersion_AuthenticationError(t *testing.T) {
	// Create a test server that returns 401 Unauthorized
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	_, detector, _ := setupVersionDetectorTest(server.URL, models.APIVersion130)

	version, err := detector.DetectVersion(context.Background())
	if err == nil {
		t.Error("DetectVersion() expected error for authentication failure, got nil")
	}

	if version != "" {
		t.Errorf("DetectVersion() = %v, want empty string on auth error", version)
	}
}

func TestAPIVersionDetector_TryVersion_RetryLogic(t *testing.T) {
	tests := []struct {
		name           string
		failureCount   int // Number of times to fail before succeeding
		expectedResult bool
	}{
		{
			name:           "succeeds on first attempt",
			failureCount:   0,
			expectedResult: true,
		},
		{
			name:           "succeeds after one retry",
			failureCount:   1,
			expectedResult: true,
		},
		{
			name:           "succeeds after two retries",
			failureCount:   2,
			expectedResult: true,
		},
		{
			name:           "fails after max retries",
			failureCount:   3,
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attemptCount := 0

			// Create a test server that fails N times then succeeds
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if attemptCount < tt.failureCount {
					attemptCount++
					// Simulate transient network error by returning 503
					w.WriteHeader(http.StatusServiceUnavailable)
					return
				}

				// Success after N failures
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(mockJobsResponse{
					Data: []struct {
						ID string `json:"id"`
					}{},
				})
			}))
			defer server.Close()

			_, detector, _ := setupVersionDetectorTest(server.URL, models.APIVersion130)

			// Use a shorter retry delay for testing
			detector.retryConfig = RetryConfig{
				MaxAttempts:   3,
				InitialDelay:  10 * time.Millisecond,
				MaxDelay:      100 * time.Millisecond,
				BackoffFactor: 2.0,
			}

			result := detector.tryVersion(context.Background(), models.APIVersion130)
			if result != tt.expectedResult {
				t.Errorf("tryVersion() = %v, want %v", result, tt.expectedResult)
			}
		})
	}
}

func TestAPIVersionDetector_TryVersion_HTTPStatusCodes(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		expectedResult bool
		description    string
	}{
		{
			name:           "returns true for HTTP 200",
			statusCode:     http.StatusOK,
			expectedResult: true,
			description:    "Version is supported",
		},
		{
			name:           "returns false for HTTP 406",
			statusCode:     http.StatusNotAcceptable,
			expectedResult: false,
			description:    "Version not supported",
		},
		{
			name:           "returns false for HTTP 401",
			statusCode:     http.StatusUnauthorized,
			expectedResult: false,
			description:    "Authentication error",
		},
		{
			name:           "returns false for HTTP 500",
			statusCode:     http.StatusInternalServerError,
			expectedResult: false,
			description:    "Server error",
		},
		{
			name:           "returns false for HTTP 404",
			statusCode:     http.StatusNotFound,
			expectedResult: false,
			description:    "Endpoint not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test server that returns the specified status code
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(mockJobsResponse{
						Data: []struct {
							ID string `json:"id"`
						}{},
					})
				}
			}))
			defer server.Close()

			_, detector, _ := setupVersionDetectorTest(server.URL, models.APIVersion130)

			result := detector.tryVersion(context.Background(), models.APIVersion130)
			if result != tt.expectedResult {
				t.Errorf("tryVersion() = %v, want %v for status %d (%s)",
					result, tt.expectedResult, tt.statusCode, tt.description)
			}
		})
	}
}

func TestAPIVersionDetector_TryVersion_ContextCancellation(t *testing.T) {
	// Create a test server with a delay
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_, detector, _ := setupVersionDetectorTest(server.URL, models.APIVersion130)

	// Create a context that will be cancelled immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result := detector.tryVersion(ctx, models.APIVersion130)
	if result {
		t.Error("tryVersion() expected false for cancelled context, got true")
	}
}

func TestAPIVersionDetector_ExponentialBackoff(t *testing.T) {
	attemptTimes := []time.Time{}

	// Create a test server that tracks request times
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptTimes = append(attemptTimes, time.Now())
		// Always fail to trigger retries
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	_, detector, _ := setupVersionDetectorTest(server.URL, models.APIVersion130)

	// Use short delays for testing
	detector.retryConfig = RetryConfig{
		MaxAttempts:   3,
		InitialDelay:  50 * time.Millisecond,
		MaxDelay:      500 * time.Millisecond,
		BackoffFactor: 2.0,
	}

	detector.tryVersion(context.Background(), models.APIVersion130)

	// Verify we made 3 attempts
	if len(attemptTimes) != 3 {
		t.Errorf("Expected 3 attempts, got %d", len(attemptTimes))
		return
	}

	// Verify exponential backoff between attempts
	// First delay should be ~50ms, second should be ~100ms
	if len(attemptTimes) >= 2 {
		firstDelay := attemptTimes[1].Sub(attemptTimes[0])
		if firstDelay < 40*time.Millisecond || firstDelay > 100*time.Millisecond {
			t.Errorf("First retry delay = %v, expected ~50ms", firstDelay)
		}
	}

	if len(attemptTimes) >= 3 {
		secondDelay := attemptTimes[2].Sub(attemptTimes[1])
		if secondDelay < 80*time.Millisecond || secondDelay > 200*time.Millisecond {
			t.Errorf("Second retry delay = %v, expected ~100ms", secondDelay)
		}
	}
}
