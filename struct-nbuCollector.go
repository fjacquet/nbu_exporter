package main

import "github.com/prometheus/client_golang/prometheus"

//Define a struct for you collector that contains pointers
//to prometheus descriptors for each metric you wish to expose.
//Note you can also include fields of other types if they provide utility
//but we just won't be exposing them as metrics.
type nbuCollector struct {
	nbuDiskSize        *prometheus.Desc
	nbuResponseTime    *prometheus.Desc
	nbuJobsSize        *prometheus.Desc
	nbuJobsCount       *prometheus.Desc
	nbuJobsStatusCount *prometheus.Desc
}
