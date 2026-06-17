package exporter

import (
	"context"
	"fmt"

	"github.com/fjacquet/nbu_exporter/internal/models"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

// perClientCollector is an opt-in sub-collector that emits, per allowlisted client,
// the timestamp of that client's most recent successful backup.
type perClientCollector struct {
	client NetBackupClient
	cfg    models.Config
	site   string
	desc   *prometheus.Desc
}

func newPerClientCollector(client NetBackupClient, cfg models.Config, site string) *perClientCollector {
	return &perClientCollector{
		client: client,
		cfg:    cfg,
		site:   site,
		desc: prometheus.NewDesc(
			"nbu_client_last_successful_backup_timestamp_seconds",
			"Unix timestamp of the client's most recent successful backup",
			[]string{"site", "client"}, nil,
		),
	}
}

func (c *perClientCollector) Name() string { return "perclient" }

// Collect queries each allowlisted client's most recent successful backup. An empty
// allowlist is a natural no-op (no queries, no series) — see the package docs.
func (c *perClientCollector) Collect(ctx context.Context, ch chan<- prometheus.Metric) error {
	for _, name := range c.cfg.Collectors.PerClient.Allowlist {
		c.collectClient(ctx, ch, name)
	}
	return nil
}

// collectClient queries the single most recent successful backup for one client and
// emits its endTime. A fetch error / no result is logged-and-skipped for that client.
func (c *perClientCollector) collectClient(ctx context.Context, ch chan<- prometheus.Metric, name string) {
	// odataQuoteString escapes any single quote in the name (OData '' escaping) so a
	// client such as "O'Brien" produces a well-formed filter rather than a malformed one.
	filter := fmt.Sprintf("clientName eq '%s' and jobType eq 'BACKUP' and status eq 0", odataQuoteString(name))
	url := c.cfg.BuildURL(jobsPath, map[string]string{
		QueryParamFilter: filter,
		QueryParamSort:   "-endTime",
		QueryParamLimit:  "1",
	})
	var resp models.Jobs
	if err := c.client.FetchData(ctx, url, &resp); err != nil {
		log.WithError(err).WithField("site", c.site).WithField("client", name).
			Warn("perClient: jobs query failed; skipping client")
		return
	}
	if len(resp.Data) == 0 {
		return // no successful backup on record for this client
	}
	end := resp.Data[0].Attributes.EndTime
	if end.IsZero() {
		return
	}
	ch <- prometheus.MustNewConstMetric(c.desc, prometheus.GaugeValue, float64(end.Unix()), c.site, name)
}
