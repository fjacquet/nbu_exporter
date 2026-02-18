// Package exporter provides caching functionality for NetBackup metrics.
package exporter

import (
	"sync"
	"time"

	"github.com/patrickmn/go-cache"
)

const (
	storageCacheKey = "storage_metrics"
	defaultCacheTTL = 5 * time.Minute
)

// StorageCache provides TTL-based caching for storage metrics.
// It wraps patrickmn/go-cache to cache expensive API calls and reduce
// NetBackup server load.
//
// Storage metrics change infrequently (every few minutes) but scrapes happen
// frequently (every 15-60 seconds). Caching reduces API calls from every scrape
// to once per TTL interval.
//
// Thread-safety: All methods are safe for concurrent use.
type StorageCache struct {
	cache              *cache.Cache
	ttl                time.Duration
	lastCollectionMu   sync.RWMutex
	lastCollectionTime time.Time
}

// NewStorageCache creates a new cache with the specified TTL.
// Cleanup interval is set to 2x TTL.
//
// If ttl <= 0, defaults to 5 minutes.
func NewStorageCache(ttl time.Duration) *StorageCache {
	if ttl <= 0 {
		ttl = defaultCacheTTL
	}
	return &StorageCache{
		cache: cache.New(ttl, ttl*2),
		ttl:   ttl,
	}
}

// Get returns cached storage metrics if available.
// Returns nil, false if cache miss.
func (sc *StorageCache) Get() ([]StorageMetricValue, bool) {
	if cached, found := sc.cache.Get(storageCacheKey); found {
		return cached.([]StorageMetricValue), true
	}
	return nil, false
}

// Set stores storage metrics in the cache with default TTL.
func (sc *StorageCache) Set(metrics []StorageMetricValue) {
	sc.cache.Set(storageCacheKey, metrics, cache.DefaultExpiration)
	sc.lastCollectionMu.Lock()
	sc.lastCollectionTime = time.Now()
	sc.lastCollectionMu.Unlock()
}

// GetLastCollectionTime returns the timestamp of the last successful fetch.
func (sc *StorageCache) GetLastCollectionTime() time.Time {
	sc.lastCollectionMu.RLock()
	defer sc.lastCollectionMu.RUnlock()
	return sc.lastCollectionTime
}

// TTL returns the configured cache TTL.
func (sc *StorageCache) TTL() time.Duration {
	return sc.ttl
}

// Flush clears all cached data.
// Use on config reload when NBU server changes.
func (sc *StorageCache) Flush() {
	sc.cache.Flush()
}
