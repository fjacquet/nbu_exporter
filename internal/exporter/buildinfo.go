package exporter

import "github.com/prometheus/client_golang/prometheus"

// NewBuildInfoCollector returns a collector exposing `nbu_exporter_build_info{version,goversion} 1`,
// so a scrape reveals exactly which exporter build is running. Standard Prometheus build-info pattern.
func NewBuildInfoCollector(version, goversion string) prometheus.Collector {
	g := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace:   "nbu_exporter",
		Name:        "build_info",
		Help:        "Exporter build information; constant 1, with the running version and Go version in the `version` and `goversion` labels.",
		ConstLabels: prometheus.Labels{"version": version, "goversion": goversion},
	})
	g.Set(1)
	return g
}
