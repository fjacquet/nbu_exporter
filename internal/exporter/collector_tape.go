package exporter

import (
	"context"
	"strconv"

	"github.com/fjacquet/nbu_exporter/internal/models"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

const (
	drivesPath         = "/storage/drives"
	tapeMediaPath      = "/storage/tape-media"
	robotHostPath      = "/storage/robots-device-hosts"
	tapeVolumePoolPath = "/storage/tape-volume-pools" // API v12.0+ (absent on v10.0)
	diskPoolsPath      = "/storage/disk-pools"        // API v12.0+ (absent on v10.0)
	maxMediaPages      = 1000                         // safety cap so a backend ignoring page[offset] cannot loop forever
	pageLimitInt       = 100                          // int form of pageLimit ("100"), for full-page truncation checks
)

// tapeCollector is an opt-in sub-collector for tape/drive health. It reads
// /storage/drives, /storage/tape-media, /storage/robots-device-hosts and the
// v12.0+ /storage/tape-volume-pools and /storage/disk-pools endpoints, with
// per-endpoint graceful degradation (a missing endpoint on older NetBackup, or a
// permission error, is logged and skipped — so the pool endpoints simply contribute
// nothing on appliances that lack them).
type tapeCollector struct {
	client NetBackupClient
	cfg    models.Config
	site   string

	drivesCount     *prometheus.Desc
	driveInfo       *prometheus.Desc
	mediaCount      *prometheus.Desc
	robotHostsCount *prometheus.Desc
	poolPartial     *prometheus.Desc
	diskPoolVolumes *prometheus.Desc
}

func newTapeCollector(client NetBackupClient, cfg models.Config, site string) *tapeCollector {
	return &tapeCollector{
		client: client,
		cfg:    cfg,
		site:   site,
		drivesCount: prometheus.NewDesc(
			"nbu_tape_drives_count",
			"Number of tape drives grouped by drive type, robot type and raw drive status",
			[]string{"site", "drive_type", "robot_type", "status"}, nil,
		),
		driveInfo: prometheus.NewDesc(
			"nbu_tape_drive_info",
			"Tape drive info (always 1; metadata in labels)",
			[]string{"site", "drive_name", "media_server", "drive_type", "robot_number", "status"}, nil,
		),
		mediaCount: prometheus.NewDesc(
			"nbu_tape_media_count",
			"Number of tape volumes grouped by volume pool, media type and robot type",
			[]string{"site", "pool", "media_type", "robot_type"}, nil,
		),
		robotHostsCount: prometheus.NewDesc(
			"nbu_tape_robot_device_hosts",
			"Number of device hosts that have robots configured",
			[]string{"site"}, nil,
		),
		poolPartial: prometheus.NewDesc(
			"nbu_tape_pool_partially_full",
			"Number of partially full tape media volumes in each volume pool",
			[]string{"site", "pool_name", "pool_type"}, nil,
		),
		diskPoolVolumes: prometheus.NewDesc(
			"nbu_disk_pool_volume_count",
			"Number of disk volumes per disk pool, grouped by pool name, storage category and volume state",
			[]string{"site", "pool_name", "storage_category", "state"}, nil,
		),
	}
}

func (c *tapeCollector) Name() string { return "tape" }

func (c *tapeCollector) Collect(ctx context.Context, ch chan<- prometheus.Metric) error {
	c.collectDrives(ctx, ch)
	c.collectMedia(ctx, ch)
	c.collectRobotHosts(ctx, ch)
	c.collectPools(ctx, ch)
	c.collectDiskPools(ctx, ch)
	return nil
}

func (c *tapeCollector) collectRobotHosts(ctx context.Context, ch chan<- prometheus.Metric) {
	url := c.cfg.BuildURL(robotHostPath, map[string]string{QueryParamLimit: pageLimit})
	var resp models.RobotDeviceHosts
	if err := c.client.FetchData(ctx, url, &resp); err != nil {
		log.WithError(err).WithField("site", c.site).Warn("tape: robots-device-hosts fetch failed; skipping")
		return
	}
	// This endpoint returns a flat list (no pagination). If it fills a full page the
	// count may be truncated — surface it rather than silently under-reporting.
	if len(resp.Data) >= pageLimitInt {
		log.WithField("site", c.site).Warnf("tape: robots-device-hosts returned a full page (%s); count may be truncated", pageLimit)
	}
	ch <- prometheus.MustNewConstMetric(c.robotHostsCount, prometheus.GaugeValue, float64(len(resp.Data)), c.site)
}

func (c *tapeCollector) collectMedia(ctx context.Context, ch chan<- prometheus.Metric) {
	type key struct{ pool, mediaType, robotType string }
	counts := map[key]float64{}
	offset := 0
	for page := 0; page < maxMediaPages; page++ {
		url := c.cfg.BuildURL(tapeMediaPath, map[string]string{
			QueryParamLimit:  pageLimit,
			QueryParamOffset: strconv.Itoa(offset),
		})
		var resp models.TapeMedia
		if err := c.client.FetchData(ctx, url, &resp); err != nil {
			// Break (not return) so counts already aggregated from earlier pages are
			// still emitted — degraded-but-partial rather than all-or-nothing.
			log.WithError(err).WithField("site", c.site).Warn("tape: tape-media fetch failed; emitting partial counts")
			break
		}
		if len(resp.Data) == 0 { // empty page: no more rows
			break
		}
		for _, d := range resp.Data {
			counts[key{d.Attributes.VolumePool, d.Attributes.MediaType, d.Attributes.RobotType}]++
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
		ch <- prometheus.MustNewConstMetric(c.mediaCount, prometheus.GaugeValue, v, c.site, k.pool, k.mediaType, k.robotType)
	}
}

func (c *tapeCollector) collectDrives(ctx context.Context, ch chan<- prometheus.Metric) {
	url := c.cfg.BuildURL(drivesPath, map[string]string{QueryParamLimit: pageLimit, QueryParamOffset: "0"})
	var resp models.TapeDrives
	if err := c.client.FetchData(ctx, url, &resp); err != nil {
		log.WithError(err).WithField("site", c.site).Warn("tape: drives fetch failed; skipping")
		return
	}
	type key struct{ driveType, robotType, status string }
	counts := map[key]float64{}
	for _, d := range resp.Data {
		a := d.Attributes
		counts[key{a.DriveType, a.RobotType, a.DriveStatus}]++
		ch <- prometheus.MustNewConstMetric(c.driveInfo, prometheus.GaugeValue, 1,
			c.site, a.DriveName, a.DeviceHost, a.DriveType, strconv.Itoa(a.RobotNumber), a.DriveStatus)
	}
	for k, v := range counts {
		ch <- prometheus.MustNewConstMetric(c.drivesCount, prometheus.GaugeValue, v,
			c.site, k.driveType, k.robotType, k.status)
	}
}

// collectPools paginates GET /storage/tape-volume-pools (API v12.0+) and emits
// nbu_tape_pool_partially_full per (pool_name, pool_type). Absent on older
// appliances; the fetch error is logged and skipped (graceful degradation).
func (c *tapeCollector) collectPools(ctx context.Context, ch chan<- prometheus.Metric) {
	offset := 0
	for page := 0; page < maxMediaPages; page++ {
		url := c.cfg.BuildURL(tapeVolumePoolPath, map[string]string{
			QueryParamLimit:  pageLimit,
			QueryParamOffset: strconv.Itoa(offset),
		})
		var resp models.TapeVolumePools
		if err := c.client.FetchData(ctx, url, &resp); err != nil {
			log.WithError(err).WithField("site", c.site).Warn("tape: tape-volume-pools fetch failed; skipping")
			return
		}
		for _, p := range resp.Data {
			a := p.Attributes
			ch <- prometheus.MustNewConstMetric(c.poolPartial, prometheus.GaugeValue,
				float64(a.PartiallyFullMedia), c.site, a.VolumePoolName, a.PoolType)
		}
		if resp.Meta.Pagination.Next == 0 || len(resp.Data) == 0 {
			break
		}
		offset = resp.Meta.Pagination.Next
		if ctx.Err() != nil {
			break
		}
	}
}

// collectDiskPools paginates GET /storage/disk-pools (API v12.0+) and emits
// nbu_disk_pool_volume_count per (pool_name, storage_category, volume state).
// Absent on older appliances; the fetch error is logged and skipped.
func (c *tapeCollector) collectDiskPools(ctx context.Context, ch chan<- prometheus.Metric) {
	type key struct{ poolName, storageCategory, state string }
	counts := map[key]float64{}
	offset := 0
	for page := 0; page < maxMediaPages; page++ {
		url := c.cfg.BuildURL(diskPoolsPath, map[string]string{
			QueryParamLimit:  pageLimit,
			QueryParamOffset: strconv.Itoa(offset),
		})
		var resp models.DiskPools
		if err := c.client.FetchData(ctx, url, &resp); err != nil {
			log.WithError(err).WithField("site", c.site).Warn("tape: disk-pools fetch failed; skipping")
			return
		}
		for _, pool := range resp.Data {
			a := pool.Attributes
			for _, vol := range a.DiskVolumes {
				counts[key{a.Name, a.StorageCategory, vol.State}]++
			}
		}
		if resp.Meta.Pagination.Next == 0 || len(resp.Data) == 0 {
			break
		}
		offset = resp.Meta.Pagination.Next
		if ctx.Err() != nil {
			break
		}
	}
	for k, v := range counts {
		ch <- prometheus.MustNewConstMetric(c.diskPoolVolumes, prometheus.GaugeValue, v,
			c.site, k.poolName, k.storageCategory, k.state)
	}
}
