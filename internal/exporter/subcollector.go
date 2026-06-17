package exporter

import (
	"context"

	"github.com/fjacquet/nbu_exporter/internal/models"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
)

// buildSubCollectorsFor returns the enabled opt-in collectors for one target,
// each labelled with the given site. Shared by the per-target collection loop
// and the (single-site) live collector.
func buildSubCollectorsFor(client NetBackupClient, cfg models.Config, site string) []subCollector {
	var subs []subCollector
	if cfg.Collectors.Alerts.Enabled {
		subs = append(subs, newAlertsCollector(client, cfg, site))
	}
	if cfg.Collectors.Malware.Enabled {
		subs = append(subs, newMalwareCollector(client, cfg, site))
	}
	if cfg.Collectors.Catalog.Enabled {
		subs = append(subs, newCatalogCollector(client, cfg, site))
	}
	if cfg.Collectors.SLO.Enabled {
		subs = append(subs, newSLOCollector(client, cfg, site))
	}
	if cfg.Collectors.Tape.Enabled {
		subs = append(subs, newTapeCollector(client, cfg, site))
	}
	if cfg.Collectors.PerClient.Enabled {
		subs = append(subs, newPerClientCollector(client, cfg, site))
	}
	return subs
}

// collectSubMetrics runs every sub-collector and returns the metrics they emit
// as a buffered slice, so they can be stored in a snapshot and re-emitted on
// each scrape. Errors are handled inside runSubCollectors (logged, never fatal).
func collectSubMetrics(ctx context.Context, collectors []subCollector, tracing *TracerWrapper) []prometheus.Metric {
	if len(collectors) == 0 {
		return nil
	}
	ch := make(chan prometheus.Metric, 256)
	done := make(chan struct{})
	var out []prometheus.Metric
	go func() {
		for m := range ch {
			out = append(out, m)
		}
		close(done)
	}()
	runSubCollectors(ctx, collectors, ch, tracing)
	close(ch)
	<-done
	return out
}

// subCollector is an optional, opt-in metric source (alerts, malware, catalog, SLO).
// Each runs independently; a failure is logged and skipped so other collectors and
// the core storage/jobs metrics are unaffected (graceful degradation).
type subCollector interface {
	Name() string
	Collect(ctx context.Context, ch chan<- prometheus.Metric) error
}

// runSubCollectors runs every sub-collector concurrently. Errors are logged, never
// propagated, so one failing endpoint cannot suppress the others.
func runSubCollectors(ctx context.Context, collectors []subCollector, ch chan<- prometheus.Metric, tracing *TracerWrapper) {
	if len(collectors) == 0 {
		return
	}
	g, gCtx := errgroup.WithContext(ctx)
	for _, sc := range collectors {
		sc := sc
		g.Go(func() error {
			spanCtx, span := tracing.StartSpan(gCtx, "subcollector."+sc.Name(), trace.SpanKindClient)
			defer span.End()
			if err := sc.Collect(spanCtx, ch); err != nil {
				log.WithField("collector", sc.Name()).WithError(err).Warn("sub-collector failed; skipping")
			}
			return nil
		})
	}
	_ = g.Wait()
}
