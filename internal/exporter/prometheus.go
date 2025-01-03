package exporter

import (
	"strings"

	"github.com/fjacquet/nbu_exporter/internal/models"
	"github.com/prometheus/client_golang/prometheus"
)

// Define a struct for you collector that contains pointers
// to prometheus descriptors for each metric you wish to expose.
// Note you can also include fields of other types if they provide utility
// but we just won't be exposing them as metrics.
type NbuCollector struct {
	cfg                models.Config
	nbuDiskSize        *prometheus.Desc
	nbuResponseTime    *prometheus.Desc
	nbuJobsSize        *prometheus.Desc
	nbuJobsCount       *prometheus.Desc
	nbuJobsStatusCount *prometheus.Desc
}

// NewNbuCollector You must create a constructor for you collector that
// initializes every descriptor and returns a pointer to the collector
func NewNbuCollector(cfg models.Config) *NbuCollector {

	return &NbuCollector{
		cfg: cfg, // Injected configuration
		nbuResponseTime: prometheus.NewDesc(
			"nbu_response_time_ms",
			"The server response time in millisecond",
			nil, nil),
		nbuDiskSize: prometheus.NewDesc(
			"nbu_disk_bytes",
			"The quantity of storage bytes",
			[]string{"name", "type", "size"}, nil),
		nbuJobsSize: prometheus.NewDesc(
			"nbu_jobs_bytes",
			"The quantity of processed bytes",
			[]string{"action", "policy_type", "status"}, nil),
		nbuJobsCount: prometheus.NewDesc(
			"nbu_jobs_count",
			"The quantity of jobs",
			[]string{"action", "policy_type", "status"}, nil),
		nbuJobsStatusCount: prometheus.NewDesc(
			"nbu_status_count",
			"The quantity per status",
			[]string{"action", "status"}, nil),
	}
}

//	Describe Each and every collector must implement the Describe function.
//
// It essentially writes all descriptors to the prometheus desc channel.
func (collector *NbuCollector) Describe(ch chan<- *prometheus.Desc) {

	//Update this section with the each metric you create for a given collector
	ch <- collector.nbuDiskSize
	ch <- collector.nbuResponseTime
	ch <- collector.nbuJobsSize
	ch <- collector.nbuJobsCount
	ch <- collector.nbuJobsStatusCount

}

// Collect implements required collect function for all promehteus collectors
func (collector *NbuCollector) Collect(ch chan<- prometheus.Metric) {

	//Implement logic here to determine proper metric value to return to prometheus
	//for each descriptor or call other functions that do so.

	var disks = make(map[string]float64)
	fetchStorage(disks, collector.cfg)
	var jobsSize = make(map[string]float64)
	var jobsCount = make(map[string]float64)
	var jobsStatusCount = make(map[string]float64)
	fetchAllJobs(jobsSize, jobsCount, jobsStatusCount, collector.cfg)

	//Write latest value for each metric in the prometheus metric channel.
	//Note that you can pass CounterValue, GaugeValue, or UntypedValue types here
	for key, value := range disks {
		labels := strings.Split(key, "|")
		ch <- prometheus.MustNewConstMetric(collector.nbuDiskSize, prometheus.GaugeValue, value, labels[0], labels[1], labels[2])
	}

	for key, value := range jobsSize {
		labels := strings.Split(key, "|")
		ch <- prometheus.MustNewConstMetric(collector.nbuJobsSize, prometheus.GaugeValue, value, labels[0], labels[1], labels[2])
	}

	for key, value := range jobsCount {
		labels := strings.Split(key, "|")
		ch <- prometheus.MustNewConstMetric(collector.nbuJobsCount, prometheus.GaugeValue, value, labels[0], labels[1], labels[2])
	}

	for key, value := range jobsStatusCount {
		labels := strings.Split(key, "|")
		ch <- prometheus.MustNewConstMetric(collector.nbuJobsStatusCount, prometheus.GaugeValue, value, labels[0], labels[1])
	}

}
