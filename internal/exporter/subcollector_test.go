package exporter

import (
	"context"
	"errors"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

type fakeSub struct {
	name string
	err  error
	ran  bool
}

func (f *fakeSub) Name() string { return f.name }
func (f *fakeSub) Collect(_ context.Context, _ chan<- prometheus.Metric) error {
	f.ran = true
	return f.err
}

func TestRunSubCollectorsGracefulDegradation(t *testing.T) {
	ok := &fakeSub{name: "ok"}
	bad := &fakeSub{name: "bad", err: errors.New("boom")}
	ch := make(chan prometheus.Metric, 8)
	tracing := NewTracerWrapper(nil, "test-collector") // noop tracer
	runSubCollectors(context.Background(), []subCollector{ok, bad}, ch, tracing)
	if !ok.ran || !bad.ran {
		t.Fatalf("all sub-collectors must run despite one failing: ok=%v bad=%v", ok.ran, bad.ran)
	}
}
