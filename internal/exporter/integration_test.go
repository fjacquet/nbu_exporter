package exporter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fjacquet/nbu_exporter/internal/models"
)

// TestStorageMetricsCollection tests storage metrics collection with 10.5 responses
func TestStorageMetricsCollection(t *testing.T) {
	storageResponse := loadStorageTestData(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify correct endpoint
		if !strings.Contains(r.URL.Path, "/storage/storage-units") {
			t.Errorf("Expected storage endpoint, got: %s", r.URL.Path)
		}

		// Verify API version header
		acceptHeader := r.Header.Get("Accept")
		expectedAccept := "application/vnd.netbackup+json;version=12.0"
		if acceptHeader != expectedAccept {
			t.Errorf("Accept header = %v, want %v", acceptHeader, expectedAccept)
		}

		w.Header().Set("Content-Type", "application/vnd.netbackup+json;version=12.0")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(storageResponse)
	}))
	defer server.Close()

	cfg := createTestConfig(server.URL, "12.0")
	client := NewNbuClient(cfg)

	storageMetrics := make(map[string]float64)
	err := FetchStorage(context.Background(), client, storageMetrics)
	if err != nil {
		t.Fatalf("FetchStorage failed: %v", err)
	}

	// Verify we got metrics for disk storage units (excluding tape)
	// Expected: 2 storage units (disk-pool-1 and cloud-stu-1), each with free and used metrics
	expectedMetricCount := 4 // 2 units * 2 metrics (free + used)
	if len(storageMetrics) != expectedMetricCount {
		t.Errorf("Expected %d storage metrics, got %d", expectedMetricCount, len(storageMetrics))
	}

	// Verify specific metrics exist
	diskPoolFreeKey := StorageMetricKey{Name: "disk-pool-1", Type: "MEDIA_SERVER", Size: "free"}.String()
	diskPoolUsedKey := StorageMetricKey{Name: "disk-pool-1", Type: "MEDIA_SERVER", Size: "used"}.String()

	if _, exists := storageMetrics[diskPoolFreeKey]; !exists {
		t.Errorf("Missing metric for disk-pool-1 free capacity")
	}

	if _, exists := storageMetrics[diskPoolUsedKey]; !exists {
		t.Errorf("Missing metric for disk-pool-1 used capacity")
	}

	// Verify tape storage is excluded
	tapeKey := StorageMetricKey{Name: "tape-stu-1", Type: "MEDIA_SERVER", Size: "free"}.String()
	if _, exists := storageMetrics[tapeKey]; exists {
		t.Error("Tape storage should be excluded from metrics")
	}

	// Verify metric values
	expectedFree := float64(5368709120000)
	if storageMetrics[diskPoolFreeKey] != expectedFree {
		t.Errorf("disk-pool-1 free capacity = %v, want %v", storageMetrics[diskPoolFreeKey], expectedFree)
	}
}

// TestJobMetricsCollection tests job metrics collection with 10.5 responses
func TestJobMetricsCollection(t *testing.T) {
	allJobs := loadJobsTestData(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify correct endpoint
		if !strings.Contains(r.URL.Path, "/admin/jobs") {
			t.Errorf("Expected jobs endpoint, got: %s", r.URL.Path)
		}

		// Verify API version header
		acceptHeader := r.Header.Get("Accept")
		expectedAccept := "application/vnd.netbackup+json;version=12.0"
		if acceptHeader != expectedAccept {
			t.Errorf("Accept header = %v, want %v", acceptHeader, expectedAccept)
		}

		// Verify filter parameter exists
		if !strings.Contains(r.URL.RawQuery, "filter") {
			t.Error("Expected filter parameter in query")
		}

		// FetchJobDetails expects limit=1, so return one job at a time
		offsetStr := r.URL.Query().Get("page[offset]")
		offsetInt := 0
		if offsetStr != "" {
			_, _ = fmt.Sscanf(offsetStr, "%d", &offsetInt)
		}

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
		}, 1)

		// Return one job based on offset
		if offsetInt < len(allJobs.Data) {
			response.Data[0] = allJobs.Data[offsetInt]
			response.Meta.Pagination.Offset = offsetInt
			response.Meta.Pagination.Last = len(allJobs.Data) - 1
			if offsetInt < len(allJobs.Data)-1 {
				response.Meta.Pagination.Next = offsetInt + 1
			} else {
				// Last page - set offset == last to stop pagination
				response.Meta.Pagination.Next = 0
			}
		} else {
			// No more jobs
			response.Data = response.Data[:0]
			response.Meta.Pagination.Offset = len(allJobs.Data)
			response.Meta.Pagination.Last = len(allJobs.Data) - 1
		}

		w.Header().Set("Content-Type", "application/vnd.netbackup+json;version=12.0")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := createTestConfig(server.URL, "12.0")
	client := NewNbuClient(cfg)

	jobsSize := make(map[string]float64)
	jobsCount := make(map[string]float64)
	jobsStatusCount := make(map[string]float64)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := FetchAllJobs(ctx, client, jobsSize, jobsCount, jobsStatusCount, "5m")
	if err != nil {
		t.Fatalf("FetchAllJobs failed: %v", err)
	}

	// Verify job metrics were collected
	if len(jobsCount) == 0 {
		t.Error("No job count metrics collected")
	}

	if len(jobsSize) == 0 {
		t.Error("No job size metrics collected")
	}

	if len(jobsStatusCount) == 0 {
		t.Error("No job status metrics collected")
	}

	// Verify specific job metrics
	// Job 12345: BACKUP, VMWARE, status 0
	backupKey := JobMetricKey{Action: "BACKUP", PolicyType: "VMWARE", Status: "0"}.String()
	if jobsCount[backupKey] != 1 {
		t.Errorf("Expected 1 BACKUP/VMWARE/0 job, got %v", jobsCount[backupKey])
	}

	// Verify job size calculation (kilobytes * 1024)
	expectedSize := float64(52428800 * 1024)
	if jobsSize[backupKey] != expectedSize {
		t.Errorf("Job size = %v, want %v", jobsSize[backupKey], expectedSize)
	}

	// Job 12346: RESTORE, STANDARD, status 1
	restoreKey := JobMetricKey{Action: "RESTORE", PolicyType: "STANDARD", Status: "1"}.String()
	if jobsCount[restoreKey] != 1 {
		t.Errorf("Expected 1 RESTORE/STANDARD/1 job, got %v", jobsCount[restoreKey])
	}

	// Job 12347: BACKUP, STANDARD, status 150 (failed)
	failedKey := JobMetricKey{Action: "BACKUP", PolicyType: "STANDARD", Status: "150"}.String()
	if jobsCount[failedKey] != 1 {
		t.Errorf("Expected 1 BACKUP/STANDARD/150 job, got %v", jobsCount[failedKey])
	}
}

// TestPaginationHandling tests that pagination works correctly
func TestPaginationHandling(t *testing.T) {
	callCount := 0
	expectedPages := 3

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		// Create a minimal jobs response
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
		}, 1)

		response.Data[0].Type = "job"
		response.Data[0].ID = fmt.Sprintf("job-%d", callCount)
		response.Data[0].Attributes.JobID = 12345 + callCount
		response.Data[0].Attributes.JobType = "BACKUP"
		response.Data[0].Attributes.PolicyType = "STANDARD"
		response.Data[0].Attributes.Status = 0
		response.Data[0].Attributes.KilobytesTransferred = 1024

		// Set pagination based on call count
		if callCount < expectedPages {
			response.Meta.Pagination.Next = callCount
			response.Meta.Pagination.Offset = callCount - 1
			response.Meta.Pagination.Last = expectedPages - 1
		} else {
			// Last page
			response.Meta.Pagination.Offset = expectedPages - 1
			response.Meta.Pagination.Last = expectedPages - 1
			response.Meta.Pagination.Next = 0
		}

		w.Header().Set("Content-Type", "application/vnd.netbackup+json;version=12.0")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := createTestConfig(server.URL, "12.0")
	client := NewNbuClient(cfg)

	jobsSize := make(map[string]float64)
	jobsCount := make(map[string]float64)
	jobsStatusCount := make(map[string]float64)

	err := FetchAllJobs(context.Background(), client, jobsSize, jobsCount, jobsStatusCount, "5m")
	if err != nil {
		t.Fatalf("FetchAllJobs failed: %v", err)
	}

	// Verify all pages were fetched
	if callCount != expectedPages {
		t.Errorf("Expected %d API calls for pagination, got %d", expectedPages, callCount)
	}

	// Verify we collected metrics from all pages
	totalJobs := 0
	for _, count := range jobsCount {
		totalJobs += int(count)
	}

	if totalJobs != expectedPages {
		t.Errorf("Expected %d total jobs, got %d", expectedPages, totalJobs)
	}
}

// TestFilteringByTime tests that time-based filtering works correctly
func TestFilteringByTime(t *testing.T) {
	filterReceived := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify filter parameter contains time filter
		filter := r.URL.Query().Get("filter")
		if strings.Contains(filter, "endTime") && strings.Contains(filter, "gt") {
			filterReceived = true
		}

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

		w.Header().Set("Content-Type", "application/vnd.netbackup+json;version=12.0")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := createTestConfig(server.URL, "12.0")
	client := NewNbuClient(cfg)

	jobsSize := make(map[string]float64)
	jobsCount := make(map[string]float64)
	jobsStatusCount := make(map[string]float64)

	err := FetchAllJobs(context.Background(), client, jobsSize, jobsCount, jobsStatusCount, "5m")
	if err != nil {
		t.Fatalf("FetchAllJobs failed: %v", err)
	}

	if !filterReceived {
		t.Error("Expected time-based filter in API request")
	}
}

// Helper functions

func loadStorageTestData(t *testing.T) *models.Storages {
	t.Helper()

	// Load from JSON file and unmarshal into the actual model
	storages := &models.Storages{}

	// Manually create test data matching the actual struct
	storages.Data = make([]struct {
		Links struct {
			Self struct {
				Href string `json:"href"`
			} `json:"self"`
		} `json:"links"`
		Type       string `json:"type"`
		ID         string `json:"id"`
		Attributes struct {
			Name                       string `json:"name"`
			StorageType                string `json:"storageType"`
			StorageSubType             string `json:"storageSubType"`
			StorageServerType          string `json:"storageServerType"`
			UseAnyAvailableMediaServer bool   `json:"useAnyAvailableMediaServer"`
			Accelerator                bool   `json:"accelerator"`
			InstantAccessEnabled       bool   `json:"instantAccessEnabled"`
			IsCloudSTU                 bool   `json:"isCloudSTU"`
			FreeCapacityBytes          int64  `json:"freeCapacityBytes"`
			TotalCapacityBytes         int64  `json:"totalCapacityBytes"`
			UsedCapacityBytes          int64  `json:"usedCapacityBytes"`
			MaxFragmentSizeMegabytes   int    `json:"maxFragmentSizeMegabytes"`
			MaxConcurrentJobs          int    `json:"maxConcurrentJobs"`
			OnDemandOnly               bool   `json:"onDemandOnly"`
			StorageCategory            string `json:"storageCategory,omitempty"`
			ReplicationCapable         bool   `json:"replicationCapable,omitempty"`
			ReplicationSourceCapable   bool   `json:"replicationSourceCapable,omitempty"`
			ReplicationTargetCapable   bool   `json:"replicationTargetCapable,omitempty"`
			Snapshot                   bool   `json:"snapshot,omitempty"`
			Mirror                     bool   `json:"mirror,omitempty"`
			Independent                bool   `json:"independent,omitempty"`
			Primary                    bool   `json:"primary,omitempty"`
			ScaleOutEnabled            bool   `json:"scaleOutEnabled,omitempty"`
			WormCapable                bool   `json:"wormCapable,omitempty"`
			UseWorm                    bool   `json:"useWorm,omitempty"`
		} `json:"attributes,omitempty"`
		Relationships struct {
			DiskPool struct {
				Links struct {
					Related struct {
						Href string `json:"href"`
					} `json:"related"`
				} `json:"links"`
				Data struct {
					Type string `json:"type"`
					ID   string `json:"id"`
				} `json:"data"`
			} `json:"diskPool"`
		} `json:"relationships"`
	}, 3)

	// Disk pool 1
	storages.Data[0].Type = "storageUnit"
	storages.Data[0].ID = "disk-pool-1"
	storages.Data[0].Attributes.Name = "disk-pool-1"
	storages.Data[0].Attributes.StorageType = "DISK"
	storages.Data[0].Attributes.StorageServerType = "MEDIA_SERVER"
	storages.Data[0].Attributes.FreeCapacityBytes = 5368709120000
	storages.Data[0].Attributes.UsedCapacityBytes = 5368709120000
	storages.Data[0].Attributes.StorageCategory = "PRIMARY"
	storages.Data[0].Attributes.ReplicationCapable = true

	// Cloud storage
	storages.Data[1].Type = "storageUnit"
	storages.Data[1].ID = "cloud-stu-1"
	storages.Data[1].Attributes.Name = "cloud-stu-1"
	storages.Data[1].Attributes.StorageType = "CLOUD"
	storages.Data[1].Attributes.StorageServerType = "MEDIA_SERVER"
	storages.Data[1].Attributes.FreeCapacityBytes = 107374182400000
	storages.Data[1].Attributes.UsedCapacityBytes = 107374182400000
	storages.Data[1].Attributes.StorageCategory = "CLOUD"
	storages.Data[1].Attributes.ReplicationCapable = true

	// Tape storage (should be excluded)
	storages.Data[2].Type = "storageUnit"
	storages.Data[2].ID = "tape-stu-1"
	storages.Data[2].Attributes.Name = "tape-stu-1"
	storages.Data[2].Attributes.StorageType = "Tape"
	storages.Data[2].Attributes.StorageServerType = "MEDIA_SERVER"
	storages.Data[2].Attributes.FreeCapacityBytes = 0
	storages.Data[2].Attributes.UsedCapacityBytes = 0

	// Set pagination
	storages.Meta.Pagination.Count = 3
	storages.Meta.Pagination.Offset = 0
	storages.Meta.Pagination.Limit = 100
	storages.Meta.Pagination.First = 0
	storages.Meta.Pagination.Last = 2

	return storages
}

func loadJobsTestData(t *testing.T) *models.Jobs {
	t.Helper()

	jobs := &models.Jobs{}
	jobs.Data = make([]struct {
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
	}, 3)

	// Job 1: BACKUP/VMWARE/status 0 (success)
	jobs.Data[0].Type = "job"
	jobs.Data[0].ID = "12345"
	jobs.Data[0].Attributes.JobID = 12345
	jobs.Data[0].Attributes.JobType = "BACKUP"
	jobs.Data[0].Attributes.PolicyType = "VMWARE"
	jobs.Data[0].Attributes.Status = 0
	jobs.Data[0].Attributes.KilobytesTransferred = 52428800

	// Job 2: RESTORE/STANDARD/status 1 (active)
	jobs.Data[1].Type = "job"
	jobs.Data[1].ID = "12346"
	jobs.Data[1].Attributes.JobID = 12346
	jobs.Data[1].Attributes.JobType = "RESTORE"
	jobs.Data[1].Attributes.PolicyType = "STANDARD"
	jobs.Data[1].Attributes.Status = 1
	jobs.Data[1].Attributes.KilobytesTransferred = 10485760

	// Job 3: BACKUP/STANDARD/status 150 (failed)
	jobs.Data[2].Type = "job"
	jobs.Data[2].ID = "12347"
	jobs.Data[2].Attributes.JobID = 12347
	jobs.Data[2].Attributes.JobType = "BACKUP"
	jobs.Data[2].Attributes.PolicyType = "STANDARD"
	jobs.Data[2].Attributes.Status = 150
	jobs.Data[2].Attributes.KilobytesTransferred = 0

	// Set pagination
	jobs.Meta.Pagination.Count = 3
	jobs.Meta.Pagination.Offset = 0
	jobs.Meta.Pagination.Limit = 100
	jobs.Meta.Pagination.First = 0
	jobs.Meta.Pagination.Last = 2
	jobs.Meta.Pagination.Next = 0

	return jobs
}

func createTestConfig(serverURL, apiVersion string) models.Config {
	// Parse the test server URL to extract host and port
	// serverURL format: http://127.0.0.1:12345
	parts := strings.Split(strings.TrimPrefix(serverURL, "http://"), ":")
	host := parts[0]
	port := ""
	if len(parts) > 1 {
		port = parts[1]
	}

	cfg := models.Config{}
	cfg.Server.Host = "localhost"
	cfg.Server.Port = "2112"
	cfg.Server.URI = "/metrics"
	cfg.Server.ScrapingInterval = "5m"
	cfg.Server.LogName = "test.log"

	cfg.NbuServer.Host = host
	cfg.NbuServer.Port = port
	cfg.NbuServer.Scheme = "http"
	cfg.NbuServer.URI = ""
	cfg.NbuServer.APIKey = "test-api-key"
	cfg.NbuServer.APIVersion = apiVersion
	cfg.NbuServer.ContentType = "application/json"
	cfg.NbuServer.InsecureSkipVerify = true

	return cfg
}
