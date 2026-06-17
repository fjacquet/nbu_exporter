package exporter

import (
	"context"
	"runtime"
	"sync"
	"time"

	"github.com/fjacquet/nbu_exporter/internal/models"
	"golang.org/x/sync/errgroup"
)

// TargetCollector collects one site (one NetBackup primary) into a SiteSnapshot.
type TargetCollector struct {
	site    string
	cfg     models.Config  // a single-server view: cfg.NbuServer set from this site's entry
	client  *NbuClient     // built lazily with version detection; nil until first success
	tracing *TracerWrapper // noop unless a TracerProvider is wired
}

// NewTargetCollector builds a per-site collector. base supplies Server.* settings;
// entry supplies this site's NetBackup server fields.
func NewTargetCollector(base models.Config, entry models.NbuServerConfig) *TargetCollector {
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
	return &TargetCollector{
		site:    entry.Site,
		cfg:     cfg,
		tracing: NewTracerWrapper(nil, "nbu-exporter/target"),
	}
}

func (tc *TargetCollector) clientFor(ctx context.Context) (*NbuClient, error) {
	if tc.client != nil {
		return tc.client, nil
	}
	c, err := NewNbuClientWithVersionDetection(ctx, &tc.cfg)
	if err != nil {
		return nil, err
	}
	tc.client = c
	return c, nil
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
	g.Go(func() error { agg, jErr = FetchAllJobsFull(gctx, client, tc.cfg.Server.ScrapingInterval); return nil })
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

// Run collects immediately, then every interval until ctx is cancelled.
func (l *CollectionLoop) Run(ctx context.Context) {
	l.collectOnce(ctx)
	t := time.NewTicker(l.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			l.collectOnce(ctx)
		}
	}
}
