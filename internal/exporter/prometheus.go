// Package exporter implements the Prometheus Collector interface for NetBackup metrics.
// It collects storage and job statistics from the NetBackup API and exposes them
// in Prometheus format.
package exporter

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/fjacquet/nbu_exporter/internal/models"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

const collectionTimeout = 2 * time.Minute // Maximum time allowed for metric collection

// NbuCollector implements the Prometheus Collector interface for NetBackup metrics.
// It collects storage capacity and job statistics from the NetBackup API and exposes
// them as Prometheus metrics.
//
// The collector fetches:
//   - Storage unit capacity (free/used bytes) for disk-based storage
//   - Job statistics (count, bytes transferred) aggregated by type, policy, and status
//   - Job status counts aggregated by action and status code
//   - API version information
//
// Metrics are collected on-demand when Prometheus scrapes the /metrics endpoint.
type NbuCollector struct {
	cfg                models.Config
	client             *NbuClient
	nbuDiskSize        *prometheus.Desc
	nbuResponseTime    *prometheus.Desc
	nbuJobsSize        *prometheus.Desc
	nbuJobsCount       *prometheus.Desc
	nbuJobsStatusCount *prometheus.Desc
	nbuAPIVersion      *prometheus.Desc
}

// NewNbuCollector creates a new NetBackup collector with the provided configuration.
// It initializes the HTTP client, performs automatic API version detection if needed,
// and registers Prometheus metric descriptors.
//
// The collector creates the following metrics:
//   - nbu_disk_bytes: Storage capacity in bytes (labels: name, type, size)
//   - nbu_jobs_bytes: Bytes transferred by jobs (labels: action, policy_type, status)
//   - nbu_jobs_count: Number of jobs (labels: action, policy_type, status)
//   - nbu_status_count: Job counts by status (labels: action, status)
//   - nbu_response_time_ms: API response time in milliseconds
//
// Version Detection:
//   - If apiVersion is not configured, automatic detection is performed
//   - Detection tries versions in descending order: 13.0 → 12.0 → 3.0
//   - If detection fails, an error is returned and collector creation fails
//   - If apiVersion is explicitly configured, detection is bypassed
//
// Example:
//
//	cfg := models.Config{...}
//	collector, err := NewNbuCollector(cfg)
//	if err != nil {
//	    log.Fatalf("Failed to create collector: %v", err)
//	}
//	prometheus.MustRegister(collector)
func NewNbuCollector(cfg models.Config) (*NbuCollector, error) {
	// Create base client
	client := NewNbuClient(cfg)

	// Perform version detection if apiVersion is not explicitly configured
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if cfg.NbuServer.APIVersion == "" {
		log.Info("API version not configured, performing automatic detection")
		detector := NewAPIVersionDetector(client, &cfg)
		detectedVersion, err := detector.DetectVersion(ctx)
		if err != nil {
			return nil, fmt.Errorf("automatic API version detection failed: %w", err)
		}

		// Update configuration with detected version
		cfg.NbuServer.APIVersion = detectedVersion
		client.cfg.NbuServer.APIVersion = detectedVersion
		log.Infof("Detected NetBackup API version: %s", detectedVersion)
	} else {
		log.Infof("Using configured NetBackup API version: %s", cfg.NbuServer.APIVersion)
	}

	return &NbuCollector{
		cfg:    cfg,
		client: client,
		nbuResponseTime: prometheus.NewDesc(
			"nbu_response_time_ms",
			"The server response time in milliseconds",
			nil, nil,
		),
		nbuDiskSize: prometheus.NewDesc(
			"nbu_disk_bytes",
			"The quantity of storage bytes",
			[]string{"name", "type", "size"}, nil,
		),
		nbuJobsSize: prometheus.NewDesc(
			"nbu_jobs_bytes",
			"The quantity of processed bytes",
			[]string{"action", "policy_type", "status"}, nil,
		),
		nbuJobsCount: prometheus.NewDesc(
			"nbu_jobs_count",
			"The quantity of jobs",
			[]string{"action", "policy_type", "status"}, nil,
		),
		nbuJobsStatusCount: prometheus.NewDesc(
			"nbu_status_count",
			"The quantity per status",
			[]string{"action", "status"}, nil,
		),
		nbuAPIVersion: prometheus.NewDesc(
			"nbu_api_version",
			"The NetBackup API version currently in use",
			[]string{"version"}, nil,
		),
	}, nil
}

// Describe sends the descriptors of each metric to the provided channel.
// This method is required by the prometheus.Collector interface and is called
// during collector registration to validate metric definitions.
func (c *NbuCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.nbuDiskSize
	ch <- c.nbuResponseTime
	ch <- c.nbuJobsSize
	ch <- c.nbuJobsCount
	ch <- c.nbuJobsStatusCount
	ch <- c.nbuAPIVersion
}

// Collect fetches metrics from NetBackup and sends them to the provided channel.
// This method is called by Prometheus on each scrape request and performs the following:
//  1. Fetches storage unit capacity data
//  2. Fetches job statistics within the configured time window
//  3. Converts metrics to Prometheus format and sends to the channel
//
// The method uses a 2-minute timeout for the entire collection process and continues
// to expose partial metrics even if some API calls fail. Errors are logged but do not
// prevent metric exposition.
//
// This method is required by the prometheus.Collector interface.
func (c *NbuCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), collectionTimeout)
	defer cancel()

	// Collect storage metrics
	storageMetrics := make(map[string]float64)
	if err := FetchStorage(ctx, c.client, storageMetrics); err != nil {
		log.Errorf("Failed to fetch storage metrics: %v", err)
		// Continue to try fetching job metrics even if storage fails
	}

	// Collect job metrics
	jobsSize := make(map[string]float64)
	jobsCount := make(map[string]float64)
	jobsStatusCount := make(map[string]float64)

	if err := FetchAllJobs(ctx, c.client, jobsSize, jobsCount, jobsStatusCount, c.cfg.Server.ScrapingInterval); err != nil {
		log.Errorf("Failed to fetch job metrics: %v", err)
		// Continue to expose whatever metrics we have
	}

	// Expose storage metrics
	for key, value := range storageMetrics {
		labels := strings.Split(key, "|")
		ch <- prometheus.MustNewConstMetric(
			c.nbuDiskSize,
			prometheus.GaugeValue,
			value,
			labels[0], labels[1], labels[2],
		)
	}

	// Expose job size metrics
	for key, value := range jobsSize {
		labels := strings.Split(key, "|")
		ch <- prometheus.MustNewConstMetric(
			c.nbuJobsSize,
			prometheus.GaugeValue,
			value,
			labels[0], labels[1], labels[2],
		)
	}

	// Expose job count metrics
	for key, value := range jobsCount {
		labels := strings.Split(key, "|")
		ch <- prometheus.MustNewConstMetric(
			c.nbuJobsCount,
			prometheus.GaugeValue,
			value,
			labels[0], labels[1], labels[2],
		)
	}

	// Expose job status count metrics
	for key, value := range jobsStatusCount {
		labels := strings.Split(key, "|")
		ch <- prometheus.MustNewConstMetric(
			c.nbuJobsStatusCount,
			prometheus.GaugeValue,
			value,
			labels[0], labels[1],
		)
	}

	// Expose API version metric
	ch <- prometheus.MustNewConstMetric(
		c.nbuAPIVersion,
		prometheus.GaugeValue,
		1,
		c.cfg.NbuServer.APIVersion,
	)

	log.Debugf("Collected %d storage, %d job size, %d job count, %d status metrics",
		len(storageMetrics), len(jobsSize), len(jobsCount), len(jobsStatusCount))
}
