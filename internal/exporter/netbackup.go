// Package exporter provides data fetching and processing logic for NetBackup API endpoints.
// It handles pagination, metric aggregation, and data transformation for Prometheus metrics.
package exporter

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/fjacquet/nbu_exporter/internal/models"
	"github.com/fjacquet/nbu_exporter/internal/telemetry"
	"github.com/fjacquet/nbu_exporter/internal/utils"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	pageLimit        = "100"                    // Default page size for API pagination
	jobPageLimit     = "100"                    // Maximum allowed by NetBackup API for jobs endpoint
	storageTypeTape  = "Tape"                   // Storage type identifier for tape units (excluded from metrics)
	storagePath      = "/storage/storage-units" // API endpoint for storage unit data
	jobsPath         = "/admin/jobs"            // API endpoint for job data
	sizeTypeFree     = "free"                   // Metric dimension for free capacity
	sizeTypeUsed     = "used"                   // Metric dimension for used capacity
	bytesPerKilobyte = 1024                     // Conversion factor from kilobytes to bytes
	bytesPerMegabyte = 1024 * 1024              // Conversion factor from megabytes to bytes
	jobStateQueued   = "QUEUED"                 // Job state indicating the job is waiting for a resource

	// Pre-allocation capacity hints for job metrics.
	// Typical environments have 20-100 unique job type/policy/status combinations:
	//   - Job types: ~5-10 (BACKUP, RESTORE, VERIFY, etc.)
	//   - Policy types: ~5-20
	//   - Status codes: ~10-20 common values
	// These values are intentionally generous to avoid reallocations in most cases.
	expectedJobMetricKeys    = 100 // Capacity hint for job size/count maps
	expectedStatusMetricKeys = 50  // Capacity hint for job status count map
)

// FetchStorage retrieves storage unit information from the NetBackup API and returns
// typed metric values. It fetches all storage units and extracts free and used capacity
// metrics for non-tape storage.
//
// Tape storage units are excluded from metrics as they don't report meaningful capacity values.
//
// Parameters:
//   - ctx: Context for request cancellation and timeout
//   - client: NetBackup API client
//
// Returns a slice of StorageMetricValue and an error if the API request fails.
//
// FetchStorage is a backward-compatible wrapper over FetchStorageFull that
// returns only the free/used capacity metrics. New callers that also need
// per-unit attributes (capacity, concurrency, info) should use FetchStorageFull.
func FetchStorage(ctx context.Context, client *NbuClient) ([]StorageMetricValue, error) {
	metrics, _, err := FetchStorageFull(ctx, client)
	return metrics, err
}

// FetchStorageFull retrieves storage unit information from the NetBackup API and
// returns both the free/used capacity metrics and the per-unit attributes
// (StorageUnitInfo) derived from the same response.
//
// Tape storage units are excluded as they don't report meaningful capacity values.
func FetchStorageFull(ctx context.Context, client *NbuClient) ([]StorageMetricValue, []StorageUnitInfo, error) {
	// Start child span "netbackup.fetch_storage" from parent context
	ctx, span := client.tracing.StartSpan(ctx, "netbackup.fetch_storage", trace.SpanKindClient)
	defer span.End()

	var storages models.Storages

	url := client.cfg.BuildURL(storagePath, map[string]string{
		QueryParamLimit:  pageLimit,
		QueryParamOffset: "0",
	})

	if err := client.FetchData(ctx, url, &storages); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, nil, fmt.Errorf("failed to fetch storage data: %w", err)
	}

	var metrics []StorageMetricValue
	var units []StorageUnitInfo
	for _, data := range storages.Data {
		// Skip tape storage units
		if data.Attributes.StorageType == storageTypeTape {
			continue
		}

		attr := data.Attributes
		stuName := attr.Name
		stuType := attr.StorageServerType

		metrics = append(metrics, StorageMetricValue{
			Key:   StorageMetricKey{Name: stuName, Type: stuType, Size: sizeTypeFree},
			Value: float64(attr.FreeCapacityBytes),
		})
		metrics = append(metrics, StorageMetricValue{
			Key:   StorageMetricKey{Name: stuName, Type: stuType, Size: sizeTypeUsed},
			Value: float64(attr.UsedCapacityBytes),
		})

		units = append(units, StorageUnitInfo{
			Name:               stuName,
			Type:               stuType,
			SubType:            attr.StorageSubType,
			TotalCapacityBytes: float64(attr.TotalCapacityBytes),
			MaxConcurrentJobs:  float64(attr.MaxConcurrentJobs),
			MaxFragmentBytes:   float64(attr.MaxFragmentSizeMegabytes) * bytesPerMegabyte,
			IsCloud:            attr.IsCloudSTU,
			WormCapable:        attr.WormCapable,
			UseWorm:            attr.UseWorm,
			ReplicationCapable: attr.ReplicationCapable,
			InstantAccess:      attr.InstantAccessEnabled,
		})
	}

	// Batch span attributes for storage operation
	attrs := []attribute.KeyValue{
		attribute.String(telemetry.AttrNetBackupEndpoint, storagePath),
		attribute.Int(telemetry.AttrNetBackupStorageUnits, len(storages.Data)),
		attribute.String(telemetry.AttrNetBackupAPIVersion, client.cfg.NbuServer.APIVersion),
	}
	span.SetAttributes(attrs...)

	log.Debugf("Fetched storage data for %d storage units", len(storages.Data))
	return metrics, units, nil
}

// aggregateJob folds a single job's attributes into every job-derived metric.
// Centralizing the per-job logic keeps FetchJobDetails focused on pagination.
func aggregateJob(agg *JobAggregator, attr models.JobAttributes) {
	status := strconv.Itoa(attr.Status)
	jobKey := JobMetricKey{Action: attr.JobType, PolicyType: attr.PolicyType, Status: status}
	statusKey := JobStatusKey{Action: attr.JobType, Status: status}
	policyKey := JobPolicyKey{Action: attr.JobType, PolicyType: attr.PolicyType}

	// Existing metrics: bytes, count, status count.
	agg.Count[jobKey]++
	agg.StatusCount[statusKey]++
	agg.Size[jobKey] += float64(attr.KilobytesTransferred * bytesPerKilobyte)

	// State breakdown (default empty state to "UNKNOWN" to avoid an empty label).
	state := attr.State
	if state == "" {
		state = "UNKNOWN"
	}
	agg.StateCount[JobStateKey{Action: attr.JobType, State: state}]++

	// Queued jobs by reason code.
	if state == jobStateQueued {
		reason := strconv.Itoa(attr.JobQueueReason)
		agg.QueuedCount[JobQueueKey{Action: attr.JobType, Reason: reason}]++
	}

	// Files transferred and dedup-ratio mean (per action/policy).
	agg.FilesCount[policyKey] += float64(attr.NumberOfFiles)
	agg.DedupSum[policyKey] += attr.DedupRatio
	agg.DedupCount[policyKey]++

	// Duration histogram for completed jobs only (EndTime after StartTime).
	if !attr.EndTime.IsZero() && attr.EndTime.After(attr.StartTime) {
		agg.observeDuration(policyKey, attr.EndTime.Sub(attr.StartTime).Seconds())
	}
}

// FetchJobDetails retrieves a page of job records (up to 100) at the specified
// pagination offset and folds each job into the aggregator. It processes all jobs
// in the API response and uses the API's pagination metadata to determine the
// next offset.
//
// Parameters:
//   - ctx: Context for request cancellation and timeout
//   - client: Configured NetBackup API client
//   - agg: Aggregator accumulating all job-derived metrics across pages
//   - offset: Pagination offset (starts at 0, use Meta.Pagination.Next for subsequent calls)
//   - startTime: Filter to only include jobs completed after this time
//
// Returns:
//   - Next pagination offset (or -1 if no more jobs)
//   - Error if the API request fails
func FetchJobDetails(
	ctx context.Context,
	client *NbuClient,
	agg *JobAggregator,
	offset int,
	startTime time.Time,
) (int, error) {
	// Start child span "netbackup.fetch_job_page" from parent context
	ctx, span := client.tracing.StartSpan(ctx, "netbackup.fetch_job_page", trace.SpanKindClient)
	defer span.End()

	var jobs models.Jobs

	queryParams := map[string]string{
		QueryParamLimit:  jobPageLimit, // Fetch up to 100 jobs per page for efficiency
		QueryParamOffset: strconv.Itoa(offset),
		QueryParamSort:   "jobId",
		QueryParamFilter: fmt.Sprintf("endTime gt %s", utils.ConvertTimeToNBUDate(startTime)),
	}

	url := client.cfg.BuildURL(jobsPath, queryParams)

	if err := client.FetchData(ctx, url, &jobs); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return -1, fmt.Errorf("failed to fetch job details at offset %d: %w", offset, err)
	}

	// No more jobs to process
	if len(jobs.Data) == 0 {
		attrs := []attribute.KeyValue{
			attribute.Int(telemetry.AttrNetBackupPageOffset, offset),
			attribute.Int(telemetry.AttrNetBackupJobsInPage, 0),
		}
		span.SetAttributes(attrs...)
		span.SetStatus(codes.Ok, "No more jobs to process")
		return -1, nil
	}

	// Process ALL jobs in the batch response
	for _, job := range jobs.Data {
		aggregateJob(agg, job.Attributes)
	}

	// Batch span attributes for job page
	attrs := []attribute.KeyValue{
		attribute.Int(telemetry.AttrNetBackupPageOffset, offset),
		attribute.Int(telemetry.AttrNetBackupJobsInPage, len(jobs.Data)),
	}
	span.SetAttributes(attrs...)

	// Check if we've reached the last page
	if jobs.Meta.Pagination.Offset == jobs.Meta.Pagination.Last {
		span.SetStatus(codes.Ok, "Last page reached")
		return -1, nil
	}

	span.SetStatus(codes.Ok, "Page fetched successfully")
	return jobs.Meta.Pagination.Next, nil
}

// HandlePagination provides a generic pagination handler that repeatedly calls a fetch function
// until all pages have been processed. It handles context cancellation and propagates errors
// from the fetch function.
//
// The fetch function should:
//   - Accept a context and offset parameter
//   - Return the next offset (or -1 when pagination is complete)
//   - Return an error if the fetch operation fails
//
// Parameters:
//   - ctx: Context for cancellation (checked before each page fetch)
//   - fetchFunc: Function to fetch and process a single page of results
//
// Returns an error if:
//   - Context is cancelled (ctx.Err())
//   - The fetch function returns an error
//
// Example:
//
//	err := HandlePagination(ctx, func(ctx context.Context, offset int) (int, error) {
//	    return FetchJobDetails(ctx, client, metrics, offset, startTime)
//	})
func HandlePagination(ctx context.Context, fetchFunc func(ctx context.Context, offset int) (int, error)) error {
	offset := 0
	for offset != -1 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			nextOffset, err := fetchFunc(ctx, offset)
			if err != nil {
				return err
			}
			offset = nextOffset
		}
	}
	return nil
}

// FetchAllJobs aggregates job statistics by iterating over paginated job data.
// Uses batch pagination (100 jobs per request) to minimize API calls.
//
// It calculates the start time based on the scraping interval and fetches all jobs
// that completed after that time. The function returns three typed slices with
// aggregated statistics:
//   - jobsSize: Total bytes transferred per job type/policy/status
//   - jobsCount: Number of jobs per job type/policy/status
//   - statusCount: Number of jobs per action/status (simplified aggregation)
//
// Parameters:
//   - ctx: Context for request cancellation and timeout
//   - client: Configured NetBackup API client
//   - scrapingInterval: Duration string (e.g., "5m", "1h") defining the time window
//
// Returns typed slices and an error if:
//   - The scraping interval cannot be parsed
//   - Any API request fails during pagination
//
// FetchAllJobs is a backward-compatible wrapper over FetchAllJobsFull that
// returns only the size/count/status slices. New callers that also need the
// state, dedup, files, queued, and duration metrics should use FetchAllJobsFull.
func FetchAllJobs(
	ctx context.Context,
	client *NbuClient,
	scrapingInterval string,
) (jobsSize []JobMetricValue, jobsCount []JobMetricValue, statusCount []JobStatusMetricValue, err error) {
	agg, err := FetchAllJobsFull(ctx, client, scrapingInterval)
	if err != nil {
		return nil, nil, nil, err
	}

	jobsSize = make([]JobMetricValue, 0, len(agg.Size))
	jobsCount = make([]JobMetricValue, 0, len(agg.Count))
	statusCount = make([]JobStatusMetricValue, 0, len(agg.StatusCount))
	for key, value := range agg.Size {
		jobsSize = append(jobsSize, JobMetricValue{Key: key, Value: value})
	}
	for key, value := range agg.Count {
		jobsCount = append(jobsCount, JobMetricValue{Key: key, Value: value})
	}
	for key, value := range agg.StatusCount {
		statusCount = append(statusCount, JobStatusMetricValue{Key: key, Value: value})
	}
	return jobsSize, jobsCount, statusCount, nil
}

// FetchAllJobsFull aggregates all job-derived statistics over the scraping window
// and returns the populated JobAggregator. It uses batch pagination (100 jobs per
// request) and filters to jobs that completed after now-scrapingInterval.
//
// Returns an error if the scraping interval cannot be parsed or any API request
// fails during pagination.
func FetchAllJobsFull(
	ctx context.Context,
	client *NbuClient,
	scrapingInterval string,
) (*JobAggregator, error) {
	// Start child span "netbackup.fetch_jobs" from parent context
	ctx, span := client.tracing.StartSpan(ctx, "netbackup.fetch_jobs", trace.SpanKindClient)
	defer span.End()

	duration, parseErr := time.ParseDuration("-" + scrapingInterval)
	if parseErr != nil {
		span.RecordError(parseErr)
		span.SetStatus(codes.Error, parseErr.Error())
		return nil, fmt.Errorf("invalid scraping interval %s: %w", scrapingInterval, parseErr)
	}

	startTime := time.Now().Add(duration).UTC()
	log.Debugf("Fetching jobs since %s", startTime.Format(time.RFC3339))

	// Single aggregator threaded through every page; comparable struct keys enable
	// direct map lookups, and capacity hints reduce reallocation during processing.
	agg := NewJobAggregator()

	// Track page count
	pageCount := 0

	err := HandlePagination(ctx, func(ctx context.Context, offset int) (int, error) {
		pageCount++
		return FetchJobDetails(ctx, client, agg, offset, startTime)
	})

	// Calculate total jobs fetched
	totalJobs := 0
	for _, count := range agg.Count {
		totalJobs += int(count)
	}

	// Batch span attributes for job operation
	attrs := []attribute.KeyValue{
		attribute.String(telemetry.AttrNetBackupEndpoint, jobsPath),
		attribute.String(telemetry.AttrNetBackupTimeWindow, scrapingInterval),
		attribute.String(telemetry.AttrNetBackupStartTime, startTime.Format(time.RFC3339)),
		attribute.Int(telemetry.AttrNetBackupTotalJobs, totalJobs),
		attribute.Int(telemetry.AttrNetBackupTotalPages, pageCount),
	}
	span.SetAttributes(attrs...)

	// Handle errors in job fetching
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetStatus(codes.Ok, fmt.Sprintf("Fetched %d jobs across %d pages", totalJobs, pageCount))
	return agg, nil
}
