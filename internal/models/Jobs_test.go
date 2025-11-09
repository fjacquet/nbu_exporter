package models

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestJobsUnmarshalJSONWithOptionalFields(t *testing.T) {
	// Read the test fixture with 10.5 API response
	data, err := os.ReadFile("../../testdata/api-10.5/jobs-response.json")
	if err != nil {
		t.Fatalf("Failed to read test fixture: %v", err)
	}

	var jobs Jobs
	err = json.Unmarshal(data, &jobs)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Verify we have the expected number of jobs
	if len(jobs.Data) != 3 {
		t.Errorf("Expected 3 jobs, got %d", len(jobs.Data))
	}

	// Test first job (12345) with kilobytesDataTransferred field
	job1 := jobs.Data[0]
	if job1.Attributes.JobID != 12345 {
		t.Errorf("Expected jobId 12345, got %d", job1.Attributes.JobID)
	}
	if job1.Attributes.JobType != "BACKUP" {
		t.Errorf("Expected jobType 'BACKUP', got '%s'", job1.Attributes.JobType)
	}
	if job1.Attributes.PolicyType != "VMWARE" {
		t.Errorf("Expected policyType 'VMWARE', got '%s'", job1.Attributes.PolicyType)
	}
	if job1.Attributes.Status != 0 {
		t.Errorf("Expected status 0, got %d", job1.Attributes.Status)
	}
	if job1.Attributes.KilobytesTransferred != 52428800 {
		t.Errorf("Expected kilobytesTransferred 52428800, got %d", job1.Attributes.KilobytesTransferred)
	}

	// Test new optional field kilobytesDataTransferred
	if job1.Attributes.KilobytesDataTransferred != 48234567 {
		t.Errorf("Expected kilobytesDataTransferred 48234567, got %d", job1.Attributes.KilobytesDataTransferred)
	}

	// Test second job (12346) with kilobytesDataTransferred field
	job2 := jobs.Data[1]
	if job2.Attributes.JobID != 12346 {
		t.Errorf("Expected jobId 12346, got %d", job2.Attributes.JobID)
	}
	if job2.Attributes.JobType != "RESTORE" {
		t.Errorf("Expected jobType 'RESTORE', got '%s'", job2.Attributes.JobType)
	}
	if job2.Attributes.Status != 1 {
		t.Errorf("Expected status 1, got %d", job2.Attributes.Status)
	}
	if job2.Attributes.KilobytesDataTransferred != 9876543 {
		t.Errorf("Expected kilobytesDataTransferred 9876543, got %d", job2.Attributes.KilobytesDataTransferred)
	}

	// Test third job (12347) without kilobytesDataTransferred field
	job3 := jobs.Data[2]
	if job3.Attributes.JobID != 12347 {
		t.Errorf("Expected jobId 12347, got %d", job3.Attributes.JobID)
	}
	if job3.Attributes.Status != 150 {
		t.Errorf("Expected status 150, got %d", job3.Attributes.Status)
	}
	// Verify optional field is zero when not present
	if job3.Attributes.KilobytesDataTransferred != 0 {
		t.Errorf("Expected kilobytesDataTransferred 0 when not present, got %d", job3.Attributes.KilobytesDataTransferred)
	}
}

func TestJobsUnmarshalJSONWithoutOptionalFields(t *testing.T) {
	// Test JSON without optional fields (backward compatibility)
	jsonData := `{
		"data": [{
			"type": "job",
			"id": "99999",
			"links": {
				"self": {"href": "/netbackup/admin/jobs/99999"},
				"file-lists": {"href": "/netbackup/admin/jobs/99999/file-lists"},
				"try-logs": {"href": "/netbackup/admin/jobs/99999/try-logs"}
			},
			"attributes": {
				"jobId": 99999,
				"parentJobId": 0,
				"activeProcessId": 0,
				"jobType": "BACKUP",
				"jobSubType": "FULL",
				"policyType": "STANDARD",
				"policyName": "Test-Policy",
				"scheduleType": "FULL",
				"scheduleName": "Full",
				"clientName": "test-client",
				"controlHost": "master",
				"jobOwner": "admin",
				"jobGroup": "default",
				"backupId": "test-backup",
				"sourceMediaId": "",
				"sourceStorageUnitName": "",
				"sourceMediaServerName": "",
				"destinationMediaId": "",
				"destinationStorageUnitName": "disk-pool",
				"destinationMediaServerName": "media-server",
				"dataMovement": "STANDARD",
				"streamNumber": 1,
				"copyNumber": 1,
				"priority": 9999,
				"compression": 0,
				"status": 0,
				"state": "DONE",
				"numberOfFiles": 100,
				"estimatedFiles": 100,
				"kilobytesTransferred": 1024000,
				"kilobytesToTransfer": 1024000,
				"transferRate": 10240,
				"percentComplete": 100,
				"restartable": 0,
				"suspendable": 0,
				"resumable": 0,
				"frozenImage": 0,
				"transportType": "LAN",
				"dedupRatio": 1.0,
				"currentOperation": 0,
				"robotName": "",
				"vaultName": "",
				"profileName": "",
				"sessionId": 0,
				"numberOfTapeToEject": 0,
				"submissionType": 0,
				"acceleratorOptimization": 0,
				"dumpHost": "",
				"instanceDatabaseName": "",
				"auditUserName": "admin",
				"auditDomainName": "DOMAIN",
				"auditDomainType": 0,
				"restoreBackupIDs": "",
				"startTime": "2024-11-08T10:00:00Z",
				"endTime": "2024-11-08T11:00:00Z",
				"activeTryStartTime": "2024-11-08T10:00:00Z",
				"lastUpdateTime": "2024-11-08T11:00:00Z",
				"initiatorId": "admin",
				"retentionLevel": 1,
				"try": 1,
				"cancellable": 0,
				"jobQueueReason": 0,
				"jobQueueResource": "",
				"elapsedTime": "PT1H",
				"offHostType": "STANDARD"
			}
		}],
		"meta": {
			"pagination": {
				"count": 1,
				"offset": 0,
				"limit": 100,
				"first": 0,
				"last": 0,
				"page": 1,
				"pages": 1,
				"next": 0
			}
		},
		"links": {
			"self": {"href": "/netbackup/admin/jobs"},
			"first": {"href": "/netbackup/admin/jobs"},
			"last": {"href": "/netbackup/admin/jobs"},
			"next": {"href": ""}
		}
	}`

	var jobs Jobs
	err := json.Unmarshal([]byte(jsonData), &jobs)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if len(jobs.Data) != 1 {
		t.Fatalf("Expected 1 job, got %d", len(jobs.Data))
	}

	job := jobs.Data[0]
	if job.Attributes.JobID != 99999 {
		t.Errorf("Expected jobId 99999, got %d", job.Attributes.JobID)
	}
	if job.Attributes.KilobytesTransferred != 1024000 {
		t.Errorf("Expected kilobytesTransferred 1024000, got %d", job.Attributes.KilobytesTransferred)
	}

	// Verify optional field has zero value when not present
	if job.Attributes.KilobytesDataTransferred != 0 {
		t.Errorf("Expected kilobytesDataTransferred 0 when not present, got %d", job.Attributes.KilobytesDataTransferred)
	}
}

func TestJobsPagination(t *testing.T) {
	// Read the test fixture
	data, err := os.ReadFile("../../testdata/api-10.5/jobs-response.json")
	if err != nil {
		t.Fatalf("Failed to read test fixture: %v", err)
	}

	var jobs Jobs
	err = json.Unmarshal(data, &jobs)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Verify pagination metadata
	if jobs.Meta.Pagination.Count != 3 {
		t.Errorf("Expected pagination count 3, got %d", jobs.Meta.Pagination.Count)
	}
	if jobs.Meta.Pagination.Offset != 0 {
		t.Errorf("Expected pagination offset 0, got %d", jobs.Meta.Pagination.Offset)
	}
	if jobs.Meta.Pagination.Limit != 100 {
		t.Errorf("Expected pagination limit 100, got %d", jobs.Meta.Pagination.Limit)
	}
	if jobs.Meta.Pagination.Pages != 1 {
		t.Errorf("Expected pagination pages 1, got %d", jobs.Meta.Pagination.Pages)
	}
	if jobs.Meta.Pagination.First != 0 {
		t.Errorf("Expected pagination first 0, got %d", jobs.Meta.Pagination.First)
	}
	if jobs.Meta.Pagination.Last != 2 {
		t.Errorf("Expected pagination last 2, got %d", jobs.Meta.Pagination.Last)
	}
	if jobs.Meta.Pagination.Next != 0 {
		t.Errorf("Expected pagination next 0, got %d", jobs.Meta.Pagination.Next)
	}

	// Verify links
	if jobs.Links.Self.Href == "" {
		t.Error("Expected self link to be present")
	}
	if jobs.Links.First.Href == "" {
		t.Error("Expected first link to be present")
	}
	if jobs.Links.Last.Href == "" {
		t.Error("Expected last link to be present")
	}
}

func TestJobsTimeFields(t *testing.T) {
	// Read the test fixture
	data, err := os.ReadFile("../../testdata/api-10.5/jobs-response.json")
	if err != nil {
		t.Fatalf("Failed to read test fixture: %v", err)
	}

	var jobs Jobs
	err = json.Unmarshal(data, &jobs)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Test time parsing for first job
	job1 := jobs.Data[0]
	expectedStart, _ := time.Parse(time.RFC3339, "2024-11-08T10:00:00Z")
	if !job1.Attributes.StartTime.Equal(expectedStart) {
		t.Errorf("Expected startTime %v, got %v", expectedStart, job1.Attributes.StartTime)
	}

	expectedEnd, _ := time.Parse(time.RFC3339, "2024-11-08T11:30:00Z")
	if !job1.Attributes.EndTime.Equal(expectedEnd) {
		t.Errorf("Expected endTime %v, got %v", expectedEnd, job1.Attributes.EndTime)
	}

	// Test active job with zero end time
	job2 := jobs.Data[1]
	if job2.Attributes.State != "ACTIVE" {
		t.Errorf("Expected state 'ACTIVE', got '%s'", job2.Attributes.State)
	}
	// End time should be zero value for active jobs
	zeroTime := time.Time{}
	if !job2.Attributes.EndTime.Equal(zeroTime) {
		t.Errorf("Expected zero endTime for active job, got %v", job2.Attributes.EndTime)
	}
}

func TestJobsRequiredFields(t *testing.T) {
	tests := []struct {
		name        string
		jsonData    string
		expectError bool
	}{
		{
			name: "all required fields present",
			jsonData: `{
				"data": [{
					"type": "job",
					"id": "1",
					"attributes": {
						"jobId": 1,
						"jobType": "BACKUP",
						"policyType": "STANDARD",
						"status": 0,
						"kilobytesTransferred": 1000,
						"startTime": "2024-11-08T10:00:00Z",
						"endTime": "2024-11-08T11:00:00Z"
					}
				}],
				"meta": {"pagination": {"count": 1}}
			}`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var jobs Jobs
			err := json.Unmarshal([]byte(tt.jsonData), &jobs)
			if (err != nil) != tt.expectError {
				t.Errorf("Unmarshal() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestJobsDifferentJobTypes(t *testing.T) {
	// Read the test fixture
	data, err := os.ReadFile("../../testdata/api-10.5/jobs-response.json")
	if err != nil {
		t.Fatalf("Failed to read test fixture: %v", err)
	}

	var jobs Jobs
	err = json.Unmarshal(data, &jobs)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Verify different job types are parsed correctly
	jobTypes := make(map[string]bool)
	for _, job := range jobs.Data {
		jobTypes[job.Attributes.JobType] = true
	}

	if !jobTypes["BACKUP"] {
		t.Error("Expected to find BACKUP job type")
	}
	if !jobTypes["RESTORE"] {
		t.Error("Expected to find RESTORE job type")
	}

	// Verify different policy types
	policyTypes := make(map[string]bool)
	for _, job := range jobs.Data {
		policyTypes[job.Attributes.PolicyType] = true
	}

	if !policyTypes["VMWARE"] {
		t.Error("Expected to find VMWARE policy type")
	}
	if !policyTypes["STANDARD"] {
		t.Error("Expected to find STANDARD policy type")
	}
}
