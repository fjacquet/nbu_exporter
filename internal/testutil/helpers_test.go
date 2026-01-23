package testutil_test

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fjacquet/nbu_exporter/internal/testutil"
)

// mockTB is a mock implementation of testing.TB that captures fatal calls
type mockTB struct {
	testing.TB
	helperCalled bool
	fatalCalled  bool
	fatalMsg     string
}

func newMockTB() *mockTB {
	return &mockTB{}
}

func (m *mockTB) Helper() {
	m.helperCalled = true
}

func (m *mockTB) Fatalf(format string, args ...interface{}) {
	m.fatalCalled = true
	m.fatalMsg = fmt.Sprintf(format, args...)
}

func (m *mockTB) Fatal(args ...interface{}) {
	m.fatalCalled = true
	m.fatalMsg = fmt.Sprint(args...)
}

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

// TestMockServerBuilderExtended provides additional coverage for MockServerBuilder
func TestMockServerBuilderExtended(t *testing.T) {
	t.Run("WithJobsEndpoint_WrongVersion", func(t *testing.T) {
		// Test WithJobsEndpoint returns 406 when version doesn't match
		jobsResponse := map[string]interface{}{
			"data": []map[string]interface{}{
				{"id": "1", "type": "backup"},
			},
		}

		server := testutil.NewMockServer().
			WithJobsEndpoint("13.0", jobsResponse).
			Build()
		defer server.Close()

		// Request with wrong version
		req, err := http.NewRequest("GET", server.URL+testutil.TestPathAdminJobs, nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set(testutil.AcceptHeader, "application/vnd.netbackup+json;version=9.0")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer closeResponseBody(t, resp)

		if resp.StatusCode != http.StatusNotAcceptable {
			t.Errorf("Expected status 406 for wrong version, got %d", resp.StatusCode)
		}

		// Verify error message in response
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response body: %v", err)
		}
		if !strings.Contains(string(body), "API version not supported") {
			t.Errorf("Expected error message about API version, got %q", string(body))
		}
	})

	t.Run("WithErrorResponse_Success", func(t *testing.T) {
		// Test WithErrorResponse with 200 OK (non-error status)
		server := testutil.NewMockServer().
			WithErrorResponse("/success", http.StatusOK).
			Build()
		defer server.Close()

		req, _ := http.NewRequest("GET", server.URL+"/success", nil)
		resp, _ := http.DefaultClient.Do(req)
		defer closeResponseBody(t, resp)

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("WithErrorResponse_ServerError", func(t *testing.T) {
		// Test WithErrorResponse with 500 Internal Server Error
		server := testutil.NewMockServer().
			WithErrorResponse("/error", http.StatusInternalServerError).
			Build()
		defer server.Close()

		req, _ := http.NewRequest("GET", server.URL+"/error", nil)
		resp, _ := http.DefaultClient.Do(req)
		defer closeResponseBody(t, resp)

		if resp.StatusCode != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		if !strings.Contains(string(body), "Internal Server Error") {
			t.Errorf("Expected error message, got %q", string(body))
		}
	})

	t.Run("MultipleEndpoints", func(t *testing.T) {
		// Test server with multiple different endpoints
		server := testutil.NewMockServer().
			WithJobsEndpoint("13.0", map[string]interface{}{"jobs": "data"}).
			WithStorageEndpoint("13.0", map[string]interface{}{"storage": "data"}).
			WithErrorResponse("/health", http.StatusOK).
			WithCustomEndpoint("/custom", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusCreated)
			}).
			Build()
		defer server.Close()

		// Test each endpoint
		endpoints := map[string]int{
			testutil.TestPathAdminJobs:    http.StatusOK,
			testutil.TestPathStorageUnits: http.StatusOK,
			"/health":                     http.StatusOK,
			"/custom":                     http.StatusCreated,
		}

		for path, expectedStatus := range endpoints {
			req, _ := http.NewRequest("GET", server.URL+path, nil)
			if path == testutil.TestPathAdminJobs || path == testutil.TestPathStorageUnits {
				req.Header.Set(testutil.AcceptHeader, "application/vnd.netbackup+json;version=13.0")
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("Request to %s failed: %v", path, err)
			}
			if resp.StatusCode != expectedStatus {
				t.Errorf("Endpoint %s: expected status %d, got %d", path, expectedStatus, resp.StatusCode)
			}
			resp.Body.Close()
		}
	})

	t.Run("WithVersionDetection_AcceptedFalse", func(t *testing.T) {
		// Test WithVersionDetection where version exists but is rejected
		server := testutil.NewMockServer().
			WithVersionDetection(map[string]bool{
				"13.0": false, // Explicitly rejected
				"12.0": true,  // Accepted
			}).
			Build()
		defer server.Close()

		// 13.0 should be rejected
		req13, _ := http.NewRequest("GET", server.URL+testutil.TestPathAdminJobs, nil)
		req13.Header.Set(testutil.AcceptHeader, "application/vnd.netbackup+json;version=13.0")
		resp13, _ := http.DefaultClient.Do(req13)
		if resp13.StatusCode != http.StatusNotAcceptable {
			t.Errorf("Expected 406 for rejected version 13.0, got %d", resp13.StatusCode)
		}
		resp13.Body.Close()

		// 12.0 should be accepted
		req12, _ := http.NewRequest("GET", server.URL+testutil.TestPathAdminJobs, nil)
		req12.Header.Set(testutil.AcceptHeader, "application/vnd.netbackup+json;version=12.0")
		resp12, _ := http.DefaultClient.Do(req12)
		if resp12.StatusCode != http.StatusOK {
			t.Errorf("Expected 200 for accepted version 12.0, got %d", resp12.StatusCode)
		}
		resp12.Body.Close()
	})

	t.Run("WithVersionDetection_StorageEndpoint", func(t *testing.T) {
		// Test WithVersionDetection sets up storage endpoint correctly
		server := testutil.NewMockServer().
			WithVersionDetection(map[string]bool{
				"13.0": true,
			}).
			Build()
		defer server.Close()

		req, _ := http.NewRequest("GET", server.URL+testutil.TestPathStorageUnits, nil)
		req.Header.Set(testutil.AcceptHeader, "application/vnd.netbackup+json;version=13.0")
		resp, _ := http.DefaultClient.Do(req)
		defer closeResponseBody(t, resp)

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected 200 for storage endpoint with version 13.0, got %d", resp.StatusCode)
		}
	})

	t.Run("DefaultHandler404_NoHandlers", func(t *testing.T) {
		// Test server with no handlers at all returns 404 for any path
		server := testutil.NewMockServer().Build()
		defer server.Close()

		req, _ := http.NewRequest("GET", server.URL+"/any/path", nil)
		resp, _ := http.DefaultClient.Do(req)
		defer closeResponseBody(t, resp)

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected 404, got %d", resp.StatusCode)
		}
	})
}

// TestLoadTestData tests the LoadTestData helper function
func TestLoadTestData(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// Create a temporary test file
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.json")
		expectedContent := `{"key": "value"}`
		if err := os.WriteFile(testFile, []byte(expectedContent), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Test LoadTestData successfully reads the file
		data := testutil.LoadTestData(t, testFile)
		if string(data) != expectedContent {
			t.Errorf("Expected %q, got %q", expectedContent, string(data))
		}
	})

	t.Run("ReadExistingTestdata", func(t *testing.T) {
		// Test reading existing testdata file (created for coverage tests)
		data := testutil.LoadTestData(t, "testdata/sample.json")
		if len(data) == 0 {
			t.Error("Expected non-empty data from sample.json")
		}
		if !strings.Contains(string(data), "test") {
			t.Errorf("Expected sample.json to contain 'test', got %q", string(data))
		}
	})
}

// TestAssertionHelpersEdgeCases tests assertion helpers with various edge cases
func TestAssertionHelpersEdgeCases(t *testing.T) {
	t.Run("AssertNoError_NoMsgAndArgs", func(t *testing.T) {
		// AssertNoError with nil error and no message
		testutil.AssertNoError(t, nil)
	})

	t.Run("AssertNoError_WithMsgAndArgs", func(t *testing.T) {
		// AssertNoError with nil error and message with args
		testutil.AssertNoError(t, nil, "operation %s should succeed", "test")
	})

	t.Run("AssertNoError_WithSingleMsg", func(t *testing.T) {
		// AssertNoError with just a message, no args
		testutil.AssertNoError(t, nil, "simple message")
	})

	t.Run("AssertError_Success", func(t *testing.T) {
		// AssertError should not fail when error is not nil
		testutil.AssertError(t, errors.New("expected error"))
	})

	t.Run("AssertError_WithMsg", func(t *testing.T) {
		// AssertError with custom message when error is present
		testutil.AssertError(t, errors.New("expected error"), "validation should fail")
	})

	t.Run("AssertError_WithMsgAndArgs", func(t *testing.T) {
		// AssertError with format message and args
		testutil.AssertError(t, errors.New("expected error"), "%s should fail", "validation")
	})

	t.Run("AssertContains_NoMsgAndArgs", func(t *testing.T) {
		// AssertContains without custom message
		testutil.AssertContains(t, "hello world", "world")
	})

	t.Run("AssertContains_WithMultipleArgs", func(t *testing.T) {
		// AssertContains with message containing multiple format args
		testutil.AssertContains(t, "hello world", "world", "%s should contain %s", "greeting", "world")
	})

	t.Run("AssertContains_SingleMsg", func(t *testing.T) {
		// AssertContains with just a message, no args
		testutil.AssertContains(t, "hello world", "hello", "should start with hello")
	})

	t.Run("AssertContains_EmptySubstring", func(t *testing.T) {
		// Empty substring is always contained
		testutil.AssertContains(t, "hello world", "")
	})

	t.Run("AssertEqual_NoMsgAndArgs", func(t *testing.T) {
		// AssertEqual without custom message
		testutil.AssertEqual(t, "test", "test")
	})

	t.Run("AssertEqual_WithMsgAndArgs", func(t *testing.T) {
		// AssertEqual with custom message
		testutil.AssertEqual(t, 100, 100, "count should be %d", 100)
	})

	t.Run("AssertEqual_SingleMsg", func(t *testing.T) {
		// AssertEqual with just a message
		testutil.AssertEqual(t, 42, 42, "meaning of life")
	})

	t.Run("AssertEqual_DifferentTypes", func(t *testing.T) {
		// Test AssertEqual with various comparable types
		testutil.AssertEqual(t, true, true, "bool comparison")
		testutil.AssertEqual(t, 3.14, 3.14, "float comparison")
		testutil.AssertEqual(t, byte(65), byte(65), "byte comparison")
	})

	t.Run("AssertEqual_Strings", func(t *testing.T) {
		testutil.AssertEqual(t, "hello", "hello", "string comparison")
		testutil.AssertEqual(t, "", "", "empty string comparison")
	})

	t.Run("AssertEqual_Integers", func(t *testing.T) {
		testutil.AssertEqual(t, int(42), int(42), "int comparison")
		testutil.AssertEqual(t, int64(9223372036854775807), int64(9223372036854775807), "int64 comparison")
		testutil.AssertEqual(t, uint(100), uint(100), "uint comparison")
	})

	t.Run("AssertEqual_Zero", func(t *testing.T) {
		testutil.AssertEqual(t, 0, 0, "zero comparison")
		testutil.AssertEqual(t, 0.0, 0.0, "zero float comparison")
	})

	t.Run("AssertEqual_Negative", func(t *testing.T) {
		testutil.AssertEqual(t, -1, -1, "negative int comparison")
		testutil.AssertEqual(t, -3.14, -3.14, "negative float comparison")
	})
}

// TestAssertionHelpersFailurePaths tests the failure paths using mockTB
func TestAssertionHelpersFailurePaths(t *testing.T) {
	t.Run("AssertNoError_Fails_NoMsg", func(t *testing.T) {
		mock := newMockTB()
		testutil.AssertNoError(mock, errors.New("test error"))
		if !mock.fatalCalled {
			t.Error("Expected Fatalf to be called")
		}
		if !strings.Contains(mock.fatalMsg, "test error") {
			t.Errorf("Expected message to contain error, got %q", mock.fatalMsg)
		}
	})

	t.Run("AssertNoError_Fails_WithMsg", func(t *testing.T) {
		mock := newMockTB()
		testutil.AssertNoError(mock, errors.New("net error"), "fetch from %s failed", "server")
		if !mock.fatalCalled {
			t.Error("Expected Fatalf to be called")
		}
		if !strings.Contains(mock.fatalMsg, "fetch from server failed") {
			t.Errorf("Expected custom message, got %q", mock.fatalMsg)
		}
	})

	t.Run("AssertError_Fails_NoMsg", func(t *testing.T) {
		mock := newMockTB()
		testutil.AssertError(mock, nil)
		if !mock.fatalCalled {
			t.Error("Expected Fatal to be called")
		}
		if !strings.Contains(mock.fatalMsg, "Expected error") {
			t.Errorf("Expected default message, got %q", mock.fatalMsg)
		}
	})

	t.Run("AssertError_Fails_WithMsg", func(t *testing.T) {
		mock := newMockTB()
		testutil.AssertError(mock, nil, "validate %s should fail", "input")
		if !mock.fatalCalled {
			t.Error("Expected Fatalf to be called")
		}
		if !strings.Contains(mock.fatalMsg, "validate input should fail") {
			t.Errorf("Expected custom message, got %q", mock.fatalMsg)
		}
	})

	t.Run("AssertContains_Fails_NoMsg", func(t *testing.T) {
		mock := newMockTB()
		testutil.AssertContains(mock, "hello", "xyz")
		if !mock.fatalCalled {
			t.Error("Expected Fatalf to be called")
		}
		if !strings.Contains(mock.fatalMsg, "hello") || !strings.Contains(mock.fatalMsg, "xyz") {
			t.Errorf("Expected both strings in message, got %q", mock.fatalMsg)
		}
	})

	t.Run("AssertContains_Fails_WithMsg", func(t *testing.T) {
		mock := newMockTB()
		testutil.AssertContains(mock, "hello", "xyz", "response should contain %s", "xyz")
		if !mock.fatalCalled {
			t.Error("Expected Fatalf to be called")
		}
		if !strings.Contains(mock.fatalMsg, "response should contain xyz") {
			t.Errorf("Expected custom message, got %q", mock.fatalMsg)
		}
	})

	t.Run("AssertEqual_Fails_NoMsg", func(t *testing.T) {
		mock := newMockTB()
		testutil.AssertEqual(mock, 42, 43)
		if !mock.fatalCalled {
			t.Error("Expected Fatalf to be called")
		}
		if !strings.Contains(mock.fatalMsg, "42") || !strings.Contains(mock.fatalMsg, "43") {
			t.Errorf("Expected both values in message, got %q", mock.fatalMsg)
		}
	})

	t.Run("AssertEqual_Fails_WithMsg", func(t *testing.T) {
		mock := newMockTB()
		testutil.AssertEqual(mock, "expected", "actual", "value should be %q", "expected")
		if !mock.fatalCalled {
			t.Error("Expected Fatalf to be called")
		}
		if !strings.Contains(mock.fatalMsg, "value should be") {
			t.Errorf("Expected custom message, got %q", mock.fatalMsg)
		}
	})
}
