package exporter

import (
	"context"
	"runtime"
	"sync"
	"time"

	"github.com/fjacquet/nbu_exporter/internal/models"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
)

// TargetCollector collects one site (one NetBackup primary) into a SiteSnapshot.
type TargetCollector struct {
	site           string
	cfg            models.Config // a single-server view: cfg.NbuServer set from this site's entry
	clientMu       sync.Mutex    // guards client (lazy build vs. close)
	client         *NbuClient    // built lazily with version detection; nil until first success
	tracing        *TracerWrapper
	tracerProvider trace.TracerProvider // threaded into the lazily-built client
	apiTrace       bool                 // --trace: log API response bodies on this site's client
}

// TargetOption configures optional per-target settings (tracing, API trace).
type TargetOption func(*TargetCollector)

// WithTargetTracerProvider wires an OpenTelemetry TracerProvider into the target's
// client and sub-collector spans. Without it, tracing is a noop.
func WithTargetTracerProvider(tp trace.TracerProvider) TargetOption {
	return func(tc *TargetCollector) {
		tc.tracerProvider = tp
		tc.tracing = NewTracerWrapper(tp, "nbu-exporter/target")
	}
}

// WithTargetAPITrace enables NetBackup API response-body trace logging on the
// target's client (the --trace flag).
func WithTargetAPITrace(enabled bool) TargetOption {
	return func(tc *TargetCollector) { tc.apiTrace = enabled }
}

// NewTargetCollector builds a per-site collector. base supplies Server.* settings;
// entry supplies this site's NetBackup server fields.
func NewTargetCollector(base models.Config, entry models.NbuServerConfig, opts ...TargetOption) *TargetCollector {
	cfg := base
	cfg.NbuServer.Host = entry.Host
	cfg.NbuServer.Port = entry.Port
	cfg.NbuServer.Scheme = entry.Scheme
	cfg.NbuServer.URI = entry.URI
	cfg.NbuServer.Domain = entry.Domain
	cfg.NbuServer.DomainType = entry.DomainType
	cfg.NbuServer.APIKey = entry.APIKey
	cfg.NbuServer.APIVersion = entry.APIVersion
	cfg.NbuServer.ContentType = entry.ContentType
	cfg.NbuServer.InsecureSkipVerify = entry.InsecureSkipVerify
	tc := &TargetCollector{
		site:    entry.Site,
		cfg:     cfg,
		tracing: NewTracerWrapper(nil, "nbu-exporter/target"),
	}
	for _, opt := range opts {
		opt(tc)
	}
	return tc
}

func (tc *TargetCollector) clientFor(ctx context.Context) (*NbuClient, error) {
	tc.clientMu.Lock()
	defer tc.clientMu.Unlock()
	if tc.client != nil {
		return tc.client, nil
	}
	c, err := NewNbuClientWithVersionDetection(ctx, &tc.cfg,
		WithTracerProvider(tc.tracerProvider), WithAPITrace(tc.apiTrace))
	if err != nil {
		return nil, err
	}
	tc.client = c
	return c, nil
}

// jobWindow returns the job lookback window for one collection: the larger of the
// configured scrapingInterval and collectionInterval. Because the loop polls every
// collectionInterval, the window must be at least that long so each poll covers
// the time since the previous one (no gaps in job coverage).
func (tc *TargetCollector) jobWindow() string {
	return maxDurationString(tc.cfg.Server.ScrapingInterval, tc.cfg.Server.CollectionInterval)
}

// maxDurationString returns whichever of two duration strings parses to the larger
// duration. If exactly one fails to parse, the other is returned. (Both inputs are
// validated/defaulted upstream, so the both-unparseable case is unreachable.)
func maxDurationString(a, b string) string {
	da, ea := time.ParseDuration(a)
	db, eb := time.ParseDuration(b)
	switch {
	case ea != nil:
		return b
	case eb != nil:
		return a
	case db > da:
		return b
	default:
		return a
	}
}

// close releases this target's client connections, if a client was built.
func (tc *TargetCollector) close() error {
	tc.clientMu.Lock()
	defer tc.clientMu.Unlock()
	if tc.client != nil {
		return tc.client.Close()
	}
	return nil
}

// collect fetches storage + jobs for this site and returns a SiteSnapshot. It never
// returns an error: failures are captured per-source so other sites are unaffected.
func (tc *TargetCollector) collect(ctx context.Context) *SiteSnapshot {
	ss := &SiteSnapshot{Site: tc.site}
	client, err := tc.clientFor(ctx)
	if err != nil {
		ss.Up = false
		ss.StorageErr = err
		ss.JobsErr = err
		return ss
	}
	ss.APIVersion = client.cfg.NbuServer.APIVersion

	var sm []StorageMetricValue
	var su []StorageUnitInfo
	var agg *JobAggregator
	var sErr, jErr error
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error { sm, su, sErr = FetchStorageFull(gctx, client); return nil })
	g.Go(func() error { agg, jErr = FetchAllJobsFull(gctx, client, tc.jobWindow()); return nil })
	_ = g.Wait()

	now := time.Now()
	ss.StorageMetrics, ss.StorageUnits, ss.JobAgg = sm, su, agg
	ss.StorageErr, ss.JobsErr = sErr, jErr
	if sErr == nil {
		ss.LastStorageScrape = now
	}
	if jErr == nil {
		ss.LastJobsScrape = now
	}
	ss.Up = sErr == nil || jErr == nil

	// Run enabled opt-in sub-collectors for this site and buffer their metrics
	// into the snapshot, to be re-emitted on each scrape (graceful degradation:
	// a failing sub-collector is logged and skipped, never aborting the cycle).
	subs := buildSubCollectorsFor(client, tc.cfg, tc.site)
	ss.SubMetrics = collectSubMetrics(ctx, subs, tc.tracing)

	return ss
}

// CollectionLoop polls every target on interval and publishes an immutable Snapshot.
type CollectionLoop struct {
	targets  []*TargetCollector
	store    *SnapshotStore
	interval time.Duration
}

// NewCollectionLoop creates a CollectionLoop that collects from all targets on the given
// interval and publishes each complete sweep as an immutable Snapshot into store.
func NewCollectionLoop(targets []*TargetCollector, store *SnapshotStore, interval time.Duration) *CollectionLoop {
	return &CollectionLoop{targets: targets, store: store, interval: interval}
}

func (l *CollectionLoop) collectOnce(ctx context.Context) {
	sites := make(map[string]*SiteSnapshot, len(l.targets))
	var mu sync.Mutex
	g, gctx := errgroup.WithContext(ctx)
	limit := runtime.NumCPU()
	if len(l.targets) < limit {
		limit = len(l.targets)
	}
	if limit > 0 {
		g.SetLimit(limit)
	}
	for _, tc := range l.targets {
		tc := tc
		g.Go(func() error {
			ss := tc.collect(gctx)
			mu.Lock()
			sites[tc.site] = ss
			mu.Unlock()
			return nil
		})
	}
	_ = g.Wait()
	l.store.Store(&Snapshot{Sites: sites})
}

// Close releases every target's client connections. Call it only after Run has
// returned (the loop goroutine has stopped), so no collection is in flight.
func (l *CollectionLoop) Close() error {
	var firstErr error
	for _, tc := range l.targets {
		if err := tc.close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// Run collects immediately, then every interval until ctx is cancelled.
func (l *CollectionLoop) Run(ctx context.Context) {
	// Start the ticker before the first sweep so its cadence is anchored at t0:
	// the next sweep fires at `interval`, not `first_sweep_duration + interval`,
	// avoiding a startup coverage gap.
	t := time.NewTicker(l.interval)
	defer t.Stop()
	l.collectOnce(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			l.collectOnce(ctx)
		}
	}
}
