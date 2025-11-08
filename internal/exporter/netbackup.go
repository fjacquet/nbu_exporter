package exporter

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/fjacquet/nbu_exporter/internal/models"
	"github.com/fjacquet/nbu_exporter/internal/utils"
	log "github.com/sirupsen/logrus"
)

const (
	pageLimit        = "100"
	storageTypeTape  = "Tape"
	storagePath      = "/storage/storage-units"
	jobsPath         = "/admin/jobs"
	sizeTypeFree     = "free"
	sizeTypeUsed     = "used"
	bytesPerKilobyte = 1024
)

// FetchStorage retrieves and processes storage unit information from the NetBackup API.
func FetchStorage(ctx context.Context, client *NbuClient, storageMetrics map[string]float64) error {
	var storages models.Storages

	url := client.cfg.BuildURL(storagePath, map[string]string{
		QueryParamLimit:  pageLimit,
		QueryParamOffset: "0",
	})

	if err := client.FetchData(ctx, url, &storages); err != nil {
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

	log.Debugf("Fetched storage data for %d storage units", len(storages.Data))
	return nil
}

// FetchJobDetails retrieves and processes job details for a specific offset.
func FetchJobDetails(
	ctx context.Context,
	client *NbuClient,
	jobsSize, jobsCount, jobsStatusCount map[string]float64,
	offset int,
	startTime time.Time,
) (int, error) {
	var jobs models.Jobs

	queryParams := map[string]string{
		QueryParamLimit:  "1",
		QueryParamOffset: strconv.Itoa(offset),
		QueryParamSort:   "jobId",
		QueryParamFilter: fmt.Sprintf("endTime%%20gt%%20%s", utils.ConvertTimeToNBUDate(startTime)),
	}

	url := client.cfg.BuildURL(jobsPath, queryParams)

	if err := client.FetchData(ctx, url, &jobs); err != nil {
		return -1, fmt.Errorf("failed to fetch job details at offset %d: %w", offset, err)
	}

	// No more jobs to process
	if len(jobs.Data) == 0 {
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

	// Check if we've reached the last page
	if jobs.Meta.Pagination.Offset == jobs.Meta.Pagination.Last {
		return -1, nil
	}

	return jobs.Meta.Pagination.Next, nil
}

// HandlePagination iterates over paginated responses and processes them.
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
func FetchAllJobs(
	ctx context.Context,
	client *NbuClient,
	jobsSize, jobsCount, jobsStatusCount map[string]float64,
	scrapingInterval string,
) error {
	duration, err := time.ParseDuration("-" + scrapingInterval)
	if err != nil {
		return fmt.Errorf("invalid scraping interval %s: %w", scrapingInterval, err)
	}

	startTime := time.Now().Add(duration).UTC()
	log.Debugf("Fetching jobs since %s", startTime.Format(time.RFC3339))

	return HandlePagination(ctx, func(ctx context.Context, offset int) (int, error) {
		return FetchJobDetails(ctx, client, jobsSize, jobsCount, jobsStatusCount, offset, startTime)
	})
}
