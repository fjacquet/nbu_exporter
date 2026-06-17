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
	cfg     models.Config
	tracing *TracerWrapper
	site    string // identity label value for live mode (snapshot mode uses per-site snapshot keys)

	// store is set for the multi-site snapshot-reading collector (the registered
	// collector). When non-nil, Collect reads the latest Snapshot and emits each
	// site's metrics, performing no live fetch on scrape. When nil, the collector
	// is in the legacy single-target live-fetch mode (used by tests).
	store *SnapshotStore

	client             *NbuClient
	storageCache       *StorageCache  // TTL cache for storage metrics (live mode only)
	subCollectors      []subCollector // Enabled opt-in metric collectors (live mode only)
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

	// Per-client lifecycle metrics (opt-in collectors.perClient + allowlist).
	nbuClientJobsCount   *prometheus.Desc // Job count per site/client/action/status (current window)
	nbuClientLastSuccess *prometheus.Desc // Unix ts of last successful job per site/client/policy/action
	// Persistent last-success cache, keyed by site so a client's value survives scrape
	// windows with no new jobs without leaking across sites. Bounded by the allowlist.
	clientSuccessMu   sync.RWMutex
	clientLastSuccess map[clientLastSuccessKey]float64

	// Scrape time tracking
	scrapeMu              sync.RWMutex // Protects lastStorageScrapeTime and lastJobsScrapeTime
	lastStorageScrapeTime time.Time    // Last successful storage metric collection
	lastJobsScrapeTime    time.Time    // Last successful jobs metric collection
}

// clientLastSuccessKey keys the persistent per-client last-success cache. It adds
// the site dimension to ClientSuccessKey so cached values do not collide or leak
// across NetBackup primaries in the multi-site model.
type clientLastSuccessKey struct {
	Site   string
	Client string
	Policy string
	Action string
}

// withSite prepends the site label value to a slice of additional label values.
// Used at emission time so every metric series carries the site dimension.
func withSite(site string, labels []string) []string {
	result := make([]string, 0, 1+len(labels))
	return append(append(result, site), labels...)
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

	c := newBaseCollector(cfg, tracing)
	c.client = client
	c.storageCache = NewStorageCache(cfg.GetCacheTTL())
	c.site = primarySite(cfg)

	// Populate enabled opt-in collectors from config (empty unless configured).
	c.subCollectors = buildSubCollectors(c)

	return c, nil
}

// primarySite returns the identity label value for a single-target (live-mode)
// collector: the configured primary site, falling back to the legacy host for
// configs/tests that bypass SetDefaults.
func primarySite(cfg models.Config) string {
	if len(cfg.NbuServers) > 0 {
		return cfg.NbuServers[0].Site
	}
	return cfg.NbuServer.Host
}

// NewSnapshotCollector creates the multi-site collector that exposes metrics from
// the latest Snapshot published by the background collection loop. It performs no
// live fetching on scrape: Collect reads the SnapshotStore and emits each site's
// metrics. This is the collector registered with Prometheus in production.
//
// cfg supplies only descriptor metadata (e.g. help strings); all metric values
// come from the snapshot, so per-site connection details live in the targets.
func NewSnapshotCollector(cfg models.Config, store *SnapshotStore, opts ...CollectorOption) *NbuCollector {
	options := defaultCollectorOptions()
	for _, opt := range opts {
		opt(&options)
	}
	tracing := NewTracerWrapper(options.tracerProvider, "nbu-exporter/collector")
	c := newBaseCollector(cfg, tracing)
	c.store = store
	return c
}

// newBaseCollector builds an NbuCollector with its configuration, tracer, and all
// Prometheus metric descriptors set, but without a client, cache, store, or
// sub-collectors. Both constructors share it so the descriptor set stays identical.
func newBaseCollector(cfg models.Config, tracing *TracerWrapper) *NbuCollector {
	return &NbuCollector{
		cfg:     cfg,
		tracing: tracing,
		nbuResponseTime: prometheus.NewDesc(
			"nbu_response_time_ms",
			"The server response time in milliseconds",
			[]string{"site"}, nil,
		),
		nbuDiskSize: prometheus.NewDesc(
			"nbu_disk_bytes",
			fmt.Sprintf("The quantity of storage bytes (cached: %s TTL)", cfg.GetCacheTTL()),
			[]string{"site", "name", "type", "size"}, nil,
		),
		nbuJobsSize: prometheus.NewDesc(
			"nbu_jobs_bytes",
			"The quantity of processed bytes",
			[]string{"site", "action", "policy_type", "status"}, nil,
		),
		nbuJobsCount: prometheus.NewDesc(
			"nbu_jobs_count",
			"The quantity of jobs",
			[]string{"site", "action", "policy_type", "status"}, nil,
		),
		nbuJobsStatusCount: prometheus.NewDesc(
			"nbu_status_count",
			"The quantity per status",
			[]string{"site", "action", "status"}, nil,
		),
		nbuAPIVersion: prometheus.NewDesc(
			"nbu_api_version",
			"The NetBackup API version currently in use",
			[]string{"site", "version"}, nil,
		),
		nbuUp: prometheus.NewDesc(
			"nbu_up",
			"1 if NetBackup API is reachable, 0 if all collections failed",
			[]string{"site"}, nil,
		),
		nbuLastScrapeTime: prometheus.NewDesc(
			"nbu_last_scrape_timestamp_seconds",
			"Unix timestamp of the last successful metric collection",
			[]string{"site", "source"}, nil, // source: "storage" or "jobs"
		),
		nbuDiskCapacity: prometheus.NewDesc(
			"nbu_disk_capacity_bytes",
			"Authoritative total capacity of the storage unit in bytes",
			[]string{"site", "name", "type"}, nil,
		),
		nbuStorageMaxJobs: prometheus.NewDesc(
			"nbu_storage_max_concurrent_jobs",
			"Maximum number of concurrent jobs the storage unit accepts",
			[]string{"site", "name", "type"}, nil,
		),
		nbuStorageMaxFragment: prometheus.NewDesc(
			"nbu_storage_max_fragment_size_bytes",
			"Maximum fragment size the storage unit accepts, in bytes",
			[]string{"site", "name", "type"}, nil,
		),
		nbuStorageInfo: prometheus.NewDesc(
			"nbu_storage_info",
			"Storage unit capabilities (always 1; metadata carried in labels)",
			[]string{"site", "name", "type", "subtype", "is_cloud", "worm_capable", "use_worm", "replication_capable", "instant_access"}, nil,
		),
		nbuJobsStateCount: prometheus.NewDesc(
			"nbu_jobs_state_count",
			"The quantity of jobs per lifecycle state",
			[]string{"site", "action", "state"}, nil,
		),
		nbuJobsFilesCount: prometheus.NewDesc(
			"nbu_jobs_files_count",
			"The total number of files processed by jobs",
			[]string{"site", "action", "policy_type"}, nil,
		),
		nbuJobsDedupRatio: prometheus.NewDesc(
			"nbu_jobs_dedup_ratio",
			"The mean deduplication ratio across jobs",
			[]string{"site", "action", "policy_type"}, nil,
		),
		nbuJobsQueuedCount: prometheus.NewDesc(
			"nbu_jobs_queued_count",
			"The quantity of queued jobs per queue reason code",
			[]string{"site", "action", "reason"}, nil,
		),
		nbuJobDuration: prometheus.NewDesc(
			"nbu_job_duration_seconds",
			"Histogram of completed job durations in seconds",
			[]string{"site", "action", "policy_type"}, nil,
		),
		nbuClientJobsCount: prometheus.NewDesc(
			"nbu_client_jobs_count",
			"Number of jobs per client, action and status in the scraping window (opt-in, allowlisted clients)",
			[]string{"site", "client", "action", "status"}, nil,
		),
		nbuClientLastSuccess: prometheus.NewDesc(
			"nbu_client_last_job_success_seconds",
			"Unix timestamp of the last successful (status=0) job per client, policy and action (opt-in, allowlisted clients)",
			[]string{"site", "client", "policy", "action"}, nil,
		),
		clientLastSuccess: make(map[clientLastSuccessKey]float64),
	}
}

// buildSubCollectors returns the enabled optional collectors based on config.
func buildSubCollectors(c *NbuCollector) []subCollector {
	return buildSubCollectorsFor(c.client, c.cfg, c.site)
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
	ch <- c.nbuClientJobsCount
	ch <- c.nbuClientLastSuccess
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
	// Multi-site mode: read the latest snapshot and emit per-site (no live fetch).
	if c.store != nil {
		c.collectFromSnapshot(ch)
		return
	}

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
	c.exposeMetrics(ch, c.site, storageMetrics, storageUnits, jobAgg, storageErr, jobsErr)

	// Run enabled opt-in sub-collectors (graceful degradation: errors logged, never propagated).
	runSubCollectors(ctx, c.subCollectors, ch, c.tracing)

	log.Debugf("Collected %d storage, %d storage units, %d job metric keys",
		len(storageMetrics), len(storageUnits), jobMetricCount)
}

// collectFromSnapshot emits metrics for every site in the latest snapshot
// published by the background collection loop. If no snapshot exists yet (first
// collection still in flight), it emits nothing: /metrics is up but empty until
// the first cycle completes (serve-before-first-collect).
func (c *NbuCollector) collectFromSnapshot(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), collectionTimeout)
	defer cancel()
	_, span := c.createScrapeSpan(ctx)
	defer span.End()

	snap := c.store.Load()
	if snap == nil {
		span.SetStatus(codes.Ok, "no snapshot yet")
		return
	}
	for site, ss := range snap.Sites {
		c.exposeSnapshot(ch, site, ss)
	}
	span.SetStatus(codes.Ok, "")
}

// exposeSnapshot emits one site's metrics from its already-aggregated SiteSnapshot.
// Every series carries the site label (first), and a failed site still emits
// nbu_up{site}=0 without affecting the others (graceful degradation).
func (c *NbuCollector) exposeSnapshot(ch chan<- prometheus.Metric, site string, ss *SiteSnapshot) {
	if ss == nil {
		return
	}

	upValue := 0.0
	if ss.Up {
		upValue = 1.0
	}
	ch <- prometheus.MustNewConstMetric(c.nbuUp, prometheus.GaugeValue, upValue, site)

	// Per-site degradation contract: a fully-down site exposes ONLY nbu_up=0, so
	// no stale/misleading series (e.g. nbu_api_version) are published for it.
	if !ss.Up {
		return
	}

	if !ss.LastStorageScrape.IsZero() {
		ch <- prometheus.MustNewConstMetric(c.nbuLastScrapeTime, prometheus.GaugeValue,
			float64(ss.LastStorageScrape.Unix()), site, "storage")
	}
	if !ss.LastJobsScrape.IsZero() {
		ch <- prometheus.MustNewConstMetric(c.nbuLastScrapeTime, prometheus.GaugeValue,
			float64(ss.LastJobsScrape.Unix()), site, "jobs")
	}

	c.exposeStorageMetrics(ch, site, ss.StorageMetrics)
	c.exposeStorageUnitMetrics(ch, site, ss.StorageUnits)
	c.exposeJobAggregateMetrics(ch, site, ss.JobAgg)

	ch <- prometheus.MustNewConstMetric(c.nbuAPIVersion, prometheus.GaugeValue, 1, site, ss.APIVersion)

	// Re-emit the buffered opt-in sub-collector metrics (already site-labelled).
	for _, m := range ss.SubMetrics {
		ch <- m
	}
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
func (c *NbuCollector) exposeMetrics(ch chan<- prometheus.Metric, site string, storageMetrics []StorageMetricValue, storageUnits []StorageUnitInfo, jobAgg *JobAggregator, storageErr, jobsErr error) {
	// Expose nbu_up metric first (Prometheus convention)
	// up=1 if ANY collection succeeded, up=0 if ALL failed
	upValue := 0.0
	if storageErr == nil || jobsErr == nil {
		upValue = 1.0
	}
	ch <- prometheus.MustNewConstMetric(c.nbuUp, prometheus.GaugeValue, upValue, site)

	// Expose last scrape timestamps for staleness detection
	c.scrapeMu.RLock()
	if !c.lastStorageScrapeTime.IsZero() {
		ch <- prometheus.MustNewConstMetric(
			c.nbuLastScrapeTime,
			prometheus.GaugeValue,
			float64(c.lastStorageScrapeTime.Unix()),
			site, "storage",
		)
	}
	if !c.lastJobsScrapeTime.IsZero() {
		ch <- prometheus.MustNewConstMetric(
			c.nbuLastScrapeTime,
			prometheus.GaugeValue,
			float64(c.lastJobsScrapeTime.Unix()),
			site, "jobs",
		)
	}
	c.scrapeMu.RUnlock()

	// Expose existing storage + job metrics
	c.exposeStorageMetrics(ch, site, storageMetrics)
	c.exposeStorageUnitMetrics(ch, site, storageUnits)
	c.exposeJobAggregateMetrics(ch, site, jobAgg)
	c.exposeAPIVersionMetric(ch, site)
}

// exposeStorageUnitMetrics emits the per-unit storage attribute metrics:
// capacity, max concurrent jobs, max fragment size, and the info metric.
func (c *NbuCollector) exposeStorageUnitMetrics(ch chan<- prometheus.Metric, site string, units []StorageUnitInfo) {
	for _, u := range units {
		labels := withSite(site, []string{u.Name, u.Type})
		ch <- prometheus.MustNewConstMetric(c.nbuDiskCapacity, prometheus.GaugeValue, u.TotalCapacityBytes, labels...)
		ch <- prometheus.MustNewConstMetric(c.nbuStorageMaxJobs, prometheus.GaugeValue, u.MaxConcurrentJobs, labels...)
		ch <- prometheus.MustNewConstMetric(c.nbuStorageMaxFragment, prometheus.GaugeValue, u.MaxFragmentBytes, labels...)
		ch <- prometheus.MustNewConstMetric(c.nbuStorageInfo, prometheus.GaugeValue, 1, withSite(site, u.InfoLabels())...)
	}
}

// exposeJobAggregateMetrics emits every job-derived metric from the aggregator.
// A nil aggregator (jobs fetch failed) yields no job metrics, preserving the
// graceful-degradation contract. Dedup ratio is emitted only when at least one
// job contributed (absent, never a fake zero).
func (c *NbuCollector) exposeJobAggregateMetrics(ch chan<- prometheus.Metric, site string, agg *JobAggregator) {
	if agg == nil {
		return
	}

	for key, value := range agg.Size {
		ch <- prometheus.MustNewConstMetric(c.nbuJobsSize, prometheus.GaugeValue, value, withSite(site, key.Labels())...)
	}
	for key, value := range agg.Count {
		ch <- prometheus.MustNewConstMetric(c.nbuJobsCount, prometheus.GaugeValue, value, withSite(site, key.Labels())...)
	}
	for key, value := range agg.StatusCount {
		ch <- prometheus.MustNewConstMetric(c.nbuJobsStatusCount, prometheus.GaugeValue, value, withSite(site, key.Labels())...)
	}
	for key, value := range agg.StateCount {
		ch <- prometheus.MustNewConstMetric(c.nbuJobsStateCount, prometheus.GaugeValue, value, withSite(site, key.Labels())...)
	}
	for key, value := range agg.FilesCount {
		ch <- prometheus.MustNewConstMetric(c.nbuJobsFilesCount, prometheus.GaugeValue, value, withSite(site, key.Labels())...)
	}
	for key, value := range agg.QueuedCount {
		ch <- prometheus.MustNewConstMetric(c.nbuJobsQueuedCount, prometheus.GaugeValue, value, withSite(site, key.Labels())...)
	}
	for key, sum := range agg.DedupSum {
		if n := agg.DedupCount[key]; n > 0 {
			ch <- prometheus.MustNewConstMetric(c.nbuJobsDedupRatio, prometheus.GaugeValue, sum/n, withSite(site, key.Labels())...)
		}
	}
	for key, acc := range agg.Duration {
		ch <- prometheus.MustNewConstHistogram(c.nbuJobDuration, acc.Count, acc.Sum, acc.Buckets(), withSite(site, key.Labels())...)
	}

	c.exposePerClientMetrics(ch, site, agg)
}

// exposePerClientMetrics emits the opt-in per-client lifecycle metrics for one site.
// The aggregator only ever holds allowlisted clients (gated at the source), so the
// emitted cardinality and the persistent last-success cache stay bounded by the
// allowlist. The cache is keyed by site so a value survives scrape windows with no
// new jobs without leaking across sites.
func (c *NbuCollector) exposePerClientMetrics(ch chan<- prometheus.Metric, site string, agg *JobAggregator) {
	for key, value := range agg.ClientCount {
		ch <- prometheus.MustNewConstMetric(c.nbuClientJobsCount, prometheus.GaugeValue, value,
			site, key.Client, key.Action, key.Status)
	}

	c.clientSuccessMu.Lock()
	for key, ts := range agg.ClientLastSuccess {
		ck := clientLastSuccessKey{Site: site, Client: key.Client, Policy: key.Policy, Action: key.Action}
		if existing, ok := c.clientLastSuccess[ck]; !ok || ts > existing {
			c.clientLastSuccess[ck] = ts
		}
	}
	c.clientSuccessMu.Unlock()

	c.clientSuccessMu.RLock()
	for ck, ts := range c.clientLastSuccess {
		if ck.Site != site {
			continue
		}
		ch <- prometheus.MustNewConstMetric(c.nbuClientLastSuccess, prometheus.GaugeValue, ts,
			ck.Site, ck.Client, ck.Policy, ck.Action)
	}
	c.clientSuccessMu.RUnlock()
}

// exposeStorageMetrics sends storage metrics to the Prometheus channel.
// Uses Labels() method directly - no string parsing needed.
func (c *NbuCollector) exposeStorageMetrics(ch chan<- prometheus.Metric, site string, storageMetrics []StorageMetricValue) {
	for _, m := range storageMetrics {
		ch <- prometheus.MustNewConstMetric(
			c.nbuDiskSize,
			prometheus.GaugeValue,
			m.Value,
			withSite(site, m.Key.Labels())...,
		)
	}
}

// exposeAPIVersionMetric sends the API version metric to the Prometheus channel.
func (c *NbuCollector) exposeAPIVersionMetric(ch chan<- prometheus.Metric, site string) {
	ch <- prometheus.MustNewConstMetric(
		c.nbuAPIVersion,
		prometheus.GaugeValue,
		1,
		site, c.cfg.NbuServer.APIVersion,
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
