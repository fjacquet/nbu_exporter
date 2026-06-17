package exporter

import (
	"sync/atomic"
	"time"
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
