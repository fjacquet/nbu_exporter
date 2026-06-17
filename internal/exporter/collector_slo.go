package exporter

import (
	"context"

	"github.com/fjacquet/nbu_exporter/internal/models"
	"github.com/prometheus/client_golang/prometheus"
)

const sloPath = "/servicecatalog/slos"

// sloCollector is an opt-in sub-collector that emits nbu_slo_count from
// GET /servicecatalog/slos.
//
// The 11.2 servicecatalog.yaml SLO schema has no per-SLO enforcement-type
// attribute (the spec's "Enforcement Type" text describes the endpoint's
// access-control model, not response data), so the metric is a single unlabeled
// gauge holding the total number of configured SLOs.
type sloCollector struct {
	client NetBackupClient
	cfg    models.Config
	site   string
	desc   *prometheus.Desc
}

func newSLOCollector(client NetBackupClient, cfg models.Config, site string) *sloCollector {
	return &sloCollector{
		client: client,
		cfg:    cfg,
		site:   site,
		desc: prometheus.NewDesc(
			"nbu_slo_count",
			"Number of configured NetBackup SLOs",
			[]string{"site"}, nil,
		),
	}
}

func (s *sloCollector) Name() string { return "slo" }

func (s *sloCollector) Collect(ctx context.Context, ch chan<- prometheus.Metric) error {
	url := s.cfg.BuildURL(sloPath, map[string]string{QueryParamLimit: pageLimit})
	var resp models.Slos
	if err := s.client.FetchData(ctx, url, &resp); err != nil {
		return err
	}
	ch <- prometheus.MustNewConstMetric(s.desc, prometheus.GaugeValue, float64(len(resp.Data)), s.site)
	return nil
}
