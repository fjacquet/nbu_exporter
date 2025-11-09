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
	"time"

	"github.com/fjacquet/nbu_exporter/internal/models"
)

// Test constants
const (
	testAPIVersionFormat     = "API_v%s"
	fetchJobsErrorFormat     = "FetchAllJobs failed for version %s: %v"
	fetchStorageErrorFormat  = "FetchStorage failed for version %s: %v"
	contentTypeHeader        = "Content-Type"
	acceptHeader             = "Accept"
	authorizationHeader      = "Authorization"
	netbackupMediaTypeFormat = "application/vnd.netbackup+json;version=%s"
	testAPIKey               = "test-api-key"
	testDataPathFormat       = "../../testdata/api-versions/%s-response-v%s.json"
)

// TestJobsAPICompatibilityAcrossVersions tests jobs API with all three versions
func TestJobsAPICompatibilityAcrossVersions(t *testing.T) {
	versions := []struct {
		version  string
		filename string
	}{
		{"3.0", "../../testdata/api-versions/jobs-response-v3.json"},
		{"12.0", "../../testdata/api-versions/jobs-response-v12.json"},
		{"13.0", "../../testdata/api-versions/jobs-response-v13.json"},
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

	jobsSize, jobsCount, jobsStatusCount := createJobMetricMaps()

	err := FetchAllJobs(context.Background(), client, jobsSize, jobsCount, jobsStatusCount, "5m")
	if err != nil {
		t.Fatalf(fetchJobsErrorFormat, version, err)
	}

	verifyJobMetrics(t, jobsSize, jobsCount)
}

// createJobMetricMaps creates empty metric maps for job testing
func createJobMetricMaps() (map[string]float64, map[string]float64, map[string]float64) {
	return make(map[string]float64), make(map[string]float64), make(map[string]float64)
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
		{"3.0", "../../testdata/api-versions/storage-response-v3.json"},
		{"12.0", "../../testdata/api-versions/storage-response-v12.json"},
		{"13.0", "../../testdata/api-versions/storage-response-v13.json"},
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

	storageMetrics := make(map[string]float64)
	err := FetchStorage(context.Background(), client, storageMetrics)
	if err != nil {
		t.Fatalf(fetchStorageErrorFormat, version, err)
	}

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
	versions := []string{"3.0", "12.0", "13.0"}

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

		jobsSize, jobsCount, jobsStatusCount := createJobMetricMaps()

		err := FetchAllJobs(context.Background(), jobsClient, jobsSize, jobsCount, jobsStatusCount, "5m")
		if err != nil {
			t.Fatalf(fetchJobsErrorFormat, version, err)
		}

		allJobMetrics[version] = jobsCount
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

		storageMetrics := make(map[string]float64)
		err := FetchStorage(context.Background(), storageClient, storageMetrics)
		if err != nil {
			t.Fatalf(fetchStorageErrorFormat, version, err)
		}

		allStorageMetrics[version] = storageMetrics
		storageServer.Close()
	}

	return allStorageMetrics
}

// getVersionSuffix extracts the major version number from a version string
func getVersionSuffix(version string) string {
	switch version {
	case "3.0":
		return "3"
	case "12.0":
		return "12"
	case "13.0":
		return "13"
	default:
		return strings.Split(version, ".")[0]
	}
}

// verifyJobMetricConsistency verifies job metrics are consistent across versions
func verifyJobMetricConsistency(t *testing.T, allJobMetrics map[string]map[string]float64) {
	t.Helper()
	
	baseJobKeys := make(map[string]bool)
	for key := range allJobMetrics["3.0"] {
		baseJobKeys[key] = true
	}

	for version, metrics := range allJobMetrics {
		if version == "3.0" {
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
	for key := range allStorageMetrics["3.0"] {
		baseStorageKeys[key] = true
	}

	for version, metrics := range allStorageMetrics {
		if version == "3.0" {
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
	versions := []string{"3.0", "12.0", "13.0"}

	for _, version := range versions {
		t.Run(fmt.Sprintf(testAPIVersionFormat, version), func(t *testing.T) {
			authHeaderReceived := ""

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				authHeaderReceived = r.Header.Get("Authorization")

				// Verify correct API version header
				acceptHeader := r.Header.Get("Accept")
				expectedAccept := fmt.Sprintf("application/vnd.netbackup+json;version=%s", version)
				if acceptHeader != expectedAccept {
					w.WriteHeader(http.StatusNotAcceptable)
					return
				}

				// Return minimal valid response
				response := createMinimalJobsResponse()
				w.Header().Set("Content-Type", fmt.Sprintf("application/vnd.netbackup+json;version=%s", version))
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()

			cfg := createTestConfig(server.URL, version)
			client := NewNbuClient(cfg)

			jobsSize := make(map[string]float64)
			jobsCount := make(map[string]float64)
			jobsStatusCount := make(map[string]float64)

			err := FetchAllJobs(context.Background(), client, jobsSize, jobsCount, jobsStatusCount, "5m")
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
			name:        "NetBackup 10.0 (API v3.0)",
			version:     "3.0",
			jobsFile:    "../../testdata/api-versions/jobs-response-v3.json",
			storageFile: "../../testdata/api-versions/storage-response-v3.json",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test jobs parsing
			t.Run("Jobs", func(t *testing.T) {
				data := loadTestDataFromFile(t, tt.jobsFile)

				var jobs models.Jobs
				err := json.Unmarshal(data, &jobs)
				if err != nil {
					t.Fatalf("Failed to unmarshal jobs response: %v", err)
				}

				if len(jobs.Data) == 0 {
					t.Error("No jobs data parsed")
				}

				// Verify common fields are present
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
			})

			// Test storage parsing
			t.Run("Storage", func(t *testing.T) {
				data := loadTestDataFromFile(t, tt.storageFile)

				var storage models.Storages
				err := json.Unmarshal(data, &storage)
				if err != nil {
					t.Fatalf("Failed to unmarshal storage response: %v", err)
				}

				if len(storage.Data) == 0 {
					t.Error("No storage data parsed")
				}

				// Verify common fields are present
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
			})
		})
	}
}

// TestErrorHandlingAcrossVersions tests error handling with different API versions
func TestErrorHandlingAcrossVersions(t *testing.T) {
	versions := []string{"3.0", "12.0", "13.0"}

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

			jobsSize := make(map[string]float64)
			jobsCount := make(map[string]float64)
			jobsStatusCount := make(map[string]float64)

			err := FetchAllJobs(context.Background(), client, jobsSize, jobsCount, jobsStatusCount, "5m")

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
	acceptHeader := r.Header.Get("Accept")
	expectedAccept := fmt.Sprintf("application/vnd.netbackup+json;version=%s", version)
	return acceptHeader == expectedAccept
}

// isPaginatedJobsRequest checks if the request is for paginated jobs data
func isPaginatedJobsRequest(r *http.Request) bool {
	return strings.Contains(r.URL.Path, "/admin/jobs") && r.URL.Query().Get("page[limit]") == "1"
}

// handlePaginatedJobsRequest handles paginated jobs API requests
func handlePaginatedJobsRequest(t *testing.T, w http.ResponseWriter, r *http.Request, data []byte, version string) {
	t.Helper()
	
	fullJobs := unmarshalJobsData(t, data)
	offset := parseOffsetFromRequest(r)
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

// parseOffsetFromRequest extracts the pagination offset from the request
func parseOffsetFromRequest(r *http.Request) int {
	offsetStr := r.URL.Query().Get("page[offset]")
	offset := 0
	if offsetStr != "" {
		_, _ = fmt.Sscanf(offsetStr, "%d", &offset)
	}
	return offset
}

// createPaginatedJobsResponse creates a paginated response with a single job
func createPaginatedJobsResponse(fullJobs models.Jobs, offset int) *models.Jobs {
	response := &models.Jobs{}
	response.Data = make([]struct {
		Links struct {
			Self struct {
				Href string `json:"href"`
			} `json:"self"`
			FileLists struct {
				Href string `json:"href"`
			} `json:"file-lists"`
			TryLogs struct {
				Href string `json:"href"`
			} `json:"try-logs"`
		} `json:"links"`
		Type       string `json:"type"`
		ID         string `json:"id"`
		Attributes struct {
			JobID                      int       `json:"jobId"`
			ParentJobID                int       `json:"parentJobId"`
			ActiveProcessID            int       `json:"activeProcessId"`
			JobType                    string    `json:"jobType"`
			JobSubType                 string    `json:"jobSubType"`
			PolicyType                 string    `json:"policyType"`
			PolicyName                 string    `json:"policyName"`
			ScheduleType               string    `json:"scheduleType"`
			ScheduleName               string    `json:"scheduleName"`
			ClientName                 string    `json:"clientName"`
			ControlHost                string    `json:"controlHost"`
			JobOwner                   string    `json:"jobOwner"`
			JobGroup                   string    `json:"jobGroup"`
			BackupID                   string    `json:"backupId"`
			SourceMediaID              string    `json:"sourceMediaId"`
			SourceStorageUnitName      string    `json:"sourceStorageUnitName"`
			SourceMediaServerName      string    `json:"sourceMediaServerName"`
			DestinationMediaID         string    `json:"destinationMediaId"`
			DestinationStorageUnitName string    `json:"destinationStorageUnitName"`
			DestinationMediaServerName string    `json:"destinationMediaServerName"`
			DataMovement               string    `json:"dataMovement"`
			StreamNumber               int       `json:"streamNumber"`
			CopyNumber                 int       `json:"copyNumber"`
			Priority                   int       `json:"priority"`
			Compression                int       `json:"compression"`
			Status                     int       `json:"status"`
			State                      string    `json:"state"`
			NumberOfFiles              int       `json:"numberOfFiles"`
			EstimatedFiles             int       `json:"estimatedFiles"`
			KilobytesTransferred       int       `json:"kilobytesTransferred"`
			KilobytesToTransfer        int       `json:"kilobytesToTransfer"`
			TransferRate               int       `json:"transferRate"`
			PercentComplete            int       `json:"percentComplete"`
			Restartable                int       `json:"restartable"`
			Suspendable                int       `json:"suspendable"`
			Resumable                  int       `json:"resumable"`
			FrozenImage                int       `json:"frozenImage"`
			TransportType              string    `json:"transportType"`
			DedupRatio                 float64   `json:"dedupRatio"`
			CurrentOperation           int       `json:"currentOperation"`
			RobotName                  string    `json:"robotName"`
			VaultName                  string    `json:"vaultName"`
			ProfileName                string    `json:"profileName"`
			SessionID                  int       `json:"sessionId"`
			NumberOfTapeToEject        int       `json:"numberOfTapeToEject"`
			SubmissionType             int       `json:"submissionType"`
			AcceleratorOptimization    int       `json:"acceleratorOptimization"`
			DumpHost                   string    `json:"dumpHost"`
			InstanceDatabaseName       string    `json:"instanceDatabaseName"`
			AuditUserName              string    `json:"auditUserName"`
			AuditDomainName            string    `json:"auditDomainName"`
			AuditDomainType            int       `json:"auditDomainType"`
			RestoreBackupIDs           string    `json:"restoreBackupIDs"`
			StartTime                  time.Time `json:"startTime"`
			EndTime                    time.Time `json:"endTime"`
			ActiveTryStartTime         time.Time `json:"activeTryStartTime"`
			LastUpdateTime             time.Time `json:"lastUpdateTime"`
			InitiatorID                string    `json:"initiatorId"`
			RetentionLevel             int       `json:"retentionLevel"`
			Try                        int       `json:"try"`
			Cancellable                int       `json:"cancellable"`
			JobQueueReason             int       `json:"jobQueueReason"`
			JobQueueResource           string    `json:"jobQueueResource"`
			KilobytesDataTransferred   int       `json:"kilobytesDataTransferred,omitempty"`
			ElapsedTime                string    `json:"elapsedTime"`
			OffHostType                string    `json:"offHostType"`
		} `json:"attributes"`
	}, 0)

	if offset < len(fullJobs.Data) {
		response.Data = append(response.Data, fullJobs.Data[offset])
		setPaginationMetadata(response, offset, len(fullJobs.Data))
	} else {
		setEmptyPaginationMetadata(response, len(fullJobs.Data))
	}

	return response
}

// setPaginationMetadata sets pagination metadata for a valid page
func setPaginationMetadata(response *models.Jobs, offset int, totalJobs int) {
	response.Meta.Pagination.Offset = offset
	response.Meta.Pagination.Last = totalJobs - 1
	
	if offset < totalJobs-1 {
		response.Meta.Pagination.Next = offset + 1
	} else {
		response.Meta.Pagination.Next = 0
	}
}

// setEmptyPaginationMetadata sets pagination metadata when no more jobs are available
func setEmptyPaginationMetadata(response *models.Jobs, totalJobs int) {
	response.Meta.Pagination.Offset = totalJobs
	response.Meta.Pagination.Last = totalJobs - 1
}

// writeJSONResponse writes a JSON response with appropriate headers
func writeJSONResponse(w http.ResponseWriter, response interface{}, version string) {
	w.Header().Set("Content-Type", fmt.Sprintf("application/vnd.netbackup+json;version=%s", version))
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

// handleFullDataResponse handles non-paginated requests by returning full data
func handleFullDataResponse(w http.ResponseWriter, data []byte, version string) {
	w.Header().Set("Content-Type", fmt.Sprintf("application/vnd.netbackup+json;version=%s", version))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// createMinimalJobsResponse creates a minimal jobs response for testing
func createMinimalJobsResponse() *models.Jobs {
	response := &models.Jobs{}
	response.Data = make([]struct {
		Links struct {
			Self struct {
				Href string `json:"href"`
			} `json:"self"`
			FileLists struct {
				Href string `json:"href"`
			} `json:"file-lists"`
			TryLogs struct {
				Href string `json:"href"`
			} `json:"try-logs"`
		} `json:"links"`
		Type       string `json:"type"`
		ID         string `json:"id"`
		Attributes struct {
			JobID                      int       `json:"jobId"`
			ParentJobID                int       `json:"parentJobId"`
			ActiveProcessID            int       `json:"activeProcessId"`
			JobType                    string    `json:"jobType"`
			JobSubType                 string    `json:"jobSubType"`
			PolicyType                 string    `json:"policyType"`
			PolicyName                 string    `json:"policyName"`
			ScheduleType               string    `json:"scheduleType"`
			ScheduleName               string    `json:"scheduleName"`
			ClientName                 string    `json:"clientName"`
			ControlHost                string    `json:"controlHost"`
			JobOwner                   string    `json:"jobOwner"`
			JobGroup                   string    `json:"jobGroup"`
			BackupID                   string    `json:"backupId"`
			SourceMediaID              string    `json:"sourceMediaId"`
			SourceStorageUnitName      string    `json:"sourceStorageUnitName"`
			SourceMediaServerName      string    `json:"sourceMediaServerName"`
			DestinationMediaID         string    `json:"destinationMediaId"`
			DestinationStorageUnitName string    `json:"destinationStorageUnitName"`
			DestinationMediaServerName string    `json:"destinationMediaServerName"`
			DataMovement               string    `json:"dataMovement"`
			StreamNumber               int       `json:"streamNumber"`
			CopyNumber                 int       `json:"copyNumber"`
			Priority                   int       `json:"priority"`
			Compression                int       `json:"compression"`
			Status                     int       `json:"status"`
			State                      string    `json:"state"`
			NumberOfFiles              int       `json:"numberOfFiles"`
			EstimatedFiles             int       `json:"estimatedFiles"`
			KilobytesTransferred       int       `json:"kilobytesTransferred"`
			KilobytesToTransfer        int       `json:"kilobytesToTransfer"`
			TransferRate               int       `json:"transferRate"`
			PercentComplete            int       `json:"percentComplete"`
			Restartable                int       `json:"restartable"`
			Suspendable                int       `json:"suspendable"`
			Resumable                  int       `json:"resumable"`
			FrozenImage                int       `json:"frozenImage"`
			TransportType              string    `json:"transportType"`
			DedupRatio                 float64   `json:"dedupRatio"`
			CurrentOperation           int       `json:"currentOperation"`
			RobotName                  string    `json:"robotName"`
			VaultName                  string    `json:"vaultName"`
			ProfileName                string    `json:"profileName"`
			SessionID                  int       `json:"sessionId"`
			NumberOfTapeToEject        int       `json:"numberOfTapeToEject"`
			SubmissionType             int       `json:"submissionType"`
			AcceleratorOptimization    int       `json:"acceleratorOptimization"`
			DumpHost                   string    `json:"dumpHost"`
			InstanceDatabaseName       string    `json:"instanceDatabaseName"`
			AuditUserName              string    `json:"auditUserName"`
			AuditDomainName            string    `json:"auditDomainName"`
			AuditDomainType            int       `json:"auditDomainType"`
			RestoreBackupIDs           string    `json:"restoreBackupIDs"`
			StartTime                  time.Time `json:"startTime"`
			EndTime                    time.Time `json:"endTime"`
			ActiveTryStartTime         time.Time `json:"activeTryStartTime"`
			LastUpdateTime             time.Time `json:"lastUpdateTime"`
			InitiatorID                string    `json:"initiatorId"`
			RetentionLevel             int       `json:"retentionLevel"`
			Try                        int       `json:"try"`
			Cancellable                int       `json:"cancellable"`
			JobQueueReason             int       `json:"jobQueueReason"`
			JobQueueResource           string    `json:"jobQueueResource"`
			KilobytesDataTransferred   int       `json:"kilobytesDataTransferred,omitempty"`
			ElapsedTime                string    `json:"elapsedTime"`
			OffHostType                string    `json:"offHostType"`
		} `json:"attributes"`
	}, 0)
	response.Meta.Pagination.Offset = 0
	response.Meta.Pagination.Last = 0

	return response
}
