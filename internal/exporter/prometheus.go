package exporter

import (
	"context"
	"strings"
	"time"

	"github.com/fjacquet/nbu_exporter/internal/models"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

const collectionTimeout = 2 * time.Minute

// NbuCollector implements the Prometheus Collector interface for NetBackup metrics.
type NbuCollector struct {
	cfg                models.Config
	client             *NbuClient
	nbuDiskSize        *prometheus.Desc
	nbuResponseTime    *prometheus.Desc
	nbuJobsSize        *prometheus.Desc
	nbuJobsCount       *prometheus.Desc
	nbuJobsStatusCount *prometheus.Desc
}

// NewNbuCollector creates a new NetBackup collector with the provided configuration.
func NewNbuCollector(cfg models.Config) *NbuCollector {
	client := NewNbuClient(cfg)

	// Attempt to detect API version during initialization
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if detectedVersion, err := client.DetectAPIVersion(ctx); err != nil {
		log.Warnf("Failed to detect NetBackup API version: %v", err)
		log.Infof("Using configured API version: %s", cfg.NbuServer.APIVersion)
	} else {
		log.Infof("Successfully connected to NetBackup API version: %s", detectedVersion)
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
	}
}

// Describe sends the descriptors of each metric to the provided channel.
func (c *NbuCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.nbuDiskSize
	ch <- c.nbuResponseTime
	ch <- c.nbuJobsSize
	ch <- c.nbuJobsCount
	ch <- c.nbuJobsStatusCount
}

// Collect fetches metrics from NetBackup and sends them to the provided channel.
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

	log.Debugf("Collected %d storage, %d job size, %d job count, %d status metrics",
		len(storageMetrics), len(jobsSize), len(jobsCount), len(jobsStatusCount))
}
