package exporter

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
)

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
