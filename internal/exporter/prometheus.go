// Package exporter implements the Prometheus Collector interface for NetBackup metrics.
// It collects storage and job statistics from the NetBackup API and exposes them
// in Prometheus format.
package exporter

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/fjacquet/nbu_exporter/internal/models"
	"github.com/fjacquet/nbu_exporter/internal/telemetry"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
)

const collectionTimeout = 2 * time.Minute // Maximum time allowed for metric collection

// CollectorOption configures optional NbuCollector settings.
type CollectorOption func(*collectorOptions)

type collectorOptions struct {
	tracerProvider trace.TracerProvider
	apiTrace       bool
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

// WithCollectorAPITrace enables NetBackup API response-body trace logging on
// the collector's HTTP client (see WithAPITrace). Used by the --trace flag for
// live-appliance payload validation.
func WithCollectorAPITrace(enabled bool) CollectorOption {
	return func(o *collectorOptions) {
		o.apiTrace = enabled
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
//
// Storage metrics are cached to reduce NetBackup API load. The cache TTL is
// configurable via Config.Server.CacheTTL (default 5m).
type NbuCollector struct {
	cfg                models.Config
	client             *NbuClient
	tracing            *TracerWrapper
	storageCache       *StorageCache  // TTL cache for storage metrics
	subCollectors      []subCollector // Enabled opt-in metric collectors
	nbuDiskSize        *prometheus.Desc
	nbuResponseTime    *prometheus.Desc
	nbuJobsSize        *prometheus.Desc
	nbuJobsCount       *prometheus.Desc
	nbuJobsStatusCount *prometheus.Desc
	nbuAPIVersion      *prometheus.Desc
	nbuUp              *prometheus.Desc // 1 if NetBackup API is reachable, 0 if all collections failed
	nbuLastScrapeTime  *prometheus.Desc // Unix timestamp of last successful collection

	// Storage attribute metrics (derived from the same storage API response)
	nbuDiskCapacity       *prometheus.Desc // Authoritative total capacity in bytes
	nbuStorageMaxJobs     *prometheus.Desc // Max concurrent jobs per storage unit
	nbuStorageMaxFragment *prometheus.Desc // Max fragment size in bytes per storage unit
	nbuStorageInfo        *prometheus.Desc // Storage unit capability info (value 1)

	// Job attribute metrics (derived from the jobs API response)
	nbuJobsStateCount  *prometheus.Desc // Job count per action/state
	nbuJobsFilesCount  *prometheus.Desc // Sum of files per action/policy
	nbuJobsDedupRatio  *prometheus.Desc // Mean dedup ratio per action/policy
	nbuJobsQueuedCount *prometheus.Desc // Queued job count per action/reason
	nbuJobDuration     *prometheus.Desc // Job duration histogram per action/policy

	// Scrape time tracking
	scrapeMu              sync.RWMutex // Protects lastStorageScrapeTime and lastJobsScrapeTime
	lastStorageScrapeTime time.Time    // Last successful storage metric collection
	lastJobsScrapeTime    time.Time    // Last successful jobs metric collection
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
//   - Detection tries versions in descending order: 14.0 → 13.0 → 12.0 → 10.0
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

	// Create base client with same TracerProvider and API trace setting
	client := NewNbuClient(cfg, WithTracerProvider(options.tracerProvider), WithAPITrace(options.apiTrace))

	// Perform version detection if needed (reuses shared logic)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := performVersionDetectionIfNeeded(ctx, client, &cfg); err != nil {
		return nil, err
	}

	// Create TracerWrapper for collector
	tracing := NewTracerWrapper(options.tracerProvider, "nbu-exporter/collector")

	// Create storage cache with configured TTL
	storageCache := NewStorageCache(cfg.GetCacheTTL())

	c := &NbuCollector{
		cfg:          cfg,
		client:       client,
		tracing:      tracing,
		storageCache: storageCache,
		nbuResponseTime: prometheus.NewDesc(
			"nbu_response_time_ms",
			"The server response time in milliseconds",
			nil, nil,
		),
		nbuDiskSize: prometheus.NewDesc(
			"nbu_disk_bytes",
			fmt.Sprintf("The quantity of storage bytes (cached: %s TTL)", cfg.GetCacheTTL()),
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
		nbuUp: prometheus.NewDesc(
			"nbu_up",
			"1 if NetBackup API is reachable, 0 if all collections failed",
			nil, nil,
		),
		nbuLastScrapeTime: prometheus.NewDesc(
			"nbu_last_scrape_timestamp_seconds",
			"Unix timestamp of the last successful metric collection",
			[]string{"source"}, nil, // source: "storage" or "jobs"
		),
		nbuDiskCapacity: prometheus.NewDesc(
			"nbu_disk_capacity_bytes",
			"Authoritative total capacity of the storage unit in bytes",
			[]string{"name", "type"}, nil,
		),
		nbuStorageMaxJobs: prometheus.NewDesc(
			"nbu_storage_max_concurrent_jobs",
			"Maximum number of concurrent jobs the storage unit accepts",
			[]string{"name", "type"}, nil,
		),
		nbuStorageMaxFragment: prometheus.NewDesc(
			"nbu_storage_max_fragment_size_bytes",
			"Maximum fragment size the storage unit accepts, in bytes",
			[]string{"name", "type"}, nil,
		),
		nbuStorageInfo: prometheus.NewDesc(
			"nbu_storage_info",
			"Storage unit capabilities (always 1; metadata carried in labels)",
			[]string{"name", "type", "subtype", "is_cloud", "worm_capable", "use_worm", "replication_capable", "instant_access"}, nil,
		),
		nbuJobsStateCount: prometheus.NewDesc(
			"nbu_jobs_state_count",
			"The quantity of jobs per lifecycle state",
			[]string{"action", "state"}, nil,
		),
		nbuJobsFilesCount: prometheus.NewDesc(
			"nbu_jobs_files_count",
			"The total number of files processed by jobs",
			[]string{"action", "policy_type"}, nil,
		),
		nbuJobsDedupRatio: prometheus.NewDesc(
			"nbu_jobs_dedup_ratio",
			"The mean deduplication ratio across jobs",
			[]string{"action", "policy_type"}, nil,
		),
		nbuJobsQueuedCount: prometheus.NewDesc(
			"nbu_jobs_queued_count",
			"The quantity of queued jobs per queue reason code",
			[]string{"action", "reason"}, nil,
		),
		nbuJobDuration: prometheus.NewDesc(
			"nbu_job_duration_seconds",
			"Histogram of completed job durations in seconds",
			[]string{"action", "policy_type"}, nil,
		),
	}

	// Populate enabled opt-in collectors from config (empty until Phase 4).
	c.subCollectors = buildSubCollectors(c)

	return c, nil
}

// buildSubCollectors returns the enabled optional collectors based on config.
func buildSubCollectors(c *NbuCollector) []subCollector {
	var subs []subCollector
	if c.cfg.Collectors.Alerts.Enabled {
		subs = append(subs, newAlertsCollector(c.client, c.cfg))
	}
	if c.cfg.Collectors.Malware.Enabled {
		subs = append(subs, newMalwareCollector(c.client, c.cfg))
	}
	if c.cfg.Collectors.Catalog.Enabled {
		subs = append(subs, newCatalogCollector(c.client, c.cfg))
	}
	if c.cfg.Collectors.SLO.Enabled {
		subs = append(subs, newSLOCollector(c.client, c.cfg))
	}
	return subs
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
	ch <- c.nbuUp
	ch <- c.nbuLastScrapeTime
	ch <- c.nbuDiskCapacity
	ch <- c.nbuStorageMaxJobs
	ch <- c.nbuStorageMaxFragment
	ch <- c.nbuStorageInfo
	ch <- c.nbuJobsStateCount
	ch <- c.nbuJobsFilesCount
	ch <- c.nbuJobsDedupRatio
	ch <- c.nbuJobsQueuedCount
	ch <- c.nbuJobDuration
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
	storageMetrics, storageUnits, jobAgg, storageErr, jobsErr := c.collectAllMetrics(ctx, span)

	jobMetricCount := 0
	if jobAgg != nil {
		jobMetricCount = len(jobAgg.Size)
	}

	// Update span with scrape results
	c.updateScrapeSpan(span, scrapeStart, storageMetrics, jobMetricCount, storageErr, jobsErr)

	// Expose all collected metrics (pass errors for nbu_up calculation)
	c.exposeMetrics(ch, storageMetrics, storageUnits, jobAgg, storageErr, jobsErr)

	// Run enabled opt-in sub-collectors (graceful degradation: errors logged, never propagated).
	runSubCollectors(ctx, c.subCollectors, ch, c.tracing)

	log.Debugf("Collected %d storage, %d storage units, %d job metric keys",
		len(storageMetrics), len(storageUnits), jobMetricCount)
}

// collectAllMetrics fetches storage and job metrics from NetBackup API in parallel.
// Returns the collected typed metric slices and any errors encountered.
//
// Parallel Execution:
// Storage and job metrics are fetched concurrently using errgroup, reducing total
// scrape time from (storage_time + jobs_time) to max(storage_time, jobs_time).
//
// Graceful Degradation:
// Each goroutine returns nil to errgroup regardless of fetch errors. This ensures:
//   - Storage failure does not cancel job fetching
//   - Job failure does not cancel storage fetching
//   - Errors are tracked in storageErr/jobsErr for the caller to handle
//   - Partial metrics are still collected when one source fails
func (c *NbuCollector) collectAllMetrics(ctx context.Context, span trace.Span) (
	storageMetrics []StorageMetricValue,
	storageUnits []StorageUnitInfo,
	jobAgg *JobAggregator,
	storageErr, jobsErr error,
) {
	// Create errgroup with context for coordinated cancellation
	// Note: We don't use the error from g.Wait() because we always return nil
	// from goroutines to maintain graceful degradation behavior.
	g, gCtx := errgroup.WithContext(ctx)

	// Collect storage metrics in parallel
	g.Go(func() error {
		storageMetrics, storageErr = c.collectStorageMetrics(gCtx, span)
		// Return nil to continue even if storage fails
		// This maintains graceful degradation: job collection continues
		return nil
	})

	// Collect job metrics in parallel
	g.Go(func() error {
		jobAgg, jobsErr = c.collectJobMetrics(gCtx, span)
		// Return nil to continue even if jobs fail
		// This maintains graceful degradation: storage collection continues
		return nil
	})

	// Wait for both goroutines to complete
	// Since we always return nil, g.Wait() only returns nil or context error
	_ = g.Wait()

	// Per-unit storage attributes share the storage cache TTL; read them back so
	// both cache-hit and cache-miss paths expose the same data set.
	if storageErr == nil {
		storageUnits, _ = c.storageCache.GetUnits()
	}

	return
}

// collectStorageMetrics fetches storage metrics with caching.
// Returns cached metrics if available (cache hit), otherwise fetches from API.
// Records cache hit/miss events in the span for observability.
// On success, updates lastStorageScrapeTime for staleness tracking.
func (c *NbuCollector) collectStorageMetrics(ctx context.Context, span trace.Span) ([]StorageMetricValue, error) {
	// Check cache first
	if metrics, found := c.storageCache.Get(); found {
		log.Debug("Cache hit for storage metrics")
		span.AddEvent("cache_hit", trace.WithAttributes(
			attribute.String("cache_type", "storage"),
		))
		// Cache hit counts as success for staleness tracking
		c.scrapeMu.Lock()
		c.lastStorageScrapeTime = time.Now()
		c.scrapeMu.Unlock()
		return metrics, nil
	}

	// Cache miss - fetch from API
	log.Debug("Cache miss for storage metrics, fetching from API")
	span.AddEvent("cache_miss", trace.WithAttributes(
		attribute.String("cache_type", "storage"),
	))

	metrics, units, err := FetchStorageFull(ctx, c.client)
	if err != nil {
		log.Errorf("Failed to fetch storage metrics: %v", err)
		c.recordFetchError(span, "storage_fetch_error", err)
		return nil, err
	}

	// Store metrics and per-unit attributes in cache (same TTL) and update last scrape time
	c.storageCache.Set(metrics)
	c.storageCache.SetUnits(units)
	c.scrapeMu.Lock()
	c.lastStorageScrapeTime = time.Now()
	c.scrapeMu.Unlock()
	return metrics, nil
}

// collectJobMetrics fetches job metrics and records errors in span.
// On success, updates lastJobsScrapeTime for staleness tracking.
func (c *NbuCollector) collectJobMetrics(ctx context.Context, span trace.Span) (*JobAggregator, error) {
	agg, err := FetchAllJobsFull(ctx, c.client, c.cfg.Server.ScrapingInterval)
	if err != nil {
		log.Errorf("Failed to fetch job metrics: %v", err)
		c.recordFetchError(span, "jobs_fetch_error", err)
		return nil, err
	}

	// Update last scrape time on success
	c.scrapeMu.Lock()
	c.lastJobsScrapeTime = time.Now()
	c.scrapeMu.Unlock()
	return agg, nil
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
func (c *NbuCollector) updateScrapeSpan(span trace.Span, scrapeStart time.Time, storageMetrics []StorageMetricValue, jobMetricCount int, storageErr, jobsErr error) {
	scrapeStatus := c.determineScrapeStatus(storageErr, jobsErr)
	c.setSpanStatus(span, scrapeStatus)
	c.recordScrapeAttributes(span, scrapeStart, storageMetrics, jobMetricCount, scrapeStatus)
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
func (c *NbuCollector) recordScrapeAttributes(span trace.Span, scrapeStart time.Time, storageMetrics []StorageMetricValue, jobMetricCount int, scrapeStatus string) {
	scrapeDuration := time.Since(scrapeStart)
	attrs := []attribute.KeyValue{
		attribute.Float64(telemetry.AttrScrapeDurationMS, float64(scrapeDuration.Milliseconds())),
		attribute.Int(telemetry.AttrScrapeStorageMetricsCount, len(storageMetrics)),
		attribute.Int(telemetry.AttrScrapeJobMetricsCount, jobMetricCount),
		attribute.String(telemetry.AttrScrapeStatus, scrapeStatus),
	}
	span.SetAttributes(attrs...)
}

// exposeMetrics sends all collected metrics to the Prometheus channel.
// It calculates nbu_up based on collection errors: 1 if any collection succeeded, 0 if all failed.
// It also exposes nbu_last_scrape_timestamp_seconds for staleness detection.
func (c *NbuCollector) exposeMetrics(ch chan<- prometheus.Metric, storageMetrics []StorageMetricValue, storageUnits []StorageUnitInfo, jobAgg *JobAggregator, storageErr, jobsErr error) {
	// Expose nbu_up metric first (Prometheus convention)
	// up=1 if ANY collection succeeded, up=0 if ALL failed
	upValue := 0.0
	if storageErr == nil || jobsErr == nil {
		upValue = 1.0
	}
	ch <- prometheus.MustNewConstMetric(c.nbuUp, prometheus.GaugeValue, upValue)

	// Expose last scrape timestamps for staleness detection
	c.scrapeMu.RLock()
	if !c.lastStorageScrapeTime.IsZero() {
		ch <- prometheus.MustNewConstMetric(
			c.nbuLastScrapeTime,
			prometheus.GaugeValue,
			float64(c.lastStorageScrapeTime.Unix()),
			"storage",
		)
	}
	if !c.lastJobsScrapeTime.IsZero() {
		ch <- prometheus.MustNewConstMetric(
			c.nbuLastScrapeTime,
			prometheus.GaugeValue,
			float64(c.lastJobsScrapeTime.Unix()),
			"jobs",
		)
	}
	c.scrapeMu.RUnlock()

	// Expose existing storage + job metrics
	c.exposeStorageMetrics(ch, storageMetrics)
	c.exposeStorageUnitMetrics(ch, storageUnits)
	c.exposeJobAggregateMetrics(ch, jobAgg)
	c.exposeAPIVersionMetric(ch)
}

// exposeStorageUnitMetrics emits the per-unit storage attribute metrics:
// capacity, max concurrent jobs, max fragment size, and the info metric.
func (c *NbuCollector) exposeStorageUnitMetrics(ch chan<- prometheus.Metric, units []StorageUnitInfo) {
	for _, u := range units {
		labels := []string{u.Name, u.Type}
		ch <- prometheus.MustNewConstMetric(c.nbuDiskCapacity, prometheus.GaugeValue, u.TotalCapacityBytes, labels...)
		ch <- prometheus.MustNewConstMetric(c.nbuStorageMaxJobs, prometheus.GaugeValue, u.MaxConcurrentJobs, labels...)
		ch <- prometheus.MustNewConstMetric(c.nbuStorageMaxFragment, prometheus.GaugeValue, u.MaxFragmentBytes, labels...)
		ch <- prometheus.MustNewConstMetric(c.nbuStorageInfo, prometheus.GaugeValue, 1, u.InfoLabels()...)
	}
}

// exposeJobAggregateMetrics emits every job-derived metric from the aggregator.
// A nil aggregator (jobs fetch failed) yields no job metrics, preserving the
// graceful-degradation contract. Dedup ratio is emitted only when at least one
// job contributed (absent, never a fake zero).
func (c *NbuCollector) exposeJobAggregateMetrics(ch chan<- prometheus.Metric, agg *JobAggregator) {
	if agg == nil {
		return
	}

	for key, value := range agg.Size {
		ch <- prometheus.MustNewConstMetric(c.nbuJobsSize, prometheus.GaugeValue, value, key.Labels()...)
	}
	for key, value := range agg.Count {
		ch <- prometheus.MustNewConstMetric(c.nbuJobsCount, prometheus.GaugeValue, value, key.Labels()...)
	}
	for key, value := range agg.StatusCount {
		ch <- prometheus.MustNewConstMetric(c.nbuJobsStatusCount, prometheus.GaugeValue, value, key.Labels()...)
	}
	for key, value := range agg.StateCount {
		ch <- prometheus.MustNewConstMetric(c.nbuJobsStateCount, prometheus.GaugeValue, value, key.Labels()...)
	}
	for key, value := range agg.FilesCount {
		ch <- prometheus.MustNewConstMetric(c.nbuJobsFilesCount, prometheus.GaugeValue, value, key.Labels()...)
	}
	for key, value := range agg.QueuedCount {
		ch <- prometheus.MustNewConstMetric(c.nbuJobsQueuedCount, prometheus.GaugeValue, value, key.Labels()...)
	}
	for key, sum := range agg.DedupSum {
		if n := agg.DedupCount[key]; n > 0 {
			ch <- prometheus.MustNewConstMetric(c.nbuJobsDedupRatio, prometheus.GaugeValue, sum/n, key.Labels()...)
		}
	}
	for key, acc := range agg.Duration {
		ch <- prometheus.MustNewConstHistogram(c.nbuJobDuration, acc.Count, acc.Sum, acc.Buckets(), key.Labels()...)
	}
}

// exposeStorageMetrics sends storage metrics to the Prometheus channel.
// Uses Labels() method directly - no string parsing needed.
func (c *NbuCollector) exposeStorageMetrics(ch chan<- prometheus.Metric, storageMetrics []StorageMetricValue) {
	for _, m := range storageMetrics {
		ch <- prometheus.MustNewConstMetric(
			c.nbuDiskSize,
			prometheus.GaugeValue,
			m.Value,
			m.Key.Labels()...,
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

// Close releases resources associated with the collector, including
// closing the internal HTTP client connections.
//
// Shutdown Order (documented):
//  1. Stop accepting new scrapes (HTTP server stopped first)
//  2. Wait for active Collect() calls to complete
//  3. Close NbuClient (drains API connections)
//  4. Shutdown OpenTelemetry (flush traces)
//
// This method should be called during graceful shutdown after the
// HTTP server has stopped accepting requests.
//
// Returns an error if client closure fails.
func (c *NbuCollector) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// CloseWithContext releases resources with explicit timeout control.
// Use this when you need custom shutdown timeout behavior.
//
// Parameters:
//   - ctx: Context for shutdown timeout
//
// Returns an error if client closure fails or context is cancelled.
func (c *NbuCollector) CloseWithContext(ctx context.Context) error {
	if c.client != nil {
		return c.client.CloseWithContext(ctx)
	}
	return nil
}

// GetStorageCache returns the storage cache for management (e.g., flush on config reload).
// This allows external code to clear the cache when the NBU server configuration changes.
func (c *NbuCollector) GetStorageCache() *StorageCache {
	return c.storageCache
}
