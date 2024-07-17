package main

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestNewNbuCollector(t *testing.T) {
	collector := newNbuCollector()

	if collector == nil {
		t.Error("newNbuCollector returned nil")
	}

	if collector.nbuResponseTime == nil {
		t.Error("nbuResponseTime metric not initialized")
	}

	if collector.nbuDiskSize == nil {
		t.Error("nbuDiskSize metric not initialized")
	}

	if collector.nbuJobsSize == nil {
		t.Error("nbuJobsSize metric not initialized")
	}

	if collector.nbuJobsCount == nil {
		t.Error("nbuJobsCount metric not initialized")
	}

	if collector.nbuJobsStatusCount == nil {
		t.Error("nbuJobsStatusCount metric not initialized")
	}


}


// func TestNbuCollectorDescribe(t *testing.T) {
// 	collector := newNbuCollector()
// 	ch := make(chan *prometheus.Desc)
// 	defer close(ch)

// 	go collector.Describe(ch)

// 	expectedDescs := []*prometheus.Desc{
// 		collector.nbuDiskSize,
// 		collector.nbuResponseTime,
// 		collector.nbuJobsSize,
// 		collector.nbuJobsCount,
// 		collector.nbuJobsStatusCount,
// 	}

// 	for _, expected := range expectedDescs {
// 		select {
// 		case desc := <-ch:
// 			if desc != expected {
// 				t.Errorf("Unexpected desc: %v, expected: %v", desc, expected)
// 			}
// 		default:
// 			t.Errorf("Missing expected desc: %v", expected)
// 		}
// 	}

// 	select {
// 	case unexpected := <-ch:
// 		t.Errorf("Unexpected desc received: %v", unexpected)
// 	default:
// 	}
// }

// TestNewNbuCollectorMetricDescriptions tests that the metric descriptions for the
// nbu collector are as expected.
// func TestNewNbuCollectorMetricDescriptions(t *testing.T) {
// 	collector := newNbuCollector()

// 	expectedResponseTimeDesc := prometheus.NewDesc(
// 		"nbu_response_time_ms",
// 		"The server response time in millisecond",
// 		nil, nil)
// 	if collector.nbuResponseTime != expectedResponseTimeDesc {
// 		t.Errorf("Unexpected nbuResponseTime description: %v", collector.nbuResponseTime)
// 	}

// 	expectedDiskSizeDesc := prometheus.NewDesc(
// 		"nbu_disk_bytes",
// 		"The quantity of storage bytes",
// 		[]string{"name", "type", "size"}, nil)
// 	if collector.nbuDiskSize != expectedDiskSizeDesc {
// 		t.Errorf("Unexpected nbuDiskSize description: %v", collector.nbuDiskSize)
// 	}

// 	expectedJobsSizeDesc := prometheus.NewDesc(
// 		"nbu_jobs_bytes",
// 		"The quantity of processed bytes",
// 		[]string{"action", "policy_type", "status"}, nil)
// 	if collector.nbuJobsSize != expectedJobsSizeDesc {
// 		t.Errorf("Unexpected nbuJobsSize description: %v", collector.nbuJobsSize)
// 	}

// 	expectedJobsCountDesc := prometheus.NewDesc(
// 		"nbu_jobs_count",
// 		"The quantity of jobs",
// 		[]string{"action", "policy_type", "status"}, nil)
// 	if collector.nbuJobsCount != expectedJobsCountDesc {
// 		t.Errorf("Unexpected nbuJobsCount description: %v", collector.nbuJobsCount)
// 	}

// 	expectedJobsStatusCountDesc := prometheus.NewDesc(
// 		"nbu_status_count",
// 		"The quantity per status",
// 		[]string{"action", "status"}, nil)
// 	if collector.nbuJobsStatusCount != expectedJobsStatusCountDesc {
// 		t.Errorf("Unexpected nbuJobsStatusCount description: %v", collector.nbuJobsStatusCount)
// 	}
// }

// func TestNbuCollectorDescribe(t *testing.T) {
// 	collector := newNbuCollector()
// 	ch := make(chan *prometheus.Desc)
// 	defer close(ch)

// 	go collector.Describe(ch)

// 	expectedDescs := []*prometheus.Desc{
// 		collector.nbuDiskSize,
// 		collector.nbuResponseTime,
// 		collector.nbuJobsSize,
// 		collector.nbuJobsCount,
// 		collector.nbuJobsStatusCount,
// 	}

// 	for _, expected := range expectedDescs {
// 		select {
// 		case desc := <-ch:
// 			if desc != expected {
// 				t.Errorf("Unexpected desc: %v, expected: %v", desc, expected)
// 			}
// 		default:
// 			t.Errorf("Missing expected desc: %v", expected)
// 		}
// 	}

// 	select {
// 	case unexpected := <-ch:
// 		t.Errorf("Unexpected desc received: %v", unexpected)
// 	default:
// 	}
// }

// TestNbuCollectorDescribeWithClosedChannel tests the behavior of the Describe method of the nbuCollector
// when the provided channel is closed. It verifies that no values are sent to the closed channel.
// func TestNbuCollectorDescribeWithClosedChannel(t *testing.T) {
// 	collector := newNbuCollector()
// 	ch := make(chan *prometheus.Desc)
// 	close(ch)

// 	collector.Describe(ch)

// 	select {
// 	case <-ch:
// 		t.Error("Unexpected value received from closed channel")
// 	default:
// 	}
// }

// TestNbuCollectorDescribeWithNilChannel tests the Describe method of the NbuCollector
// when the provided channel is nil. This ensures the method handles a nil channel
// gracefully without panicking.
// func TestNbuCollectorDescribeWithNilChannel(t *testing.T) {
// 	collector := newNbuCollector()
// 	var ch chan *prometheus.Desc

// 	collector.Describe(ch)
// }


// TestNbuCollectorCollect tests the Collect method of the nbuCollector.
// It creates an nbuCollector, sends it to the Collect method, and verifies
// that the expected metrics are received on the provided channel.
// func TestNbuCollectorCollect(t *testing.T) {
// 	collector := newNbuCollector()
// 	ch := make(chan prometheus.Metric)
// 	defer close(ch)

// 	go collector.Collect(ch)

// 	expectedMetrics := []prometheus.Metric{
// 		prometheus.MustNewConstMetric(collector.nbuDiskSize, prometheus.GaugeValue, 1024, "disk1", "type1", "size1"),
// 		prometheus.MustNewConstMetric(collector.nbuDiskSize, prometheus.GaugeValue, 2048, "disk2", "type2", "size2"),
// 		prometheus.MustNewConstMetric(collector.nbuJobsSize, prometheus.GaugeValue, 512, "backup", "policy1", "success"),
// 		prometheus.MustNewConstMetric(collector.nbuJobsSize, prometheus.GaugeValue, 1024, "restore", "policy2", "failed"),
// 		prometheus.MustNewConstMetric(collector.nbuJobsCount, prometheus.GaugeValue, 10, "backup", "policy1", "success"),
// 		prometheus.MustNewConstMetric(collector.nbuJobsCount, prometheus.GaugeValue, 5, "restore", "policy2", "failed"),
// 		prometheus.MustNewConstMetric(collector.nbuJobsStatusCount, prometheus.GaugeValue, 15, "backup", "success"),
// 		prometheus.MustNewConstMetric(collector.nbuJobsStatusCount, prometheus.GaugeValue, 5, "restore", "failed"),
// 	}

// 	for _, expected := range expectedMetrics {
// 		select {
// 		case metric := <-ch:
// 			if !metric.Desc().Equal(expected.Desc()) || metric.GetValue() != expected.GetValue() {
// 				t.Errorf("Unexpected metric: %v, expected: %v", metric, expected)
// 			}
// 		default:
// 			t.Errorf("Missing expected metric: %v", expected)
// 		}
// 	}

// 	select {
// 	case unexpected := <-ch:
// 		t.Errorf("Unexpected metric received: %v", unexpected)
// 	default:
// 	}
// }

// TestNbuCollectorCollectWithEmptyMaps tests the Collect method of the nbuCollector
// when the maps used to generate metrics are empty. This ensures the method
// handles empty maps gracefully without emitting any metrics.
func TestNbuCollectorCollectWithEmptyMaps(t *testing.T) {
	collector := newNbuCollector()
	ch := make(chan prometheus.Metric)
	defer close(ch)

	go collector.Collect(ch)

	select {
	case metric := <-ch:
		t.Errorf("Unexpected metric received: %v", metric)
	default:
	}
}
