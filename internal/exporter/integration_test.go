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
		_ = json.NewEncoder(w).Encode(storageResponse)
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
		_ = json.NewEncoder(w).Encode(response)
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
		_ = json.NewEncoder(w).Encode(response)
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
		_ = json.NewEncoder(w).Encode(response)
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
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Extract version from Accept header
				acceptHeader := r.Header.Get("Accept")
				requestedVersion := ""
				if strings.Contains(acceptHeader, "version=13.0") {
					requestedVersion = "13.0"
				} else if strings.Contains(acceptHeader, "version=12.0") {
					requestedVersion = "12.0"
				} else if strings.Contains(acceptHeader, "version=3.0") {
					requestedVersion = "3.0"
				}

				// Check if this version is supported
				supported := false
				for _, v := range tt.supportedVersions {
					if v == requestedVersion {
						supported = true
						break
					}
				}

				if supported {
					// Return success with minimal job response
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

					w.Header().Set("Content-Type", fmt.Sprintf("application/vnd.netbackup+json;version=%s", requestedVersion))
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode(response)
				} else {
					// Return 406 Not Acceptable
					w.WriteHeader(http.StatusNotAcceptable)
					errorResponse := map[string]interface{}{
						"errorCode":    406,
						"errorMessage": fmt.Sprintf("API version %s is not supported", requestedVersion),
					}
					_ = json.NewEncoder(w).Encode(errorResponse)
				}
			}))
			defer server.Close()

			cfg := createTestConfig(server.URL, "") // Empty version to trigger detection
			client := NewNbuClient(cfg)

			detector := NewAPIVersionDetector(client, &cfg)
			detectedVersion, err := detector.DetectVersion(context.Background())

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				if detectedVersion != tt.expectedVersion {
					t.Errorf("Expected version %s, got %s", tt.expectedVersion, detectedVersion)
				}
			}
		})
	}
}

// TestFallbackBehavior tests the version fallback logic in detail
func TestFallbackBehavior(t *testing.T) {
	attemptedVersions := []string{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		acceptHeader := r.Header.Get("Accept")
		requestedVersion := ""
		if strings.Contains(acceptHeader, "version=13.0") {
			requestedVersion = "13.0"
		} else if strings.Contains(acceptHeader, "version=12.0") {
			requestedVersion = "12.0"
		} else if strings.Contains(acceptHeader, "version=3.0") {
			requestedVersion = "3.0"
		}

		attemptedVersions = append(attemptedVersions, requestedVersion)

		// Only v3.0 works
		if requestedVersion == "3.0" {
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

			w.Header().Set("Content-Type", "application/vnd.netbackup+json;version=3.0")
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

	detector := NewAPIVersionDetector(client, &cfg)
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
		if strings.Contains(acceptHeader, "version=13.0") {
			detectionAttempted = true
		}

		// Only accept v12.0
		if strings.Contains(acceptHeader, "version=12.0") {
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

	// Parse the data to handle pagination for jobs endpoint
	var fullResponse interface{}
	if err := json.Unmarshal(data, &fullResponse); err != nil {
		t.Fatalf("Failed to unmarshal test data: %v", err)
	}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		acceptHeader := r.Header.Get("Accept")
		expectedAccept := fmt.Sprintf("application/vnd.netbackup+json;version=%s", version)

		if acceptHeader != expectedAccept {
			w.WriteHeader(http.StatusNotAcceptable)
			return
		}

		// Check if this is a jobs endpoint with pagination (limit=1)
		if strings.Contains(r.URL.Path, "/admin/jobs") && r.URL.Query().Get("page[limit]") == "1" {
			// Handle paginated jobs request
			var fullJobs models.Jobs
			if err := json.Unmarshal(data, &fullJobs); err != nil {
				t.Fatalf("Failed to unmarshal jobs data: %v", err)
				return
			}

			offsetStr := r.URL.Query().Get("page[offset]")
			offset := 0
			if offsetStr != "" {
				_, _ = fmt.Sscanf(offsetStr, "%d", &offset)
			}

			// Create response with single job
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
				// Return one job
				response.Data = append(response.Data, fullJobs.Data[offset])
				response.Meta.Pagination.Offset = offset
				response.Meta.Pagination.Last = len(fullJobs.Data) - 1
				if offset < len(fullJobs.Data)-1 {
					response.Meta.Pagination.Next = offset + 1
				} else {
					// Last page - set offset == last to stop pagination
					response.Meta.Pagination.Next = 0
				}
			} else {
				// No more jobs
				response.Meta.Pagination.Offset = len(fullJobs.Data)
				response.Meta.Pagination.Last = len(fullJobs.Data) - 1
			}

			w.Header().Set("Content-Type", fmt.Sprintf("application/vnd.netbackup+json;version=%s", version))
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(response)
		} else {
			// Return full response for storage or non-paginated requests
			w.Header().Set("Content-Type", fmt.Sprintf("application/vnd.netbackup+json;version=%s", version))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(data)
		}
	}))
}

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
		t.Run(fmt.Sprintf("API_v%s", v.version), func(t *testing.T) {
			server := createMockServerWithFile(t, v.filename, v.version)
			defer server.Close()

			cfg := createTestConfig(server.URL, v.version)
			client := NewNbuClient(cfg)

			jobsSize := make(map[string]float64)
			jobsCount := make(map[string]float64)
			jobsStatusCount := make(map[string]float64)

			err := FetchAllJobs(context.Background(), client, jobsSize, jobsCount, jobsStatusCount, "5m")
			if err != nil {
				t.Fatalf("FetchAllJobs failed for version %s: %v", v.version, err)
			}

			// Verify common fields are parsed correctly
			if len(jobsCount) == 0 {
				t.Error("No job count metrics collected")
			}

			// Verify specific job metrics exist (based on test data)
			// Job 1: BACKUP/VMWARE/status 0
			backupKey := JobMetricKey{Action: "BACKUP", PolicyType: "VMWARE", Status: "0"}.String()
			if jobsCount[backupKey] != 1 {
				t.Errorf("Expected 1 BACKUP/VMWARE/0 job, got %v", jobsCount[backupKey])
			}

			// Job 2: RESTORE/STANDARD/status 1
			restoreKey := JobMetricKey{Action: "RESTORE", PolicyType: "STANDARD", Status: "1"}.String()
			if jobsCount[restoreKey] != 1 {
				t.Errorf("Expected 1 RESTORE/STANDARD/1 job, got %v", jobsCount[restoreKey])
			}

			// Verify job size calculation
			if jobsSize[backupKey] == 0 {
				t.Error("Job size should not be zero for successful backup")
			}
		})
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
		t.Run(fmt.Sprintf("API_v%s", v.version), func(t *testing.T) {
			server := createMockServerWithFile(t, v.filename, v.version)
			defer server.Close()

			cfg := createTestConfig(server.URL, v.version)
			client := NewNbuClient(cfg)

			storageMetrics := make(map[string]float64)
			err := FetchStorage(context.Background(), client, storageMetrics)
			if err != nil {
				t.Fatalf("FetchStorage failed for version %s: %v", v.version, err)
			}

			// Verify we got metrics for disk storage units
			if len(storageMetrics) == 0 {
				t.Error("No storage metrics collected")
			}

			// Verify specific metrics exist (based on test data)
			diskPoolFreeKey := StorageMetricKey{Name: "disk-pool-1", Type: "MEDIA_SERVER", Size: "free"}.String()
			diskPoolUsedKey := StorageMetricKey{Name: "disk-pool-1", Type: "MEDIA_SERVER", Size: "used"}.String()

			if _, exists := storageMetrics[diskPoolFreeKey]; !exists {
				t.Errorf("Missing metric for disk-pool-1 free capacity in version %s", v.version)
			}

			if _, exists := storageMetrics[diskPoolUsedKey]; !exists {
				t.Errorf("Missing metric for disk-pool-1 used capacity in version %s", v.version)
			}

			// Verify tape storage is excluded
			tapeKey := StorageMetricKey{Name: "tape-stu-1", Type: "MEDIA_SERVER", Size: "free"}.String()
			if _, exists := storageMetrics[tapeKey]; exists {
				t.Errorf("Tape storage should be excluded from metrics in version %s", v.version)
			}
		})
	}
}

// TestMetricsConsistencyAcrossVersions verifies that metric names and labels remain consistent
func TestMetricsConsistencyAcrossVersions(t *testing.T) {
	versions := []string{"3.0", "12.0", "13.0"}

	// Collect metrics from all versions
	allJobMetrics := make(map[string]map[string]float64)     // version -> metrics
	allStorageMetrics := make(map[string]map[string]float64) // version -> metrics

	for _, version := range versions {
		// Jobs metrics
		var versionSuffix string
		switch version {
		case "3.0":
			versionSuffix = "3"
		case "12.0":
			versionSuffix = "12"
		case "13.0":
			versionSuffix = "13"
		default:
			versionSuffix = strings.Split(version, ".")[0]
		}
		jobsFile := fmt.Sprintf("../../testdata/api-versions/jobs-response-v%s.json", versionSuffix)
		jobsServer := createMockServerWithFile(t, jobsFile, version)

		jobsCfg := createTestConfig(jobsServer.URL, version)
		jobsClient := NewNbuClient(jobsCfg)

		jobsSize := make(map[string]float64)
		jobsCount := make(map[string]float64)
		jobsStatusCount := make(map[string]float64)

		err := FetchAllJobs(context.Background(), jobsClient, jobsSize, jobsCount, jobsStatusCount, "5m")
		if err != nil {
			t.Fatalf("FetchAllJobs failed for version %s: %v", version, err)
		}

		allJobMetrics[version] = jobsCount
		jobsServer.Close()

		// Storage metrics
		storageFile := fmt.Sprintf("../../testdata/api-versions/storage-response-v%s.json", versionSuffix)
		storageServer := createMockServerWithFile(t, storageFile, version)

		storageCfg := createTestConfig(storageServer.URL, version)
		storageClient := NewNbuClient(storageCfg)

		storageMetrics := make(map[string]float64)
		err = FetchStorage(context.Background(), storageClient, storageMetrics)
		if err != nil {
			t.Fatalf("FetchStorage failed for version %s: %v", version, err)
		}

		allStorageMetrics[version] = storageMetrics
		storageServer.Close()
	}

	// Verify job metric keys are consistent across versions
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

	// Verify storage metric keys are consistent across versions
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
		t.Run(fmt.Sprintf("API_v%s", version), func(t *testing.T) {
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
