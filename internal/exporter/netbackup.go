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
func FetchStorage(ctx context.Context, client *NbuClient) ([]StorageMetricValue, error) {
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
		return nil, fmt.Errorf("failed to fetch storage data: %w", err)
	}

	var metrics []StorageMetricValue
	for _, data := range storages.Data {
		// Skip tape storage units
		if data.Attributes.StorageType == storageTypeTape {
			continue
		}

		stuName := data.Attributes.Name
		stuType := data.Attributes.StorageServerType

		metrics = append(metrics, StorageMetricValue{
			Key:   StorageMetricKey{Name: stuName, Type: stuType, Size: sizeTypeFree},
			Value: float64(data.Attributes.FreeCapacityBytes),
		})
		metrics = append(metrics, StorageMetricValue{
			Key:   StorageMetricKey{Name: stuName, Type: stuType, Size: sizeTypeUsed},
			Value: float64(data.Attributes.UsedCapacityBytes),
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
	return metrics, nil
}

// FetchJobDetails retrieves a page of job records (up to 100) at the specified
// pagination offset and updates the provided metrics maps with job statistics.
// This function processes all jobs in the API response and uses the API's
// pagination metadata to determine the next offset.
//
// Parameters:
//   - ctx: Context for request cancellation and timeout
//   - client: Configured NetBackup API client
//   - jobsSize: Typed map to accumulate bytes transferred per job type/status
//   - jobsCount: Typed map to accumulate job counts per job type/status
//   - jobsStatusCount: Typed map to accumulate job counts per action/status
//   - offset: Pagination offset (starts at 0, use Meta.Pagination.Next for subsequent calls)
//   - startTime: Filter to only include jobs completed after this time
//
// Returns:
//   - Next pagination offset (or -1 if no more jobs)
//   - Error if the API request fails
func FetchJobDetails(
	ctx context.Context,
	client *NbuClient,
	jobsSize, jobsCount map[JobMetricKey]float64,
	jobsStatusCount map[JobStatusKey]float64,
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
		QueryParamFilter: fmt.Sprintf("endTime%%20gt%%20%s", utils.ConvertTimeToNBUDate(startTime)),
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
		// Create structured metric keys
		jobKey := JobMetricKey{
			Action:     job.Attributes.JobType,
			PolicyType: job.Attributes.PolicyType,
			Status:     strconv.Itoa(job.Attributes.Status),
		}

		statusKey := JobStatusKey{
			Action: job.Attributes.JobType,
			Status: strconv.Itoa(job.Attributes.Status),
		}

		// Update metrics using typed keys directly
		jobsCount[jobKey]++
		jobsStatusCount[statusKey]++
		jobsSize[jobKey] += float64(job.Attributes.KilobytesTransferred * bytesPerKilobyte)
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
func FetchAllJobs(
	ctx context.Context,
	client *NbuClient,
	scrapingInterval string,
) (jobsSize []JobMetricValue, jobsCount []JobMetricValue, statusCount []JobStatusMetricValue, err error) {
	// Start child span "netbackup.fetch_jobs" from parent context
	ctx, span := client.tracing.StartSpan(ctx, "netbackup.fetch_jobs", trace.SpanKindClient)
	defer span.End()

	duration, parseErr := time.ParseDuration("-" + scrapingInterval)
	if parseErr != nil {
		span.RecordError(parseErr)
		span.SetStatus(codes.Error, parseErr.Error())
		return nil, nil, nil, fmt.Errorf("invalid scraping interval %s: %w", scrapingInterval, parseErr)
	}

	startTime := time.Now().Add(duration).UTC()
	log.Debugf("Fetching jobs since %s", startTime.Format(time.RFC3339))

	// Use typed maps for aggregation (struct keys are comparable in Go)
	sizeMap := make(map[JobMetricKey]float64)
	countMap := make(map[JobMetricKey]float64)
	statusMap := make(map[JobStatusKey]float64)

	// Track page count
	pageCount := 0

	err = HandlePagination(ctx, func(ctx context.Context, offset int) (int, error) {
		pageCount++
		return FetchJobDetails(ctx, client, sizeMap, countMap, statusMap, offset, startTime)
	})

	// Calculate total jobs fetched
	totalJobs := 0
	for _, count := range countMap {
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
		return nil, nil, nil, err
	}

	// Convert maps to typed slices
	for key, value := range sizeMap {
		jobsSize = append(jobsSize, JobMetricValue{Key: key, Value: value})
	}
	for key, value := range countMap {
		jobsCount = append(jobsCount, JobMetricValue{Key: key, Value: value})
	}
	for key, value := range statusMap {
		statusCount = append(statusCount, JobStatusMetricValue{Key: key, Value: value})
	}

	span.SetStatus(codes.Ok, fmt.Sprintf("Fetched %d jobs across %d pages", totalJobs, pageCount))
	return jobsSize, jobsCount, statusCount, nil
}
