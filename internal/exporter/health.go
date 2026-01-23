// Package exporter provides health check functionality for the NetBackup exporter.
package exporter

import (
	"context"
	"fmt"
	"time"
)

// healthCheckTimeout is the default timeout for connectivity tests.
const healthCheckTimeout = 5 * time.Second

// TestConnectivity verifies NetBackup API is reachable.
// Uses a lightweight API call (version detection endpoint) with short timeout (5s).
// Returns nil if connectivity test passes, error otherwise.
//
// The method uses DetectAPIVersion as the connectivity test because:
//   - It's a lightweight endpoint (/admin/jobs?page[limit]=1)
//   - It validates both connectivity and authentication
//   - It doesn't modify any server state
//
// Example:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	defer cancel()
//	if err := collector.TestConnectivity(ctx); err != nil {
//	    log.Warnf("NBU connectivity failed: %v", err)
//	}
func (c *NbuCollector) TestConnectivity(ctx context.Context) error {
	// Use provided context or create one with timeout
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, healthCheckTimeout)
		defer cancel()
	}

	// Use DetectAPIVersion as lightweight connectivity test
	_, err := c.client.DetectAPIVersion(ctx)
	if err != nil {
		return fmt.Errorf("NetBackup connectivity test failed: %w", err)
	}
	return nil
}

// IsHealthy returns true if the last collection was successful.
// This is a quick check without making an API call.
// Returns true if at least one metric source (storage or jobs) was collected successfully.
//
// This method is useful for lightweight health checks that don't need to
// verify current connectivity, only whether recent scrapes succeeded.
func (c *NbuCollector) IsHealthy() bool {
	c.scrapeMu.RLock()
	defer c.scrapeMu.RUnlock()
	// Healthy if we've had at least one successful scrape
	return !c.lastStorageScrapeTime.IsZero() || !c.lastJobsScrapeTime.IsZero()
}
