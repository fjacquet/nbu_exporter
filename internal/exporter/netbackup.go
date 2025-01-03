package exporter

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/fjacquet/nbu_exporter/internal/logging"
	"github.com/fjacquet/nbu_exporter/internal/models"
	"github.com/fjacquet/nbu_exporter/internal/utils"
	"github.com/go-resty/resty/v2"
)

const (
	pageLimit           = "100"
	timeout             = 1 * time.Minute
	contentType         = "application/json"
	queryParamLimit     = "page[limit]"
	queryParamOffset    = "page[offset]"
	queryParamSort      = "sort"
	queryParamFilter    = "filter"
	headerAccept        = "Accept"
	headerAuthorization = "Authorization"
)

// createHTTPClient initializes and returns a Resty client configured for HTTP requests.
func createHTTPClient() *resty.Client {
	return resty.New().
		SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true}).
		SetTimeout(timeout)
}

// buildURL constructs a complete URL from base, path, and query parameters.
func buildURL(baseURL, path string, queryParams map[string]string) string {
	u, _ := url.Parse(baseURL)
	u.Path = path
	q := u.Query()
	for key, value := range queryParams {
		q.Set(key, value)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// fetchData sends an HTTP GET request and unmarshals the response body into the target object.
func fetchData(client *resty.Client, url string, headers map[string]string, target interface{}) error {
	resp, err := client.R().
		SetHeaders(headers).
		Get(url)
	if err != nil {
		return fmt.Errorf("HTTP request to %s failed: %w", url, err)
	}
	if err := json.Unmarshal(resp.Body(), target); err != nil {
		return fmt.Errorf("failed to unmarshal response from %s: %w", url, err)
	}
	return nil
}

// fetchStorage retrieves and processes storage unit information.
func fetchStorage(disks map[string]float64, cfg models.Config) error {
	var storages models.Storages
	nbuRoot := fmt.Sprintf("%s://%s:%s%s", cfg.NbuServer.Scheme, cfg.NbuServer.Host, cfg.NbuServer.Port, cfg.NbuServer.URI)

	url := buildURL(nbuRoot, "/storage/storage-units", map[string]string{
		queryParamLimit:  pageLimit,
		queryParamOffset: "0",
	})

	headers := map[string]string{
		headerAccept:        contentType,
		headerAuthorization: cfg.NbuServer.APIKey,
	}

	err := fetchData(createHTTPClient(), url, headers, &storages)
	if err != nil {
		logging.LogError(fmt.Sprintf("Error fetching storage data: %v", err))
		return err
	}

	for _, data := range storages.Data {
		if data.Attributes.StorageType == "Tape" {
			continue
		}

		stuName := data.Attributes.Name
		stuType := data.Attributes.StorageServerType
		disks[fmt.Sprintf("%s|%s|free", stuName, stuType)] = float64(data.Attributes.FreeCapacityBytes)
		disks[fmt.Sprintf("%s|%s|used", stuName, stuType)] = float64(data.Attributes.UsedCapacityBytes)
	}
	return nil
}

// fetchJobDetails retrieves and processes job details for a specific offset.
func fetchJobDetails(client *resty.Client, jobsSize, jobsCount, jobsStatusCount map[string]float64, offset int, cfg models.Config) (int, error) {
	var jobs models.Jobs
	nbuRoot := fmt.Sprintf("%s://%s:%s%s", cfg.NbuServer.Scheme, cfg.NbuServer.Host, cfg.NbuServer.Port, cfg.NbuServer.URI)

	duration, err := time.ParseDuration("-" + cfg.Server.ScrappingInterval)
	if err != nil {
		return -1, fmt.Errorf("invalid scrapping interval: %w", err)
	}

	startTime := time.Now().Add(duration).UTC()
	queryParams := map[string]string{
		queryParamLimit:  "1",
		queryParamOffset: fmt.Sprintf("%d", offset),
		queryParamSort:   "jobId",
		queryParamFilter: fmt.Sprintf("endTime%%20gt%%20%s", utils.ConvertTimeToNBUDate(startTime)),
	}

	url := buildURL(nbuRoot, "/admin/jobs", queryParams)
	headers := map[string]string{
		headerAccept:        contentType,
		headerAuthorization: cfg.NbuServer.APIKey,
	}

	if err := fetchData(client, url, headers, &jobs); err != nil {
		return -1, err
	}

	if len(jobs.Data) == 0 {
		return -1, nil
	}

	job := jobs.Data[0]
	key := fmt.Sprintf("%s|%s|%d", job.Attributes.JobType, job.Attributes.PolicyType, job.Attributes.Status)
	key2 := fmt.Sprintf("%s|%d", job.Attributes.JobType, job.Attributes.Status)

	jobsCount[key]++
	jobsStatusCount[key2]++
	jobsSize[key] += float64(job.Attributes.KilobytesTransferred * 1024)

	if jobs.Meta.Pagination.Offset == jobs.Meta.Pagination.Last {
		return -1, nil
	}

	return jobs.Meta.Pagination.Next, nil
}

// handlePagination iterates over paginated responses and processes them.
func handlePagination(fetchFunc func(offset int) (int, error)) error {
	offset := 0
	for offset != -1 {
		nextOffset, err := fetchFunc(offset)
		if err != nil {
			return err
		}
		offset = nextOffset
	}
	return nil
}

// fetchAllJobs aggregates job statistics by iterating over paginated job data.
func fetchAllJobs(jobsSize, jobsCount, jobsStatusCount map[string]float64, cfg models.Config) error {
	client := createHTTPClient()
	return handlePagination(func(offset int) (int, error) {
		return fetchJobDetails(client, jobsSize, jobsCount, jobsStatusCount, offset, cfg)
	})
}
