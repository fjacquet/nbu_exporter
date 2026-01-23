// Package exporter implements the Prometheus Collector interface for NetBackup metrics.
// It collects storage and job statistics from the NetBackup API and exposes them
// in Prometheus format.
package exporter

import (
	"context"
	"strings"
	"time"

	"github.com/fjacquet/nbu_exporter/internal/models"
	"github.com/fjacquet/nbu_exporter/internal/telemetry"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const collectionTimeout = 2 * time.Minute // Maximum time allowed for metric collection

// CollectorOption configures optional NbuCollector settings.
type CollectorOption func(*collectorOptions)

type collectorOptions struct {
	tracerProvider trace.TracerProvider
}

func defaultCollectorOptions() collectorOptions {
	return collectorOptions{
		tracerProvider: nil, // Will use noop via TracerWrapper
	}
}

// WithCollectorTracerProvider sets the TracerProvider for the collector.
// If not provided, tracing operations use a noop provider (no overhead).
func WithCollectorTracerProvider(tp trace.TracerProvider) CollectorOption {
	return func(o *collectorOptions) {
		o.tracerProvider = tp
	}
}

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
	tracing            *TracerWrapper
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
//	collector, err := NewNbuCollector(cfg, WithCollectorTracerProvider(tp))
//	if err != nil {
//	    log.Fatalf("Failed to create collector: %v", err)
//	}
//	prometheus.MustRegister(collector)
func NewNbuCollector(cfg models.Config, opts ...CollectorOption) (*NbuCollector, error) {
	// Apply options
	options := defaultCollectorOptions()
	for _, opt := range opts {
		opt(&options)
	}

	// Create base client with same TracerProvider
	client := NewNbuClient(cfg, WithTracerProvider(options.tracerProvider))

	// Perform version detection if needed (reuses shared logic)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := performVersionDetectionIfNeeded(ctx, client, &cfg); err != nil {
		return nil, err
	}

	// Create TracerWrapper for collector
	tracing := NewTracerWrapper(options.tracerProvider, "nbu-exporter/collector")

	return &NbuCollector{
		cfg:     cfg,
		client:  client,
		tracing: tracing,
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

// createScrapeSpan creates a root span for the Prometheus scrape cycle.
// It returns a context with the span and the span itself for lifecycle management.
// TracerWrapper ensures this is always safe (uses noop if tracing disabled).
//
// The span is configured with:
//   - Operation name: "prometheus.scrape"
//   - Span kind: SpanKindServer (representing the scrape as a server operation)
//
// Returns:
//   - context.Context: Context with span attached
//   - trace.Span: The created span (always valid, noop if tracing disabled)
func (c *NbuCollector) createScrapeSpan(ctx context.Context) (context.Context, trace.Span) {
	return c.tracing.StartSpan(ctx, "prometheus.scrape", trace.SpanKindServer)
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
// When OpenTelemetry tracing is enabled, this method creates a root span for the scrape
// cycle and records attributes including:
//   - scrape.duration_ms: Total time taken for metric collection
//   - scrape.storage_metrics_count: Number of storage metrics collected
//   - scrape.job_metrics_count: Number of job metrics collected
//   - scrape.status: Overall scrape status (success/partial_failure)
//
// Graceful Degradation:
// If storage or job fetching fails, the collector continues to expose whatever metrics
// were successfully collected. This ensures Prometheus receives partial data rather than
// no data at all, improving observability during partial outages.
//
// This method is required by the prometheus.Collector interface and is called automatically
// by Prometheus during each scrape cycle (typically every 15-60 seconds).
//
// Parameters:
//   - ch: Channel to send Prometheus metrics to (provided by Prometheus registry)
func (c *NbuCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), collectionTimeout)
	defer cancel()

	scrapeStart := time.Now()
	ctx, span := c.createScrapeSpan(ctx)
	defer span.End()

	// Collect all metrics
	storageMetrics, jobsSize, jobsCount, jobsStatusCount, storageErr, jobsErr := c.collectAllMetrics(ctx, span)

	// Update span with scrape results
	c.updateScrapeSpan(span, scrapeStart, storageMetrics, jobsSize, storageErr, jobsErr)

	// Expose all collected metrics
	c.exposeMetrics(ch, storageMetrics, jobsSize, jobsCount, jobsStatusCount)

	log.Debugf("Collected %d storage, %d job size, %d job count, %d status metrics",
		len(storageMetrics), len(jobsSize), len(jobsCount), len(jobsStatusCount))
}

// collectAllMetrics fetches storage and job metrics from NetBackup API.
// Returns the collected metrics maps and any errors encountered.
func (c *NbuCollector) collectAllMetrics(ctx context.Context, span trace.Span) (
	storageMetrics, jobsSize, jobsCount, jobsStatusCount map[string]float64,
	storageErr, jobsErr error,
) {
	storageMetrics = make(map[string]float64)
	jobsSize = make(map[string]float64)
	jobsCount = make(map[string]float64)
	jobsStatusCount = make(map[string]float64)

	// Collect storage metrics
	storageErr = c.collectStorageMetrics(ctx, span, storageMetrics)

	// Collect job metrics (continue even if storage fails)
	jobsErr = c.collectJobMetrics(ctx, span, jobsSize, jobsCount, jobsStatusCount)

	return
}

// collectStorageMetrics fetches storage metrics and records errors in span.
func (c *NbuCollector) collectStorageMetrics(ctx context.Context, span trace.Span, storageMetrics map[string]float64) error {
	if err := FetchStorage(ctx, c.client, storageMetrics); err != nil {
		log.Errorf("Failed to fetch storage metrics: %v", err)
		c.recordFetchError(span, "storage_fetch_error", err)
		return err
	}
	return nil
}

// collectJobMetrics fetches job metrics and records errors in span.
func (c *NbuCollector) collectJobMetrics(ctx context.Context, span trace.Span, jobsSize, jobsCount, jobsStatusCount map[string]float64) error {
	if err := FetchAllJobs(ctx, c.client, jobsSize, jobsCount, jobsStatusCount, c.cfg.Server.ScrapingInterval); err != nil {
		log.Errorf("Failed to fetch job metrics: %v", err)
		c.recordFetchError(span, "jobs_fetch_error", err)
		return err
	}
	return nil
}

// recordFetchError records a fetch error as a span event.
// TracerWrapper ensures span is always valid (noop if tracing disabled).
func (c *NbuCollector) recordFetchError(span trace.Span, eventName string, err error) {
	span.AddEvent(eventName, trace.WithAttributes(
		attribute.String(telemetry.AttrError, err.Error()),
	))
}

// updateScrapeSpan updates the span with scrape results and status.
// TracerWrapper ensures span is always valid (noop if tracing disabled).
func (c *NbuCollector) updateScrapeSpan(span trace.Span, scrapeStart time.Time, storageMetrics, jobsSize map[string]float64, storageErr, jobsErr error) {
	scrapeStatus := c.determineScrapeStatus(storageErr, jobsErr)
	c.setSpanStatus(span, scrapeStatus)
	c.recordScrapeAttributes(span, scrapeStart, storageMetrics, jobsSize, scrapeStatus)
}

// determineScrapeStatus returns the scrape status based on errors.
func (c *NbuCollector) determineScrapeStatus(storageErr, jobsErr error) string {
	if storageErr != nil || jobsErr != nil {
		return "partial_failure"
	}
	return "success"
}

// setSpanStatus sets the span status based on scrape status.
func (c *NbuCollector) setSpanStatus(span trace.Span, scrapeStatus string) {
	if scrapeStatus == "partial_failure" {
		span.SetStatus(codes.Error, "Partial failure during metric collection")
	} else {
		span.SetStatus(codes.Ok, "")
	}
}

// recordScrapeAttributes records scrape metrics as span attributes.
func (c *NbuCollector) recordScrapeAttributes(span trace.Span, scrapeStart time.Time, storageMetrics, jobsSize map[string]float64, scrapeStatus string) {
	scrapeDuration := time.Since(scrapeStart)
	attrs := []attribute.KeyValue{
		attribute.Float64(telemetry.AttrScrapeDurationMS, float64(scrapeDuration.Milliseconds())),
		attribute.Int(telemetry.AttrScrapeStorageMetricsCount, len(storageMetrics)),
		attribute.Int(telemetry.AttrScrapeJobMetricsCount, len(jobsSize)),
		attribute.String(telemetry.AttrScrapeStatus, scrapeStatus),
	}
	span.SetAttributes(attrs...)
}

// exposeMetrics sends all collected metrics to the Prometheus channel.
func (c *NbuCollector) exposeMetrics(ch chan<- prometheus.Metric, storageMetrics, jobsSize, jobsCount, jobsStatusCount map[string]float64) {
	c.exposeStorageMetrics(ch, storageMetrics)
	c.exposeJobSizeMetrics(ch, jobsSize)
	c.exposeJobCountMetrics(ch, jobsCount)
	c.exposeJobStatusMetrics(ch, jobsStatusCount)
	c.exposeAPIVersionMetric(ch)
}

// exposeStorageMetrics sends storage metrics to the Prometheus channel.
func (c *NbuCollector) exposeStorageMetrics(ch chan<- prometheus.Metric, storageMetrics map[string]float64) {
	for key, value := range storageMetrics {
		labels := strings.Split(key, "|")
		ch <- prometheus.MustNewConstMetric(
			c.nbuDiskSize,
			prometheus.GaugeValue,
			value,
			labels[0], labels[1], labels[2],
		)
	}
}

// exposeJobSizeMetrics sends job size metrics to the Prometheus channel.
func (c *NbuCollector) exposeJobSizeMetrics(ch chan<- prometheus.Metric, jobsSize map[string]float64) {
	for key, value := range jobsSize {
		labels := strings.Split(key, "|")
		ch <- prometheus.MustNewConstMetric(
			c.nbuJobsSize,
			prometheus.GaugeValue,
			value,
			labels[0], labels[1], labels[2],
		)
	}
}

// exposeJobCountMetrics sends job count metrics to the Prometheus channel.
func (c *NbuCollector) exposeJobCountMetrics(ch chan<- prometheus.Metric, jobsCount map[string]float64) {
	for key, value := range jobsCount {
		labels := strings.Split(key, "|")
		ch <- prometheus.MustNewConstMetric(
			c.nbuJobsCount,
			prometheus.GaugeValue,
			value,
			labels[0], labels[1], labels[2],
		)
	}
}

// exposeJobStatusMetrics sends job status count metrics to the Prometheus channel.
func (c *NbuCollector) exposeJobStatusMetrics(ch chan<- prometheus.Metric, jobsStatusCount map[string]float64) {
	for key, value := range jobsStatusCount {
		labels := strings.Split(key, "|")
		ch <- prometheus.MustNewConstMetric(
			c.nbuJobsStatusCount,
			prometheus.GaugeValue,
			value,
			labels[0], labels[1],
		)
	}
}

// exposeAPIVersionMetric sends the API version metric to the Prometheus channel.
func (c *NbuCollector) exposeAPIVersionMetric(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(
		c.nbuAPIVersion,
		prometheus.GaugeValue,
		1,
		c.cfg.NbuServer.APIVersion,
	)
}
