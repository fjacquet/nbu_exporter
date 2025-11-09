package testutil_test

import (
	"net/http"
	"net/http/httptest"
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
