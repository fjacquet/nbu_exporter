package testutil_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fjacquet/nbu_exporter/internal/testutil"
)

const (
	errMsgCloseBody = "Failed to close response body: %v"
)

// TestMockServerBuilder demonstrates the usage of MockServerBuilder
func TestMockServerBuilder(t *testing.T) {
	t.Run("WithJobsEndpoint", func(t *testing.T) {
		testJobsEndpoint(t)
	})

	t.Run("WithVersionDetection", func(t *testing.T) {
		testVersionDetection(t)
	})

	t.Run("WithTLS", func(t *testing.T) {
		testTLSServer(t)
	})

	t.Run("WithErrorResponse", func(t *testing.T) {
		testErrorResponse(t)
	})

	t.Run("WithStorageEndpoint", func(t *testing.T) {
		testStorageEndpoint(t)
	})

	t.Run("WithCustomEndpoint", func(t *testing.T) {
		testCustomEndpoint(t)
	})

	t.Run("DefaultHandler404", func(t *testing.T) {
		testDefaultHandler404(t)
	})

	t.Run("VersionDetectionNoMatch", func(t *testing.T) {
		testVersionDetectionNoMatch(t)
	})
}

// testJobsEndpoint tests the WithJobsEndpoint builder method
func testJobsEndpoint(t *testing.T) {
	t.Helper()

	jobsResponse := map[string]interface{}{
		"data": []map[string]interface{}{
			{"id": "1", "type": "backup"},
		},
	}

	server := testutil.NewMockServer().
		WithJobsEndpoint("13.0", jobsResponse).
		Build()
	defer server.Close()

	req, err := http.NewRequest("GET", server.URL+testutil.TestPathAdminJobs, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set(testutil.AcceptHeader, "application/vnd.netbackup+json;version=13.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer closeResponseBody(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

// testVersionDetection tests the WithVersionDetection builder method
func testVersionDetection(t *testing.T) {
	t.Helper()

	server := testutil.NewMockServer().
		WithVersionDetection(map[string]bool{
			"13.0": true,
			"12.0": false,
		}).
		Build()
	defer server.Close()

	testAcceptedVersion(t, server)
	testRejectedVersion(t, server)
}

// testAcceptedVersion verifies that accepted API versions return 200 OK
func testAcceptedVersion(t *testing.T, server *httptest.Server) {
	t.Helper()

	req, _ := http.NewRequest("GET", server.URL+testutil.TestPathAdminJobs, nil)
	req.Header.Set(testutil.AcceptHeader, "application/vnd.netbackup+json;version=13.0")
	resp, _ := http.DefaultClient.Do(req)
	defer closeResponseBody(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for version 13.0, got %d", resp.StatusCode)
	}
}

// testRejectedVersion verifies that rejected API versions return 406 Not Acceptable
func testRejectedVersion(t *testing.T, server *httptest.Server) {
	t.Helper()

	req, _ := http.NewRequest("GET", server.URL+testutil.TestPathAdminJobs, nil)
	req.Header.Set(testutil.AcceptHeader, "application/vnd.netbackup+json;version=12.0")
	resp, _ := http.DefaultClient.Do(req)
	defer closeResponseBody(t, resp)

	if resp.StatusCode != http.StatusNotAcceptable {
		t.Errorf("Expected status 406 for version 12.0, got %d", resp.StatusCode)
	}
}

// testTLSServer tests the WithTLS builder method
func testTLSServer(t *testing.T) {
	t.Helper()

	server := testutil.NewMockServer().
		WithTLS().
		WithJobsEndpoint("13.0", map[string]interface{}{"data": []interface{}{}}).
		Build()
	defer server.Close()

	if server.URL[:5] != "https" {
		t.Errorf("Expected HTTPS URL, got %s", server.URL)
	}
}

// testErrorResponse tests the WithErrorResponse builder method
func testErrorResponse(t *testing.T) {
	t.Helper()

	server := testutil.NewMockServer().
		WithErrorResponse(testutil.TestPathAdminJobs, http.StatusUnauthorized).
		Build()
	defer server.Close()

	req, _ := http.NewRequest("GET", server.URL+testutil.TestPathAdminJobs, nil)
	resp, _ := http.DefaultClient.Do(req)
	defer closeResponseBody(t, resp)

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

// testStorageEndpoint tests the WithStorageEndpoint builder method
func testStorageEndpoint(t *testing.T) {
	t.Helper()

	storageResponse := map[string]interface{}{
		"data": []map[string]interface{}{
			{"name": "disk-pool-1", "type": "AdvancedDisk"},
		},
	}

	server := testutil.NewMockServer().
		WithStorageEndpoint("13.0", storageResponse).
		Build()
	defer server.Close()

	// Test with correct version - should return 200 OK
	req, err := http.NewRequest("GET", server.URL+testutil.TestPathStorageUnits, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set(testutil.AcceptHeader, "application/vnd.netbackup+json;version=13.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer closeResponseBody(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for correct version, got %d", resp.StatusCode)
	}

	// Test with wrong version - should return 406 Not Acceptable
	reqWrong, err := http.NewRequest("GET", server.URL+testutil.TestPathStorageUnits, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	reqWrong.Header.Set(testutil.AcceptHeader, "application/vnd.netbackup+json;version=12.0")

	respWrong, err := http.DefaultClient.Do(reqWrong)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer closeResponseBody(t, respWrong)

	if respWrong.StatusCode != http.StatusNotAcceptable {
		t.Errorf("Expected status 406 for wrong version, got %d", respWrong.StatusCode)
	}
}

// testCustomEndpoint tests the WithCustomEndpoint builder method
func testCustomEndpoint(t *testing.T) {
	t.Helper()

	customPath := "/custom/path"
	customResponse := "custom response body"

	customHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(customResponse))
	}

	server := testutil.NewMockServer().
		WithCustomEndpoint(customPath, customHandler).
		Build()
	defer server.Close()

	req, err := http.NewRequest("GET", server.URL+customPath, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer closeResponseBody(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Read and verify response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	if string(body) != customResponse {
		t.Errorf("Expected body %q, got %q", customResponse, string(body))
	}
}

// testDefaultHandler404 tests the default 404 handler for unregistered paths
func testDefaultHandler404(t *testing.T) {
	t.Helper()

	// Create server with only jobs endpoint
	server := testutil.NewMockServer().
		WithJobsEndpoint("13.0", map[string]interface{}{"data": []interface{}{}}).
		Build()
	defer server.Close()

	// Request to nonexistent path should return 404
	req, err := http.NewRequest("GET", server.URL+"/nonexistent/path", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer closeResponseBody(t, resp)

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}

	// Verify response body contains "Endpoint not found"
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	if !strings.Contains(string(body), "Endpoint not found") {
		t.Errorf("Expected body to contain 'Endpoint not found', got %q", string(body))
	}
}

// testVersionDetectionNoMatch tests version detection when no version matches
func testVersionDetectionNoMatch(t *testing.T) {
	t.Helper()

	server := testutil.NewMockServer().
		WithVersionDetection(map[string]bool{
			"13.0": true,
		}).
		Build()
	defer server.Close()

	// Request with unregistered version (3.0) should return 406
	req, err := http.NewRequest("GET", server.URL+testutil.TestPathAdminJobs, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set(testutil.AcceptHeader, "application/vnd.netbackup+json;version=3.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer closeResponseBody(t, resp)

	if resp.StatusCode != http.StatusNotAcceptable {
		t.Errorf("Expected status 406 for unregistered version, got %d", resp.StatusCode)
	}
}

// closeResponseBody closes the response body and logs any errors
func closeResponseBody(t *testing.T, resp *http.Response) {
	t.Helper()
	if err := resp.Body.Close(); err != nil {
		t.Logf(errMsgCloseBody, err)
	}
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
