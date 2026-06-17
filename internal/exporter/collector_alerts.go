package exporter

import (
	"context"

	"github.com/fjacquet/nbu_exporter/internal/models"
	"github.com/prometheus/client_golang/prometheus"
)

const alertsPath = "/manage/alerts"

// alertsCollector is an opt-in sub-collector that emits nbu_alerts_count from
// GET /manage/alerts, grouped by severity and category.
type alertsCollector struct {
	client NetBackupClient
	cfg    models.Config
	site   string
	desc   *prometheus.Desc
}

func newAlertsCollector(client NetBackupClient, cfg models.Config, site string) *alertsCollector {
	return &alertsCollector{
		client: client,
		cfg:    cfg,
		site:   site,
		desc: prometheus.NewDesc(
			"nbu_alerts_count",
			"Number of NetBackup alerts by severity and category",
			[]string{"site", "severity", "category"}, nil,
		),
	}
}

func (a *alertsCollector) Name() string { return "alerts" }

func (a *alertsCollector) Collect(ctx context.Context, ch chan<- prometheus.Metric) error {
	url := a.cfg.BuildURL(alertsPath, map[string]string{
		QueryParamLimit: pageLimit,
	})
	var resp models.Alerts
	if err := a.client.FetchData(ctx, url, &resp); err != nil {
		return err
	}
	type key struct{ severity, category string }
	counts := map[key]float64{}
	for _, d := range resp.Data {
		counts[key{d.Attributes.Severity, d.Attributes.Category}]++
	}
	for k, v := range counts {
		ch <- prometheus.MustNewConstMetric(a.desc, prometheus.GaugeValue, v, a.site, k.severity, k.category)
	}
	return nil
}
