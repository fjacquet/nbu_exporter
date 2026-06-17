package exporter

import (
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// SiteSnapshot holds one site's already-aggregated collection results.
type SiteSnapshot struct {
	Site              string
	APIVersion        string
	Up                bool
	StorageErr        error
	JobsErr           error
	StorageMetrics    []StorageMetricValue
	StorageUnits      []StorageUnitInfo
	JobAgg            *JobAggregator
	LastStorageScrape time.Time
	LastJobsScrape    time.Time
	// SubMetrics holds the already-built metrics emitted by enabled opt-in
	// sub-collectors (alerts/malware/catalog/SLO) for this site. They are
	// buffered at collection time and re-emitted verbatim on each scrape.
	SubMetrics []prometheus.Metric
}

// Snapshot is an immutable, point-in-time view across all configured sites.
type Snapshot struct {
	Sites map[string]*SiteSnapshot
}

// SnapshotStore holds the latest Snapshot behind an atomic pointer swap, so the
// background collection loop can publish a new snapshot while scrapes read the
// previous one without locking.
type SnapshotStore struct {
	p atomic.Pointer[Snapshot]
}

// Store publishes a new snapshot.
func (s *SnapshotStore) Store(snap *Snapshot) { s.p.Store(snap) }

// Load returns the latest snapshot, or nil if none has been published yet.
func (s *SnapshotStore) Load() *Snapshot { return s.p.Load() }
