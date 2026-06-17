package exporter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/fjacquet/nbu_exporter/internal/models"
)

// Test constants specific to API compatibility tests
// Note: Common constants like contentTypeHeader, contentTypeJSON, testAPIKey, etc.
// are defined in test_common.go and shared across all test files
const (
	testAPIVersionFormat    = "API_v%s"
	fetchJobsErrorFormat    = "FetchAllJobs failed for version %s: %v"
	fetchStorageErrorFormat = "FetchStorage failed for version %s: %v"
	testDataPathFormat      = "../../testdata/api-versions/%s-response-v%s.json"
)

// TestJobsAPICompatibilityAcrossVersions tests jobs API with all three versions
func TestJobsAPICompatibilityAcrossVersions(t *testing.T) {
	versions := []struct {
		version  string
		filename string
	}{
		{"10.0", "../../testdata/api-versions/jobs-response-v10.json"},
		{"12.0", "../../testdata/api-versions/jobs-response-v12.json"},
		{"13.0", "../../testdata/api-versions/jobs-response-v13.json"},
		{"14.0", "../../testdata/api-versions/jobs-response-v14.json"},
	}

	for _, v := range versions {
		t.Run(fmt.Sprintf(testAPIVersionFormat, v.version), func(t *testing.T) {
			testJobsAPIForVersion(t, v.version, v.filename)
		})
	}
}

// testJobsAPIForVersion tests jobs API for a specific version
func testJobsAPIForVersion(t *testing.T, version, filename string) {
	t.Helper()

	server := createMockServerWithFile(t, filename, version)
	defer server.Close()

	cfg := createTestConfig(server.URL, version)
	client := NewNbuClient(cfg)

	jobsSizeSlice, jobsCountSlice, _, err := FetchAllJobs(context.Background(), client, "5m")
	if err != nil {
		t.Fatalf(fetchJobsErrorFormat, version, err)
	}

	// Convert slices to maps for verification
	jobsSize := jobMetricSliceToMap(jobsSizeSlice)
	jobsCount := jobMetricSliceToMap(jobsCountSlice)

	verifyJobMetrics(t, jobsSize, jobsCount)
}

// jobMetricSliceToMap converts a JobMetricValue slice to a map keyed by String()
func jobMetricSliceToMap(slice []JobMetricValue) map[string]float64 {
	result := make(map[string]float64)
	for _, v := range slice {
		result[v.Key.String()] = v.Value
	}
	return result
}

// storageMetricSliceToMap converts a StorageMetricValue slice to a map keyed by String()
func storageMetricSliceToMap(slice []StorageMetricValue) map[string]float64 {
	result := make(map[string]float64)
	for _, v := range slice {
		result[v.Key.String()] = v.Value
	}
	return result
}

// verifyJobMetrics verifies that job metrics were collected correctly
func verifyJobMetrics(t *testing.T, jobsSize, jobsCount map[string]float64) {
	t.Helper()

	if len(jobsCount) == 0 {
		t.Error("No job count metrics collected")
	}

	backupKey := JobMetricKey{Action: "BACKUP", PolicyType: "VMWARE", Status: "0"}.String()
	if jobsCount[backupKey] != 1 {
		t.Errorf("Expected 1 BACKUP/VMWARE/0 job, got %v", jobsCount[backupKey])
	}

	restoreKey := JobMetricKey{Action: "RESTORE", PolicyType: "STANDARD", Status: "1"}.String()
	if jobsCount[restoreKey] != 1 {
		t.Errorf("Expected 1 RESTORE/STANDARD/1 job, got %v", jobsCount[restoreKey])
	}

	if jobsSize[backupKey] == 0 {
		t.Error("Job size should not be zero for successful backup")
	}
}

// TestStorageAPICompatibilityAcrossVersions tests storage API with all three versions
func TestStorageAPICompatibilityAcrossVersions(t *testing.T) {
	versions := []struct {
		version  string
		filename string
	}{
		{"10.0", "../../testdata/api-versions/storage-response-v10.json"},
		{"12.0", "../../testdata/api-versions/storage-response-v12.json"},
		{"13.0", "../../testdata/api-versions/storage-response-v13.json"},
		{"14.0", "../../testdata/api-versions/storage-response-v14.json"},
	}

	for _, v := range versions {
		t.Run(fmt.Sprintf(testAPIVersionFormat, v.version), func(t *testing.T) {
			testStorageAPIForVersion(t, v.version, v.filename)
		})
	}
}

// testStorageAPIForVersion tests storage API for a specific version
func testStorageAPIForVersion(t *testing.T, version, filename string) {
	t.Helper()

	server := createMockServerWithFile(t, filename, version)
	defer server.Close()

	cfg := createTestConfig(server.URL, version)
	client := NewNbuClient(cfg)

	storageSlice, err := FetchStorage(context.Background(), client)
	if err != nil {
		t.Fatalf(fetchStorageErrorFormat, version, err)
	}

	// Convert slice to map for verification
	storageMetrics := storageMetricSliceToMap(storageSlice)
	verifyStorageMetrics(t, storageMetrics, version)
}

// verifyStorageMetrics verifies that storage metrics were collected correctly
func verifyStorageMetrics(t *testing.T, storageMetrics map[string]float64, version string) {
	t.Helper()

	if len(storageMetrics) == 0 {
		t.Error("No storage metrics collected")
	}

	diskPoolFreeKey := StorageMetricKey{Name: "disk-pool-1", Type: "MEDIA_SERVER", Size: "free"}.String()
	diskPoolUsedKey := StorageMetricKey{Name: "disk-pool-1", Type: "MEDIA_SERVER", Size: "used"}.String()

	if _, exists := storageMetrics[diskPoolFreeKey]; !exists {
		t.Errorf("Missing metric for disk-pool-1 free capacity in version %s", version)
	}

	if _, exists := storageMetrics[diskPoolUsedKey]; !exists {
		t.Errorf("Missing metric for disk-pool-1 used capacity in version %s", version)
	}

	tapeKey := StorageMetricKey{Name: "tape-stu-1", Type: "MEDIA_SERVER", Size: "free"}.String()
	if _, exists := storageMetrics[tapeKey]; exists {
		t.Errorf("Tape storage should be excluded from metrics in version %s", version)
	}
}

// TestMetricsConsistencyAcrossVersions verifies that metric names and labels remain consistent
func TestMetricsConsistencyAcrossVersions(t *testing.T) {
	versions := []string{"10.0", "12.0", "13.0", "14.0"}

	allJobMetrics := collectJobMetricsForAllVersions(t, versions)
	allStorageMetrics := collectStorageMetricsForAllVersions(t, versions)

	verifyJobMetricConsistency(t, allJobMetrics)
	verifyStorageMetricConsistency(t, allStorageMetrics)
}

// collectJobMetricsForAllVersions collects job metrics for all API versions
func collectJobMetricsForAllVersions(t *testing.T, versions []string) map[string]map[string]float64 {
	t.Helper()

	allJobMetrics := make(map[string]map[string]float64)

	for _, version := range versions {
		versionSuffix := getVersionSuffix(version)
		jobsFile := fmt.Sprintf(testDataPathFormat, "jobs", versionSuffix)

		jobsServer := createMockServerWithFile(t, jobsFile, version)
		jobsCfg := createTestConfig(jobsServer.URL, version)
		jobsClient := NewNbuClient(jobsCfg)

		_, jobsCountSlice, _, err := FetchAllJobs(context.Background(), jobsClient, "5m")
		if err != nil {
			t.Fatalf(fetchJobsErrorFormat, version, err)
		}

		allJobMetrics[version] = jobMetricSliceToMap(jobsCountSlice)
		jobsServer.Close()
	}

	return allJobMetrics
}

// collectStorageMetricsForAllVersions collects storage metrics for all API versions
func collectStorageMetricsForAllVersions(t *testing.T, versions []string) map[string]map[string]float64 {
	t.Helper()

	allStorageMetrics := make(map[string]map[string]float64)

	for _, version := range versions {
		versionSuffix := getVersionSuffix(version)
		storageFile := fmt.Sprintf(testDataPathFormat, "storage", versionSuffix)

		storageServer := createMockServerWithFile(t, storageFile, version)
		storageCfg := createTestConfig(storageServer.URL, version)
		storageClient := NewNbuClient(storageCfg)

		storageSlice, err := FetchStorage(context.Background(), storageClient)
		if err != nil {
			t.Fatalf(fetchStorageErrorFormat, version, err)
		}

		allStorageMetrics[version] = storageMetricSliceToMap(storageSlice)
		storageServer.Close()
	}

	return allStorageMetrics
}

// getVersionSuffix extracts the major version number from a version string
func getVersionSuffix(version string) string {
	switch version {
	case "10.0":
		return "10"
	case "12.0":
		return "12"
	case "13.0":
		return "13"
	case "14.0":
		return "14"
	default:
		return strings.Split(version, ".")[0]
	}
}

// verifyJobMetricConsistency verifies job metrics are consistent across versions
func verifyJobMetricConsistency(t *testing.T, allJobMetrics map[string]map[string]float64) {
	t.Helper()

	baseJobKeys := make(map[string]bool)
	for key := range allJobMetrics["10.0"] {
		baseJobKeys[key] = true
	}

	for version, metrics := range allJobMetrics {
		if version == "10.0" {
			continue
		}
		for key := range baseJobKeys {
			if _, exists := metrics[key]; !exists {
				t.Errorf("Job metric key %s missing in version %s", key, version)
			}
		}
	}
}

// verifyStorageMetricConsistency verifies storage metrics are consistent across versions
func verifyStorageMetricConsistency(t *testing.T, allStorageMetrics map[string]map[string]float64) {
	t.Helper()

	baseStorageKeys := make(map[string]bool)
	for key := range allStorageMetrics["10.0"] {
		baseStorageKeys[key] = true
	}

	for version, metrics := range allStorageMetrics {
		if version == "10.0" {
			continue
		}
		for key := range baseStorageKeys {
			if _, exists := metrics[key]; !exists {
				t.Errorf("Storage metric key %s missing in version %s", key, version)
			}
		}
	}
}

// TestAuthenticationWithAllVersions tests that authentication works with all API versions
func TestAuthenticationWithAllVersions(t *testing.T) {
	versions := []string{"10.0", "12.0", "13.0", "14.0"}

	for _, version := range versions {
		t.Run(fmt.Sprintf(testAPIVersionFormat, version), func(t *testing.T) {
			authHeaderReceived := ""

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				authHeaderReceived = r.Header.Get(authorizationHeader)

				// Verify correct API version header
				acceptHdr := r.Header.Get(acceptHeader)
				expectedAccept := fmt.Sprintf(contentTypeNetBackupJSONFormat, version)
				if acceptHdr != expectedAccept {
					w.WriteHeader(http.StatusNotAcceptable)
					return
				}

				// Return minimal valid response
				response := createMinimalJobsResponse()
				w.Header().Set(contentTypeHeader, fmt.Sprintf(contentTypeNetBackupJSONFormat, version))
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()

			cfg := createTestConfig(server.URL, version)
			client := NewNbuClient(cfg)

			_, _, _, err := FetchAllJobs(context.Background(), client, "5m")
			if err != nil {
				t.Fatalf("FetchAllJobs failed for version %s: %v", version, err)
			}

			// Verify API key was sent
			if authHeaderReceived != "test-api-key" {
				t.Errorf("Expected Authorization header 'test-api-key', got '%s'", authHeaderReceived)
			}
		})
	}
}

// TestParsingWithRealResponseFiles tests parsing with actual response files
func TestParsingWithRealResponseFiles(t *testing.T) {
	tests := []struct {
		name        string
		version     string
		jobsFile    string
		storageFile string
	}{
		{
			name:        "NetBackup 10.0 (API v10.0)",
			version:     "10.0",
			jobsFile:    "../../testdata/api-versions/jobs-response-v10.json",
			storageFile: "../../testdata/api-versions/storage-response-v10.json",
		},
		{
			name:        "NetBackup 10.5 (API v12.0)",
			version:     "12.0",
			jobsFile:    "../../testdata/api-versions/jobs-response-v12.json",
			storageFile: "../../testdata/api-versions/storage-response-v12.json",
		},
		{
			name:        "NetBackup 11.0 (API v13.0)",
			version:     "13.0",
			jobsFile:    "../../testdata/api-versions/jobs-response-v13.json",
			storageFile: "../../testdata/api-versions/storage-response-v13.json",
		},
		{
			name:        "NetBackup 11.2 (API v14.0)",
			version:     "14.0",
			jobsFile:    "../../testdata/api-versions/jobs-response-v14.json",
			storageFile: "../../testdata/api-versions/storage-response-v14.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Run("Jobs", func(t *testing.T) {
				testJobsParsing(t, tt.jobsFile)
			})

			t.Run("Storage", func(t *testing.T) {
				testStorageParsing(t, tt.storageFile)
			})
		})
	}
}

// testJobsParsing tests parsing of jobs response files
func testJobsParsing(t *testing.T, jobsFile string) {
	t.Helper()

	data := loadTestDataFromFile(t, jobsFile)
	jobs := unmarshalJobsResponse(t, data)

	if len(jobs.Data) == 0 {
		t.Error("No jobs data parsed")
	}

	verifyJobsFields(t, jobs)
}

// unmarshalJobsResponse unmarshals jobs response data
func unmarshalJobsResponse(t *testing.T, data []byte) models.Jobs {
	t.Helper()

	var jobs models.Jobs
	err := json.Unmarshal(data, &jobs)
	if err != nil {
		t.Fatalf("Failed to unmarshal jobs response: %v", err)
	}
	return jobs
}

// verifyJobsFields verifies that common job fields are present
func verifyJobsFields(t *testing.T, jobs models.Jobs) {
	t.Helper()

	for i, job := range jobs.Data {
		if job.Attributes.JobID == 0 {
			t.Errorf("Job %d: JobID is zero", i)
		}
		if job.Attributes.JobType == "" {
			t.Errorf("Job %d: JobType is empty", i)
		}
		if job.Attributes.PolicyType == "" {
			t.Errorf("Job %d: PolicyType is empty", i)
		}
		if job.Attributes.ClientName == "" {
			t.Errorf("Job %d: ClientName is empty", i)
		}
	}
}

// testStorageParsing tests parsing of storage response files
func testStorageParsing(t *testing.T, storageFile string) {
	t.Helper()

	data := loadTestDataFromFile(t, storageFile)
	storage := unmarshalStorageResponse(t, data)

	if len(storage.Data) == 0 {
		t.Error("No storage data parsed")
	}

	verifyStorageFields(t, storage)
}

// unmarshalStorageResponse unmarshals storage response data
func unmarshalStorageResponse(t *testing.T, data []byte) models.Storages {
	t.Helper()

	var storage models.Storages
	err := json.Unmarshal(data, &storage)
	if err != nil {
		t.Fatalf("Failed to unmarshal storage response: %v", err)
	}
	return storage
}

// verifyStorageFields verifies that common storage fields are present
func verifyStorageFields(t *testing.T, storage models.Storages) {
	t.Helper()

	for i, stu := range storage.Data {
		if stu.Attributes.Name == "" {
			t.Errorf("Storage unit %d: Name is empty", i)
		}
		if stu.Attributes.StorageType == "" {
			t.Errorf("Storage unit %d: StorageType is empty", i)
		}
		// Capacity fields can be zero for tape, so just check they exist
		_ = stu.Attributes.FreeCapacityBytes
		_ = stu.Attributes.UsedCapacityBytes
		_ = stu.Attributes.TotalCapacityBytes
	}
}

// TestErrorHandlingAcrossVersions tests error handling with different API versions
// Note: Disables retries to avoid slow exponential backoff on 500 errors
func TestErrorHandlingAcrossVersions(t *testing.T) {
	versions := []string{"10.0", "12.0", "13.0", "14.0"}

	for _, version := range versions {
		t.Run(fmt.Sprintf("API_v%s", version), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Return 500 Internal Server Error
				w.WriteHeader(http.StatusInternalServerError)
				errorResponse := map[string]interface{}{
					"errorCode":    500,
					"errorMessage": "Internal server error",
				}
				_ = json.NewEncoder(w).Encode(errorResponse)
			}))
			defer server.Close()

			cfg := createTestConfig(server.URL, version)
			client := NewNbuClient(cfg)
			// Disable retries for this test to avoid slow exponential backoff
			client.client.SetRetryCount(0)

			_, _, _, err := FetchAllJobs(context.Background(), client, "5m")

			// Should handle error gracefully
			if err == nil {
				t.Error("Expected error for 500 response, got nil")
			}
		})
	}
}

// Helper function to load test data from JSON files
func loadTestDataFromFile(t *testing.T, filename string) []byte {
	t.Helper()
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to read test data file %s: %v", filename, err)
	}
	return data
}

// Helper function to create a mock server that returns data from a file
func createMockServerWithFile(t *testing.T, filename string, version string) *httptest.Server {
	t.Helper()
	data := loadTestDataFromFile(t, filename)

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !validateAPIVersionHeader(r, version) {
			w.WriteHeader(http.StatusNotAcceptable)
			return
		}

		if isPaginatedJobsRequest(r) {
			handlePaginatedJobsRequest(t, w, r, data, version)
		} else {
			handleFullDataResponse(w, data, version)
		}
	}))
}

// validateAPIVersionHeader checks if the Accept header matches the expected API version
func validateAPIVersionHeader(r *http.Request, version string) bool {
	acceptHeader := r.Header.Get(acceptHeader)
	expectedAccept := fmt.Sprintf(contentTypeNetBackupJSONFormat, version)
	return acceptHeader == expectedAccept
}

// isPaginatedJobsRequest checks if the request is for paginated jobs data
func isPaginatedJobsRequest(r *http.Request) bool {
	return strings.Contains(r.URL.Path, "/admin/jobs") && r.URL.Query().Get("page[limit]") != ""
}

// handlePaginatedJobsRequest handles paginated jobs API requests
func handlePaginatedJobsRequest(t *testing.T, w http.ResponseWriter, r *http.Request, data []byte, version string) {
	t.Helper()

	fullJobs := unmarshalJobsData(t, data)
	offset := parseCursorOffsetFromRequest(r)
	response := createPaginatedJobsResponse(fullJobs, offset)

	writeJSONResponse(w, response, version)
}

// unmarshalJobsData unmarshals the jobs data from bytes
func unmarshalJobsData(t *testing.T, data []byte) models.Jobs {
	t.Helper()

	var fullJobs models.Jobs
	if err := json.Unmarshal(data, &fullJobs); err != nil {
		t.Fatalf("Failed to unmarshal jobs data: %v", err)
	}
	return fullJobs
}

// parseCursorOffsetFromRequest extracts the pagination offset encoded in the cursor.
// The mock encodes cursors as "cursor_N" where N is the offset of the next page.
func parseCursorOffsetFromRequest(r *http.Request) int {
	cursor := r.URL.Query().Get("page[after]")
	offset := 0
	if cursor != "" {
		_, _ = fmt.Sscanf(cursor, "cursor_%d", &offset)
	}
	return offset
}

// createPaginatedJobsResponse creates a paginated response with a batch of jobs (up to page limit)
func createPaginatedJobsResponse(fullJobs models.Jobs, offset int) *models.Jobs {
	response := &models.Jobs{}
	response.Data = make([]models.JobData, 0)

	totalJobs := len(fullJobs.Data)
	if offset < totalJobs {
		// Return all remaining jobs (since test data is small, this returns the batch)
		for i := offset; i < totalJobs; i++ {
			response.Data = append(response.Data, fullJobs.Data[i])
		}
		setBatchPaginationMetadata(response, offset, len(response.Data), totalJobs)
	} else {
		setEmptyPaginationMetadata(response, totalJobs)
	}

	return response
}

// setBatchPaginationMetadata sets cursor pagination metadata for a batch response.
func setBatchPaginationMetadata(response *models.Jobs, offset int, batchSize int, totalJobs int) {
	nextOffset := offset + batchSize
	if nextOffset < totalJobs {
		response.Meta.Pagination.Next = fmt.Sprintf("cursor_%d", nextOffset)
	} else {
		response.Meta.Pagination.Next = ""
	}
}

// setEmptyPaginationMetadata sets pagination metadata when no more jobs are available.
func setEmptyPaginationMetadata(response *models.Jobs, _ int) {
	response.Meta.Pagination.Next = ""
}

// writeJSONResponse writes a JSON response with appropriate headers
func writeJSONResponse(w http.ResponseWriter, response interface{}, version string) {
	w.Header().Set(contentTypeHeader, fmt.Sprintf(contentTypeNetBackupJSONFormat, version))
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

// handleFullDataResponse handles non-paginated requests by returning full data.
// Re-encodes via json.NewEncoder to avoid direct w.Write on raw bytes.
func handleFullDataResponse(w http.ResponseWriter, data []byte, version string) {
	var payload json.RawMessage
	if err := json.Unmarshal(data, &payload); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set(contentTypeHeader, fmt.Sprintf(contentTypeNetBackupJSONFormat, version))
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(payload)
}

// createMinimalJobsResponse creates a minimal jobs response for testing
func createMinimalJobsResponse() *models.Jobs {
	response := &models.Jobs{}
	response.Data = make([]models.JobData, 0)
	response.Meta.Pagination.Next = ""

	return response
}
