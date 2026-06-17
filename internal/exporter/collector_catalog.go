package exporter

import (
	"context"

	"github.com/fjacquet/nbu_exporter/internal/models"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

const catalogPath = "/catalog/images"

// Curated subsets keep label cardinality bounded (the full cross-product of the
// malwareStatus/anomalyStatus enums is large). Values are valid per
// docs/veritas-11.2/catalog.yaml (/catalog/images filter attributes).
var catalogMalwareStatuses = []string{"CLEAN", "INFECTION_DETECTED_BY_MALWARE_SCAN", "NOT_SCANNED"}
var catalogAnomalyStatuses = []string{"ANOMALOUS", "NOT_ANOMALOUS", "NOT_PROCESSED"}

// catalogCollector is an opt-in sub-collector that emits catalog posture metrics
// from GET /catalog/images using count-only queries (page[limit]=1 + filter).
type catalogCollector struct {
	client NetBackupClient
	cfg    models.Config
	site   string
	desc   *prometheus.Desc
}

func newCatalogCollector(client NetBackupClient, cfg models.Config, site string) *catalogCollector {
	return &catalogCollector{
		client: client,
		cfg:    cfg,
		site:   site,
		desc: prometheus.NewDesc(
			"nbu_catalog_images_count",
			"Number of catalog images by malware and anomaly status",
			[]string{"site", "malware_status", "anomaly_status"}, nil,
		),
	}
}

func (c *catalogCollector) Name() string { return "catalog" }

func (c *catalogCollector) Collect(ctx context.Context, ch chan<- prometheus.Metric) error {
	for _, mw := range catalogMalwareStatuses {
		for _, an := range catalogAnomalyStatuses {
			url := c.cfg.BuildURL(catalogPath, map[string]string{
				QueryParamLimit:  "1",
				QueryParamFilter: "malwareStatus eq " + mw + " and anomalyStatus eq " + an,
			})
			var resp models.CatalogImagesCount
			if err := c.client.FetchData(ctx, url, &resp); err != nil {
				// Per-combination graceful degradation: log and skip this
				// combination so successful combinations are still emitted.
				log.WithError(err).
					WithField("site", c.site).
					WithField("malware_status", mw).
					WithField("anomaly_status", an).
					Warn("catalog count fetch failed; skipping combination")
				continue
			}
			ch <- prometheus.MustNewConstMetric(
				c.desc, prometheus.GaugeValue,
				float64(resp.Meta.Pagination.Count), c.site, mw, an,
			)
		}
	}
	return nil
}
