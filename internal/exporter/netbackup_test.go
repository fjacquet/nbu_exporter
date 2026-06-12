package exporter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fjacquet/nbu_exporter/internal/models"
)

// TestFetchJobDetails_BatchProcessing verifies that FetchJobDetails processes all jobs in a batch response
func TestFetchJobDetails_BatchProcessing(t *testing.T) {
	callCount := 0
	jobsInBatch := 3

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		// Create a response with multiple jobs
		response := createJobsResponse(jobsInBatch, 0, jobsInBatch-1)

		w.Header().Set(contentTypeHeader, fmt.Sprintf(contentTypeNetBackupJSONFormat, "12.0"))
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := createTestConfig(server.URL, "12.0")
	client := NewNbuClient(cfg)

	// Initialize maps for metrics
	agg := NewJobAggregator()
	jobsSize := agg.Size
	jobsCount := agg.Count
	jobsStatusCount := agg.StatusCount

	startTime := time.Now().Add(-5 * time.Minute)

	nextOffset, err := FetchJobDetails(context.Background(), client, agg, 0, startTime)
	if err != nil {
		t.Fatalf("FetchJobDetails failed: %v", err)
	}

	// Verify only 1 API call was made
	if callCount != 1 {
		t.Errorf("Expected 1 API call, got %d", callCount)
	}

	// Verify all 3 jobs were counted (all have same key: BACKUP/STANDARD/0)
	expectedKey := JobMetricKey{Action: "BACKUP", PolicyType: "STANDARD", Status: "0"}
	if jobsCount[expectedKey] != float64(jobsInBatch) {
		t.Errorf("Expected %d jobs counted, got %v", jobsInBatch, jobsCount[expectedKey])
	}

	// Verify status count
	statusKey := JobStatusKey{Action: "BACKUP", Status: "0"}
	if jobsStatusCount[statusKey] != float64(jobsInBatch) {
		t.Errorf("Expected %d status count, got %v", jobsInBatch, jobsStatusCount[statusKey])
	}

	// Verify job sizes were accumulated (each job transfers 1024KB = 1MB)
	expectedSize := float64(jobsInBatch * 1024 * 1024) // 3 jobs * 1MB each
	if jobsSize[expectedKey] != expectedSize {
		t.Errorf("Expected job size %v, got %v", expectedSize, jobsSize[expectedKey])
	}

	// Since all jobs fit in one page, nextOffset should be -1 (end of pagination)
	if nextOffset != -1 {
		t.Errorf("Expected nextOffset -1 (end of pagination), got %d", nextOffset)
	}
}

// TestFetchJobDetails_BatchPagination verifies pagination works correctly with batch responses
func TestFetchJobDetails_BatchPagination(t *testing.T) {
	callCount := 0
	jobsPerPage := 100 // Simulate first page with 100 jobs
	jobsLastPage := 50 // Second page with 50 jobs
	totalJobs := jobsPerPage + jobsLastPage

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		var response *models.Jobs
		if callCount == 1 {
			// First page: 100 jobs, next offset = 100
			response = createJobsResponse(jobsPerPage, 0, totalJobs-1)
			response.Meta.Pagination.Next = jobsPerPage
			response.Meta.Pagination.Offset = 0
		} else {
			// Second page: 50 jobs, last page
			response = createJobsResponse(jobsLastPage, jobsPerPage, totalJobs-1)
			response.Meta.Pagination.Offset = response.Meta.Pagination.Last // Signal last page
		}

		w.Header().Set(contentTypeHeader, fmt.Sprintf(contentTypeNetBackupJSONFormat, "12.0"))
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := createTestConfig(server.URL, "12.0")
	client := NewNbuClient(cfg)

	// Use FetchAllJobs to verify full pagination flow
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, jobsCountSlice, _, err := FetchAllJobs(ctx, client, "5m")
	if err != nil {
		t.Fatalf("FetchAllJobs failed: %v", err)
	}

	// Verify exactly 2 API calls were made (2 pages)
	if callCount != 2 {
		t.Errorf("Expected 2 API calls for pagination, got %d", callCount)
	}

	// Verify total jobs collected
	totalCollected := 0
	for _, m := range jobsCountSlice {
		totalCollected += int(m.Value)
	}

	if totalCollected != totalJobs {
		t.Errorf("Expected %d total jobs, got %d", totalJobs, totalCollected)
	}
}

// TestFetchJobDetails_EmptyBatch verifies empty response handling
func TestFetchJobDetails_EmptyBatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return empty jobs response
		response := &models.Jobs{}
		response.Data = nil // No jobs
		response.Meta.Pagination.Offset = 0
		response.Meta.Pagination.Last = 0

		w.Header().Set(contentTypeHeader, fmt.Sprintf(contentTypeNetBackupJSONFormat, "12.0"))
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := createTestConfig(server.URL, "12.0")
	client := NewNbuClient(cfg)

	agg := NewJobAggregator()
	jobsCount := agg.Count
	jobsStatusCount := agg.StatusCount

	startTime := time.Now().Add(-5 * time.Minute)

	nextOffset, err := FetchJobDetails(context.Background(), client, agg, 0, startTime)
	if err != nil {
		t.Fatalf("FetchJobDetails failed: %v", err)
	}

	// Verify returns -1 (end of pagination) for empty response
	if nextOffset != -1 {
		t.Errorf("Expected nextOffset -1 for empty batch, got %d", nextOffset)
	}

	// Verify no panic with empty slice - maps should be empty
	if len(jobsCount) != 0 {
		t.Errorf("Expected empty jobsCount map, got %d entries", len(jobsCount))
	}

	if len(jobsStatusCount) != 0 {
		t.Errorf("Expected empty jobsStatusCount map, got %d entries", len(jobsStatusCount))
	}
}

// TestFetchJobDetails_MixedJobTypes verifies different job types are counted separately
func TestFetchJobDetails_MixedJobTypes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create response with different job types
		response := createMixedJobsResponse()

		w.Header().Set(contentTypeHeader, fmt.Sprintf(contentTypeNetBackupJSONFormat, "12.0"))
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := createTestConfig(server.URL, "12.0")
	client := NewNbuClient(cfg)

	agg := NewJobAggregator()
	jobsCount := agg.Count
	jobsStatusCount := agg.StatusCount

	startTime := time.Now().Add(-5 * time.Minute)

	_, err := FetchJobDetails(context.Background(), client, agg, 0, startTime)
	if err != nil {
		t.Fatalf("FetchJobDetails failed: %v", err)
	}

	// Verify BACKUP/VMWARE/0 job counted
	backupKey := JobMetricKey{Action: "BACKUP", PolicyType: "VMWARE", Status: "0"}
	if jobsCount[backupKey] != 1 {
		t.Errorf("Expected 1 BACKUP/VMWARE/0 job, got %v", jobsCount[backupKey])
	}

	// Verify RESTORE/STANDARD/1 job counted
	restoreKey := JobMetricKey{Action: "RESTORE", PolicyType: "STANDARD", Status: "1"}
	if jobsCount[restoreKey] != 1 {
		t.Errorf("Expected 1 RESTORE/STANDARD/1 job, got %v", jobsCount[restoreKey])
	}

	// Verify BACKUP/STANDARD/150 job counted
	failedKey := JobMetricKey{Action: "BACKUP", PolicyType: "STANDARD", Status: "150"}
	if jobsCount[failedKey] != 1 {
		t.Errorf("Expected 1 BACKUP/STANDARD/150 job, got %v", jobsCount[failedKey])
	}

	// Verify status counts are separate
	if jobsStatusCount[JobStatusKey{Action: "BACKUP", Status: "0"}] != 1 {
		t.Error("Expected 1 BACKUP/0 status count")
	}
	if jobsStatusCount[JobStatusKey{Action: "RESTORE", Status: "1"}] != 1 {
		t.Error("Expected 1 RESTORE/1 status count")
	}
	if jobsStatusCount[JobStatusKey{Action: "BACKUP", Status: "150"}] != 1 {
		t.Error("Expected 1 BACKUP/150 status count")
	}
}

// Helper functions for creating test responses

// createJobsResponse creates a jobs response with the specified number of identical jobs
func createJobsResponse(numJobs int, startID int, lastIndex int) *models.Jobs {
	response := &models.Jobs{}
	response.Data = make([]models.JobData, numJobs)

	for i := 0; i < numJobs; i++ {
		response.Data[i].Type = "job"
		response.Data[i].ID = fmt.Sprintf("job-%d", startID+i)
		response.Data[i].Attributes.JobID = 12345 + startID + i
		response.Data[i].Attributes.JobType = "BACKUP"
		response.Data[i].Attributes.PolicyType = "STANDARD"
		response.Data[i].Attributes.Status = 0
		response.Data[i].Attributes.KilobytesTransferred = 1024 // 1MB
	}

	response.Meta.Pagination.Offset = lastIndex
	response.Meta.Pagination.Last = lastIndex
	response.Meta.Pagination.Count = numJobs

	return response
}

// createMixedJobsResponse creates a response with different job types
func createMixedJobsResponse() *models.Jobs {
	response := &models.Jobs{}
	response.Data = make([]models.JobData, 3)

	// Job 1: BACKUP/VMWARE/0
	response.Data[0].Type = "job"
	response.Data[0].ID = "12345"
	response.Data[0].Attributes.JobID = 12345
	response.Data[0].Attributes.JobType = "BACKUP"
	response.Data[0].Attributes.PolicyType = "VMWARE"
	response.Data[0].Attributes.Status = 0
	response.Data[0].Attributes.KilobytesTransferred = 52428800

	// Job 2: RESTORE/STANDARD/1
	response.Data[1].Type = "job"
	response.Data[1].ID = "12346"
	response.Data[1].Attributes.JobID = 12346
	response.Data[1].Attributes.JobType = "RESTORE"
	response.Data[1].Attributes.PolicyType = "STANDARD"
	response.Data[1].Attributes.Status = 1
	response.Data[1].Attributes.KilobytesTransferred = 10485760

	// Job 3: BACKUP/STANDARD/150 (failed)
	response.Data[2].Type = "job"
	response.Data[2].ID = "12347"
	response.Data[2].Attributes.JobID = 12347
	response.Data[2].Attributes.JobType = "BACKUP"
	response.Data[2].Attributes.PolicyType = "STANDARD"
	response.Data[2].Attributes.Status = 150
	response.Data[2].Attributes.KilobytesTransferred = 0

	response.Meta.Pagination.Offset = 2
	response.Meta.Pagination.Last = 2
	response.Meta.Pagination.Count = 3

	return response
}
