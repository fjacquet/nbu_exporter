package exporter

import (
	"context"
	"strconv"
	"strings"

	"github.com/fjacquet/nbu_exporter/internal/models"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

const (
	drivesPath    = "/storage/drives"
	tapeMediaPath = "/storage/tape-media"
	robotHostPath = "/storage/robots-device-hosts"
	maxMediaPages = 1000 // safety cap so a backend ignoring page[offset] cannot loop forever
)

// tapeCollector is an opt-in sub-collector for tape/drive health. It reads
// /storage/drives, /storage/tape-media and /storage/robots-device-hosts, with
// per-endpoint graceful degradation (a missing endpoint on older NetBackup, or a
// permission error, is logged and skipped).
type tapeCollector struct {
	client NetBackupClient
	cfg    models.Config
	site   string

	drivesCount     *prometheus.Desc
	driveInfo       *prometheus.Desc
	mediaCount      *prometheus.Desc
	robotHostsCount *prometheus.Desc
}

func newTapeCollector(client NetBackupClient, cfg models.Config, site string) *tapeCollector {
	return &tapeCollector{
		client: client,
		cfg:    cfg,
		site:   site,
		drivesCount: prometheus.NewDesc(
			"nbu_tape_drives_count",
			"Number of tape drives by status, drive type and robot type",
			[]string{"site", "state", "drive_type", "robot_type"}, nil,
		),
		driveInfo: prometheus.NewDesc(
			"nbu_tape_drive_info",
			"Tape drive info (always 1; metadata in labels)",
			[]string{"site", "drive_name", "media_server", "drive_type", "robot_number", "state"}, nil,
		),
		mediaCount: prometheus.NewDesc(
			"nbu_tape_media_count",
			"Number of tape volumes by media type and status",
			[]string{"site", "media_type", "status"}, nil,
		),
		robotHostsCount: prometheus.NewDesc(
			"nbu_tape_robot_device_hosts",
			"Number of device hosts that have robots configured",
			[]string{"site"}, nil,
		),
	}
}

func (c *tapeCollector) Name() string { return "tape" }

// driveState strips the DRIVE_STATUS_ prefix so the label reads UP/DOWN/MIXED/DISABLED.
func driveState(s string) string { return strings.TrimPrefix(s, "DRIVE_STATUS_") }

func (c *tapeCollector) Collect(ctx context.Context, ch chan<- prometheus.Metric) error {
	c.collectDrives(ctx, ch)
	c.collectMedia(ctx, ch)
	c.collectRobotHosts(ctx, ch)
	return nil
}

func (c *tapeCollector) collectRobotHosts(ctx context.Context, ch chan<- prometheus.Metric) {
	url := c.cfg.BuildURL(robotHostPath, map[string]string{QueryParamLimit: pageLimit})
	var resp models.RobotDeviceHosts
	if err := c.client.FetchData(ctx, url, &resp); err != nil {
		log.WithError(err).WithField("site", c.site).Warn("tape: robots-device-hosts fetch failed; skipping")
		return
	}
	ch <- prometheus.MustNewConstMetric(c.robotHostsCount, prometheus.GaugeValue, float64(len(resp.Data)), c.site)
}

func (c *tapeCollector) collectMedia(ctx context.Context, ch chan<- prometheus.Metric) {
	type key struct{ mediaType, status string }
	counts := map[key]float64{}
	offset := 0
	for page := 0; page < maxMediaPages; page++ {
		url := c.cfg.BuildURL(tapeMediaPath, map[string]string{
			QueryParamLimit:  pageLimit,
			QueryParamOffset: strconv.Itoa(offset),
		})
		var resp models.TapeMedia
		if err := c.client.FetchData(ctx, url, &resp); err != nil {
			log.WithError(err).WithField("site", c.site).Warn("tape: tape-media fetch failed; skipping")
			return
		}
		if len(resp.Data) == 0 { // empty page: no more rows
			break
		}
		for _, d := range resp.Data {
			counts[key{d.Attributes.MediaType, d.Attributes.MediaStatus}]++
		}
		offset += len(resp.Data) // advance by rows returned; loop ends on the next empty page
		if ctx.Err() != nil {
			break
		}
		if page == maxMediaPages-1 {
			log.WithField("site", c.site).Warnf("tape: tape-media pagination hit the %d-page cap; counts may be truncated", maxMediaPages)
		}
	}
	for k, v := range counts {
		ch <- prometheus.MustNewConstMetric(c.mediaCount, prometheus.GaugeValue, v, c.site, k.mediaType, k.status)
	}
}

func (c *tapeCollector) collectDrives(ctx context.Context, ch chan<- prometheus.Metric) {
	url := c.cfg.BuildURL(drivesPath, map[string]string{QueryParamLimit: pageLimit, QueryParamOffset: "0"})
	var resp models.TapeDrives
	if err := c.client.FetchData(ctx, url, &resp); err != nil {
		log.WithError(err).WithField("site", c.site).Warn("tape: drives fetch failed; skipping")
		return
	}
	type key struct{ state, driveType, robotType string }
	counts := map[key]float64{}
	for _, d := range resp.Data {
		a := d.Attributes
		state := driveState(a.DriveStatus)
		counts[key{state, a.DriveType, a.RobotType}]++
		ch <- prometheus.MustNewConstMetric(c.driveInfo, prometheus.GaugeValue, 1,
			c.site, a.DriveName, a.DeviceHost, a.DriveType, strconv.Itoa(a.RobotNumber), state)
	}
	for k, v := range counts {
		ch <- prometheus.MustNewConstMetric(c.drivesCount, prometheus.GaugeValue, v,
			c.site, k.state, k.driveType, k.robotType)
	}
}
