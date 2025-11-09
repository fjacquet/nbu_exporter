package testutil_test

import (
	"net/http"
	"testing"

	"github.com/fjacquet/nbu_exporter/internal/testutil"
)

// TestMockServerBuilder demonstrates the usage of MockServerBuilder
func TestMockServerBuilder(t *testing.T) {
	t.Run("WithJobsEndpoint", func(t *testing.T) {
		// Create a mock server with jobs endpoint
		jobsResponse := map[string]interface{}{
			"data": []map[string]interface{}{
				{"id": "1", "type": "backup"},
			},
		}

		server := testutil.NewMockServer().
			WithJobsEndpoint("13.0", jobsResponse).
			Build()
		defer server.Close()

		// Make a request to the jobs endpoint
		req, err := http.NewRequest("GET", server.URL+testutil.TestPathAdminJobs, nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set(testutil.AcceptHeader, "application/vnd.netbackup+json;version=13.0")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("WithVersionDetection", func(t *testing.T) {
		// Create a mock server that accepts version 13.0 but rejects 12.0
		server := testutil.NewMockServer().
			WithVersionDetection(map[string]bool{
				"13.0": true,
				"12.0": false,
			}).
			Build()
		defer server.Close()

		// Test accepted version
		req, _ := http.NewRequest("GET", server.URL+testutil.TestPathAdminJobs, nil)
		req.Header.Set(testutil.AcceptHeader, "application/vnd.netbackup+json;version=13.0")
		resp, _ := http.DefaultClient.Do(req)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200 for version 13.0, got %d", resp.StatusCode)
		}

		// Test rejected version
		req2, _ := http.NewRequest("GET", server.URL+testutil.TestPathAdminJobs, nil)
		req2.Header.Set(testutil.AcceptHeader, "application/vnd.netbackup+json;version=12.0")
		resp2, _ := http.DefaultClient.Do(req2)
		defer resp2.Body.Close()

		if resp2.StatusCode != http.StatusNotAcceptable {
			t.Errorf("Expected status 406 for version 12.0, got %d", resp2.StatusCode)
		}
	})

	t.Run("WithTLS", func(t *testing.T) {
		// Create a TLS-enabled mock server
		server := testutil.NewMockServer().
			WithTLS().
			WithJobsEndpoint("13.0", map[string]interface{}{"data": []interface{}{}}).
			Build()
		defer server.Close()

		// Verify it's using HTTPS
		if server.URL[:5] != "https" {
			t.Errorf("Expected HTTPS URL, got %s", server.URL)
		}
	})

	t.Run("WithErrorResponse", func(t *testing.T) {
		// Create a mock server that returns 401 Unauthorized
		server := testutil.NewMockServer().
			WithErrorResponse(testutil.TestPathAdminJobs, http.StatusUnauthorized).
			Build()
		defer server.Close()

		req, _ := http.NewRequest("GET", server.URL+testutil.TestPathAdminJobs, nil)
		resp, _ := http.DefaultClient.Do(req)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", resp.StatusCode)
		}
	})
}

// TestHelperFunctions demonstrates the usage of helper functions
func TestHelperFunctions(t *testing.T) {
	t.Run("AssertNoError", func(t *testing.T) {
		// This would fail the test if err was not nil
		testutil.AssertNoError(t, nil, "Operation should succeed")
	})

	t.Run("AssertContains", func(t *testing.T) {
		testutil.AssertContains(t, "hello world", "world", "String should contain substring")
	})

	t.Run("AssertEqual", func(t *testing.T) {
		testutil.AssertEqual(t, 42, 42, "Values should be equal")
	})
}

// TestConstants demonstrates the usage of shared constants
func TestConstants(t *testing.T) {
	// Verify constants are accessible
	if testutil.TestAPIKey == "" {
		t.Error("TestAPIKey should not be empty")
	}

	if testutil.TestPathAdminJobs == "" {
		t.Error("TestPathAdminJobs should not be empty")
	}

	if testutil.ContentTypeJSON == "" {
		t.Error("ContentTypeJSON should not be empty")
	}
}
