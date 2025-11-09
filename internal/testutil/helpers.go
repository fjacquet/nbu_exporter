// Package testutil provides shared test utilities and helper functions.
// This file contains fluent builders and common test helpers to reduce
// duplication across test files and improve test maintainability.
package testutil

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// MockServerBuilder provides a fluent interface for creating mock HTTP servers.
// It simplifies test server setup by providing chainable methods for configuring
// different endpoints and responses.
//
// Example usage:
//
//	server := testutil.NewMockServer().
//	    WithJobsEndpoint("13.0", jobsResponse).
//	    WithStorageEndpoint("13.0", storageResponse).
//	    Build()
//	defer server.Close()
type MockServerBuilder struct {
	handlers map[string]http.HandlerFunc
	useTLS   bool
}

// NewMockServer creates a new MockServerBuilder.
func NewMockServer() *MockServerBuilder {
	return &MockServerBuilder{
		handlers: make(map[string]http.HandlerFunc),
		useTLS:   false,
	}
}

// WithTLS enables TLS for the mock server.
func (b *MockServerBuilder) WithTLS() *MockServerBuilder {
	b.useTLS = true
	return b
}

// WithJobsEndpoint adds a handler for the jobs endpoint that validates API version.
// The response parameter should be a struct that will be JSON-encoded.
func (b *MockServerBuilder) WithJobsEndpoint(version string, response interface{}) *MockServerBuilder {
	b.handlers[TestPathAdminJobs] = func(w http.ResponseWriter, r *http.Request) {
		if validateAPIVersion(r, version) {
			writeJSONResponse(w, response)
		} else {
			w.WriteHeader(http.StatusNotAcceptable)
			writeJSONResponse(w, map[string]string{
				"errorMessage": "API version not supported",
			})
		}
	}
	return b
}

// WithStorageEndpoint adds a handler for the storage endpoint that validates API version.
// The response parameter should be a struct that will be JSON-encoded.
func (b *MockServerBuilder) WithStorageEndpoint(version string, response interface{}) *MockServerBuilder {
	b.handlers[TestPathStorageUnits] = func(w http.ResponseWriter, r *http.Request) {
		if validateAPIVersion(r, version) {
			writeJSONResponse(w, response)
		} else {
			w.WriteHeader(http.StatusNotAcceptable)
			writeJSONResponse(w, map[string]string{
				"errorMessage": "API version not supported",
			})
		}
	}
	return b
}

// WithCustomEndpoint adds a custom handler for the specified path.
func (b *MockServerBuilder) WithCustomEndpoint(path string, handler http.HandlerFunc) *MockServerBuilder {
	b.handlers[path] = handler
	return b
}

// WithVersionDetection adds handlers that respond to version detection requests.
// The acceptedVersions map specifies which versions should return 200 OK.
func (b *MockServerBuilder) WithVersionDetection(acceptedVersions map[string]bool) *MockServerBuilder {
	handler := func(w http.ResponseWriter, r *http.Request) {
		acceptHeader := r.Header.Get(AcceptHeader)

		// Check each version
		for version, accepted := range acceptedVersions {
			if strings.Contains(acceptHeader, fmt.Sprintf("version=%s", version)) {
				if accepted {
					w.WriteHeader(http.StatusOK)
					writeJSONResponse(w, map[string]interface{}{"data": []interface{}{}})
				} else {
					w.WriteHeader(http.StatusNotAcceptable)
				}
				return
			}
		}

		// Default: not acceptable
		w.WriteHeader(http.StatusNotAcceptable)
	}

	// Add handler for common endpoints used in version detection
	b.handlers[TestPathAdminJobs] = handler
	b.handlers[TestPathStorageUnits] = handler

	return b
}

// WithErrorResponse adds a handler that returns the specified HTTP status code.
func (b *MockServerBuilder) WithErrorResponse(path string, statusCode int) *MockServerBuilder {
	b.handlers[path] = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		if statusCode >= 400 {
			writeJSONResponse(w, map[string]string{
				"errorMessage": http.StatusText(statusCode),
			})
		}
	}
	return b
}

// Build creates and returns the configured HTTP test server.
func (b *MockServerBuilder) Build() *httptest.Server {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Find matching handler
		if handler, ok := b.handlers[r.URL.Path]; ok {
			handler(w, r)
		} else {
			// Default: 404 Not Found
			w.WriteHeader(http.StatusNotFound)
			writeJSONResponse(w, map[string]string{
				"errorMessage": "Endpoint not found",
			})
		}
	})

	if b.useTLS {
		return httptest.NewTLSServer(handler)
	}
	return httptest.NewServer(handler)
}

// LoadTestData loads test data from a file.
// It uses t.Helper() to report errors at the caller's location.
func LoadTestData(t *testing.T, filename string) []byte {
	t.Helper()
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to read test data file %s: %v", filename, err)
	}
	return data
}

// validateAPIVersion checks if the request has the correct API version header.
func validateAPIVersion(r *http.Request, expectedVersion string) bool {
	acceptHeader := r.Header.Get(AcceptHeader)
	expectedHeader := fmt.Sprintf("version=%s", expectedVersion)
	return strings.Contains(acceptHeader, expectedHeader)
}

// writeJSONResponse writes a JSON response to the ResponseWriter.
func writeJSONResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set(ContentTypeHeader, ContentTypeJSON)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// AssertNoError is a helper that fails the test if err is not nil.
func AssertNoError(t *testing.T, err error, msgAndArgs ...interface{}) {
	t.Helper()
	if err != nil {
		if len(msgAndArgs) > 0 {
			format := msgAndArgs[0].(string)
			args := msgAndArgs[1:]
			t.Fatalf(format+": %v", append(args, err)...)
		} else {
			t.Fatalf("Unexpected error: %v", err)
		}
	}
}

// AssertError is a helper that fails the test if err is nil.
func AssertError(t *testing.T, err error, msgAndArgs ...interface{}) {
	t.Helper()
	if err == nil {
		if len(msgAndArgs) > 0 {
			format := msgAndArgs[0].(string)
			t.Fatalf(format, msgAndArgs[1:]...)
		} else {
			t.Fatal("Expected error, got nil")
		}
	}
}

// AssertContains is a helper that fails the test if the string doesn't contain the substring.
func AssertContains(t *testing.T, s, substr string, msgAndArgs ...interface{}) {
	t.Helper()
	if !strings.Contains(s, substr) {
		if len(msgAndArgs) > 0 {
			format := msgAndArgs[0].(string)
			t.Fatalf(format, msgAndArgs[1:]...)
		} else {
			t.Fatalf("String %q does not contain %q", s, substr)
		}
	}
}

// AssertEqual is a helper that fails the test if the values are not equal.
func AssertEqual(t *testing.T, expected, actual interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	if expected != actual {
		if len(msgAndArgs) > 0 {
			format := msgAndArgs[0].(string)
			t.Fatalf(format, msgAndArgs[1:]...)
		} else {
			t.Fatalf("Expected %v, got %v", expected, actual)
		}
	}
}
