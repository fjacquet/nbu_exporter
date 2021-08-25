package main

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

//You must create a constructor for you collector that
//initializes every descriptor and returns a pointer to the collector
func newNbuCollector() *nbuCollector {
	return &nbuCollector{
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

//Each and every collector must implement the Describe function.
//It essentially writes all descriptors to the prometheus desc channel.
func (collector *nbuCollector) Describe(ch chan<- *prometheus.Desc) {

	//Update this section with the each metric you create for a given collector
	ch <- collector.nbuDiskSize
	ch <- collector.nbuResponseTime
	ch <- collector.nbuJobsSize
	ch <- collector.nbuJobsCount
	ch <- collector.nbuJobsStatusCount

}

//Collect implements required collect function for all promehteus collectors
func (collector *nbuCollector) Collect(ch chan<- prometheus.Metric) {

	//Implement logic here to determine proper metric value to return to prometheus
	//for each descriptor or call other functions that do so.

	var disks = make(map[string]float64)
	getStorage(disks)
	var jobsSize = make(map[string]float64)
	var jobsCount = make(map[string]float64)
	var jobsStatusCount = make(map[string]float64)
	getJobs(jobsSize, jobsCount, jobsStatusCount)

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

	// ch <- prometheus.MustNewConstMetric(collector.nbuResponseTime, prometheus.GaugeValue, 1)
	// ch <- prometheus.MustNewConstMetric(collector.nbuJobsSize, prometheus.GaugeValue, 1, "backup", "standard", "0")
	// ch <- prometheus.MustNewConstMetric(collector.nbuJobsCount, prometheus.GaugeValue, 1, "backup", "standard", "0")

}
