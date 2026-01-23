package exporter

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewStorageCache_DefaultTTL(t *testing.T) {
	cache := NewStorageCache(0)
	assert.Equal(t, defaultCacheTTL, cache.TTL())
}

func TestNewStorageCache_NegativeTTL(t *testing.T) {
	cache := NewStorageCache(-5 * time.Minute)
	assert.Equal(t, defaultCacheTTL, cache.TTL())
}

func TestNewStorageCache_CustomTTL(t *testing.T) {
	ttl := 2 * time.Minute
	cache := NewStorageCache(ttl)
	assert.Equal(t, ttl, cache.TTL())
}

func TestStorageCache_GetSet(t *testing.T) {
	cache := NewStorageCache(5 * time.Minute)

	// Initially empty
	metrics, found := cache.Get()
	assert.False(t, found)
	assert.Nil(t, metrics)

	// Set metrics
	testMetrics := []StorageMetricValue{
		{Key: StorageMetricKey{Name: "pool1", Type: "AdvancedDisk", Size: "free"}, Value: 1000.0},
		{Key: StorageMetricKey{Name: "pool1", Type: "AdvancedDisk", Size: "used"}, Value: 500.0},
	}
	cache.Set(testMetrics)

	// Get metrics
	retrieved, found := cache.Get()
	assert.True(t, found)
	assert.Equal(t, testMetrics, retrieved)
}

func TestStorageCache_LastCollectionTime(t *testing.T) {
	cache := NewStorageCache(5 * time.Minute)

	// Initially zero
	assert.True(t, cache.GetLastCollectionTime().IsZero())

	// After Set, should be updated
	testMetrics := []StorageMetricValue{
		{Key: StorageMetricKey{Name: "pool1", Type: "AdvancedDisk", Size: "free"}, Value: 1000.0},
	}
	beforeSet := time.Now()
	cache.Set(testMetrics)
	afterSet := time.Now()

	lastCollection := cache.GetLastCollectionTime()
	assert.False(t, lastCollection.IsZero())
	assert.True(t, lastCollection.After(beforeSet) || lastCollection.Equal(beforeSet))
	assert.True(t, lastCollection.Before(afterSet) || lastCollection.Equal(afterSet))
}

func TestStorageCache_Flush(t *testing.T) {
	cache := NewStorageCache(5 * time.Minute)

	// Set metrics
	testMetrics := []StorageMetricValue{
		{Key: StorageMetricKey{Name: "pool1", Type: "AdvancedDisk", Size: "free"}, Value: 1000.0},
	}
	cache.Set(testMetrics)

	// Verify cached
	_, found := cache.Get()
	assert.True(t, found)

	// Flush
	cache.Flush()

	// Verify empty
	metrics, found := cache.Get()
	assert.False(t, found)
	assert.Nil(t, metrics)
}

func TestStorageCache_TTLExpiration(t *testing.T) {
	// Use a very short TTL for testing
	cache := NewStorageCache(50 * time.Millisecond)

	testMetrics := []StorageMetricValue{
		{Key: StorageMetricKey{Name: "pool1", Type: "AdvancedDisk", Size: "free"}, Value: 1000.0},
	}
	cache.Set(testMetrics)

	// Immediately available
	_, found := cache.Get()
	assert.True(t, found)

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should be expired
	_, found = cache.Get()
	assert.False(t, found)
}

func TestStorageCache_EmptyMetrics(t *testing.T) {
	cache := NewStorageCache(5 * time.Minute)

	// Set empty slice
	cache.Set([]StorageMetricValue{})

	// Should still be found (empty is valid)
	metrics, found := cache.Get()
	assert.True(t, found)
	assert.Empty(t, metrics)
}
