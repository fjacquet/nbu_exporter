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
	storageTypeTape  = "Tape"                   // Storage type identifier for tape units (excluded from metrics)
	storagePath      = "/storage/storage-units" // API endpoint for storage unit data
	jobsPath         = "/admin/jobs"            // API endpoint for job data
	sizeTypeFree     = "free"                   // Metric dimension for free capacity
	sizeTypeUsed     = "used"                   // Metric dimension for used capacity
	bytesPerKilobyte = 1024                     // Conversion factor from kilobytes to bytes
)

// FetchStorage retrieves storage unit information from the NetBackup API and populates
// the provided metrics map with capacity data. It fetches all storage units and extracts
// free and used capacity metrics for non-tape storage.
//
// Tape storage units are excluded from metrics as they don't report meaningful capacity values.
//
// Parameters:
//   - ctx: Context for request cancellation and timeout
//   - client: NetBackup API client (interface for testability)
//   - storageMetrics: Map to populate with storage metrics (key format: "name|type|size")
//
// The function populates storageMetrics with entries like:
//   - "disk-pool-1|MEDIA_SERVER|free" -> 5368709120000 (bytes)
//   - "disk-pool-1|MEDIA_SERVER|used" -> 5368709120000 (bytes)
//
// Returns an error if the API request fails or response cannot be parsed.
//
// Example usage:
//
//	storageMetrics := make(map[string]float64)
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//
//	if err := FetchStorage(ctx, client, storageMetrics); err != nil {
//	    log.Errorf("Failed to fetch storage: %v", err)
//	    return err
//	}
//
//	// Access metrics using structured keys
//	for key, value := range storageMetrics {
//	    labels := strings.Split(key, "|")
//	    name, storageType, sizeType := labels[0], labels[1], labels[2]
//	    log.Infof("Storage %s (%s) %s: %.2f GB", name, storageType, sizeType, value/1e9)
//	}
func FetchStorage(ctx context.Context, client *NbuClient, storageMetrics map[string]float64) error {
	// Start child span "netbackup.fetch_storage" from parent context
	ctx, span := createSpan(ctx, client.tracer, "netbackup.fetch_storage", trace.SpanKindClient)
	if span != nil {
		defer span.End()
	}

	var storages models.Storages

	url := client.cfg.BuildURL(storagePath, map[string]string{
		QueryParamLimit:  pageLimit,
		QueryParamOffset: "0",
	})

	if err := client.FetchData(ctx, url, &storages); err != nil {
		// Record error as span event and set span status to error
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		return fmt.Errorf("failed to fetch storage data: %w", err)
	}

	for _, data := range storages.Data {
		// Skip tape storage units
		if data.Attributes.StorageType == storageTypeTape {
			continue
		}

		stuName := data.Attributes.Name
		stuType := data.Attributes.StorageServerType

		freeKey := StorageMetricKey{Name: stuName, Type: stuType, Size: sizeTypeFree}
		usedKey := StorageMetricKey{Name: stuName, Type: stuType, Size: sizeTypeUsed}

		storageMetrics[freeKey.String()] = float64(data.Attributes.FreeCapacityBytes)
		storageMetrics[usedKey.String()] = float64(data.Attributes.UsedCapacityBytes)
	}

	// Batch span attributes for storage operation
	if span != nil {
		attrs := []attribute.KeyValue{
			attribute.String(telemetry.AttrNetBackupEndpoint, storagePath),
			attribute.Int(telemetry.AttrNetBackupStorageUnits, len(storages.Data)),
			attribute.String(telemetry.AttrNetBackupAPIVersion, client.cfg.NbuServer.APIVersion),
		}
		span.SetAttributes(attrs...)
	}

	log.Debugf("Fetched storage data for %d storage units", len(storages.Data))
	return nil
}

// FetchJobDetails retrieves a single job record at the specified pagination offset and
// updates the provided metrics maps with job statistics. This function is designed to be
// called repeatedly during pagination to process all jobs within a time window.
//
// The function fetches one job at a time (limit=1) to enable efficient pagination and
// filters jobs by endTime to only include jobs completed after startTime.
//
// Parameters:
//   - ctx: Context for request cancellation and timeout
//   - client: Configured NetBackup API client
//   - jobsSize: Map to accumulate bytes transferred per job type/status
//   - jobsCount: Map to accumulate job counts per job type/status
//   - jobsStatusCount: Map to accumulate job counts per action/status
//   - offset: Pagination offset for the current request
//   - startTime: Filter to only include jobs completed after this time
//
// Returns:
//   - Next pagination offset (or -1 if no more jobs)
//   - Error if the API request fails
//
// The function updates metrics maps with keys like:
//   - jobsSize["BACKUP|VMWARE|0"] += bytes transferred
//   - jobsCount["BACKUP|VMWARE|0"] += 1
//   - jobsStatusCount["BACKUP|0"] += 1
func FetchJobDetails(
	ctx context.Context,
	client *NbuClient,
	jobsSize, jobsCount, jobsStatusCount map[string]float64,
	offset int,
	startTime time.Time,
) (int, error) {
	// Start child span "netbackup.fetch_job_page" from parent context
	ctx, span := createSpan(ctx, client.tracer, "netbackup.fetch_job_page", trace.SpanKindClient)
	if span != nil {
		defer span.End()
	}

	var jobs models.Jobs

	queryParams := map[string]string{
		QueryParamLimit:  "1",
		QueryParamOffset: strconv.Itoa(offset),
		QueryParamSort:   "jobId",
		QueryParamFilter: fmt.Sprintf("endTime%%20gt%%20%s", utils.ConvertTimeToNBUDate(startTime)),
	}

	url := client.cfg.BuildURL(jobsPath, queryParams)

	if err := client.FetchData(ctx, url, &jobs); err != nil {
		// Record error as span event and set span status to error
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		return -1, fmt.Errorf("failed to fetch job details at offset %d: %w", offset, err)
	}

	// No more jobs to process
	if len(jobs.Data) == 0 {
		if span != nil {
			attrs := []attribute.KeyValue{
				attribute.Int(telemetry.AttrNetBackupJobsInPage, 0),
			}
			span.SetAttributes(attrs...)
			span.SetStatus(codes.Ok, "No more jobs to process")
		}
		return -1, nil
	}

	job := jobs.Data[0]

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

	// Update metrics
	jobsCount[jobKey.String()]++
	jobsStatusCount[statusKey.String()]++
	jobsSize[jobKey.String()] += float64(job.Attributes.KilobytesTransferred * bytesPerKilobyte)

	// Batch span attributes for job page
	if span != nil {
		// Calculate page number (assuming page size of 1 based on limit)
		pageNumber := offset + 1
		attrs := []attribute.KeyValue{
			attribute.Int(telemetry.AttrNetBackupPageOffset, offset),
			attribute.Int(telemetry.AttrNetBackupPageNumber, pageNumber),
			attribute.Int(telemetry.AttrNetBackupJobsInPage, len(jobs.Data)),
		}
		span.SetAttributes(attrs...)
	}

	// Check if we've reached the last page
	if jobs.Meta.Pagination.Offset == jobs.Meta.Pagination.Last {
		if span != nil {
			span.SetStatus(codes.Ok, "Last page reached")
		}
		return -1, nil
	}

	// Set span status to OK for successful page fetch
	if span != nil {
		span.SetStatus(codes.Ok, "Page fetched successfully")
	}

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

// FetchAllJobs aggregates job statistics by iterating over all paginated job data within
// the configured time window. It calculates the start time based on the scraping interval
// and fetches all jobs that completed after that time.
//
// The function uses pagination to efficiently process large job datasets and populates
// three metrics maps with aggregated statistics:
//   - jobsSize: Total bytes transferred per job type/policy/status
//   - jobsCount: Number of jobs per job type/policy/status
//   - jobsStatusCount: Number of jobs per action/status (simplified aggregation)
//
// Parameters:
//   - ctx: Context for request cancellation and timeout
//   - client: Configured NetBackup API client
//   - jobsSize: Map to populate with bytes transferred metrics
//   - jobsCount: Map to populate with job count metrics
//   - jobsStatusCount: Map to populate with status count metrics
//   - scrapingInterval: Duration string (e.g., "5m", "1h") defining the time window
//
// Returns an error if:
//   - The scraping interval cannot be parsed
//   - Any API request fails during pagination
//
// Example usage:
//
//	jobsSize := make(map[string]float64)
//	jobsCount := make(map[string]float64)
//	jobsStatusCount := make(map[string]float64)
//	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
//	defer cancel()
//
//	err := FetchAllJobs(ctx, client, jobsSize, jobsCount, jobsStatusCount, "5m")
//	if err != nil {
//	    log.Errorf("Failed to fetch jobs: %v", err)
//	    return err
//	}
//
//	// Access aggregated metrics
//	for key, count := range jobsCount {
//	    labels := strings.Split(key, "|")
//	    action, policyType, status := labels[0], labels[1], labels[2]
//	    bytes := jobsSize[key]
//	    log.Infof("Jobs: %s/%s (status %s) - count: %.0f, bytes: %.2f GB",
//	        action, policyType, status, count, bytes/1e9)
//	}
func FetchAllJobs(
	ctx context.Context,
	client *NbuClient,
	jobsSize, jobsCount, jobsStatusCount map[string]float64,
	scrapingInterval string,
) error {
	// Start child span "netbackup.fetch_jobs" from parent context
	ctx, span := createSpan(ctx, client.tracer, "netbackup.fetch_jobs", trace.SpanKindClient)
	if span != nil {
		defer span.End()
	}

	duration, err := time.ParseDuration("-" + scrapingInterval)
	if err != nil {
		// Record error as span event and set span status to error
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		return fmt.Errorf("invalid scraping interval %s: %w", scrapingInterval, err)
	}

	startTime := time.Now().Add(duration).UTC()
	log.Debugf("Fetching jobs since %s", startTime.Format(time.RFC3339))

	// Track initial job counts to calculate total jobs fetched
	initialJobCount := 0
	for _, count := range jobsCount {
		initialJobCount += int(count)
	}

	// Track page count
	pageCount := 0

	err = HandlePagination(ctx, func(ctx context.Context, offset int) (int, error) {
		pageCount++
		return FetchJobDetails(ctx, client, jobsSize, jobsCount, jobsStatusCount, offset, startTime)
	})

	// Calculate total jobs fetched
	finalJobCount := 0
	for _, count := range jobsCount {
		finalJobCount += int(count)
	}
	totalJobs := finalJobCount - initialJobCount

	// Batch span attributes for job operation
	if span != nil {
		attrs := []attribute.KeyValue{
			attribute.String(telemetry.AttrNetBackupEndpoint, jobsPath),
			attribute.String(telemetry.AttrNetBackupTimeWindow, scrapingInterval),
			attribute.String(telemetry.AttrNetBackupStartTime, startTime.Format(time.RFC3339)),
			attribute.Int(telemetry.AttrNetBackupTotalJobs, totalJobs),
			attribute.Int(telemetry.AttrNetBackupTotalPages, pageCount),
		}
		span.SetAttributes(attrs...)
	}

	// Handle errors in job fetching
	if err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		return err
	}

	// Set span status to OK for successful requests
	if span != nil {
		span.SetStatus(codes.Ok, fmt.Sprintf("Fetched %d jobs across %d pages", totalJobs, pageCount))
	}

	return nil
}
