package exporter

import (
	"context"
	"fmt"
	"strings"

	"github.com/fjacquet/nbu_exporter/internal/models"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

// perClientCollector is an opt-in sub-collector that emits, per allowlisted client,
// the timestamp of that client's most recent successful backup.
type perClientCollector struct {
	client    NetBackupClient
	cfg       models.Config
	site      string
	allowlist []string
	desc      *prometheus.Desc
}

func newPerClientCollector(client NetBackupClient, cfg models.Config, site string) *perClientCollector {
	return &perClientCollector{
		client:    client,
		cfg:       cfg,
		site:      site,
		allowlist: cfg.Collectors.PerClient.Allowlist,
		desc: prometheus.NewDesc(
			"nbu_client_last_successful_backup_timestamp_seconds",
			"Unix timestamp of the client's most recent successful backup",
			[]string{"site", "client"}, nil,
		),
	}
}

func (c *perClientCollector) Name() string { return "perclient" }

func (c *perClientCollector) Collect(ctx context.Context, ch chan<- prometheus.Metric) error {
	if len(c.allowlist) == 0 {
		log.WithField("site", c.site).Info("perClient enabled but allowlist empty; no per-client series emitted")
		return nil
	}
	for _, name := range c.allowlist {
		c.collectClient(ctx, ch, name)
	}
	return nil
}

// collectClient queries the single most recent successful backup for one client and
// emits its endTime. A fetch error / no result is logged-and-skipped for that client.
func (c *perClientCollector) collectClient(ctx context.Context, ch chan<- prometheus.Metric, name string) {
	// Single quotes are not percent-encoded by url.Values.Encode (Go's net/url
	// passes them through), so a name containing one would terminate the OData
	// string literal early and yield a malformed filter. Names come from the
	// operator allowlist, so this is defence-in-depth, not injection mitigation.
	if strings.ContainsRune(name, '\'') {
		log.WithField("site", c.site).WithField("client", name).
			Warn("perClient: client name contains a single quote; skipping (would build a malformed OData filter)")
		return
	}
	filter := fmt.Sprintf("clientName eq '%s' and jobType eq 'BACKUP' and status eq 0", name)
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
