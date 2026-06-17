package exporter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fjacquet/nbu_exporter/internal/models"
	dto "github.com/prometheus/client_model/go"
)

// nbuOKHandler returns HTTP 200 with valid empty NBU JSON for all requests.
// It handles both version-detection probes (/admin/jobs?page[limit]=1) and
// the actual jobs + storage fetches that FetchAllJobsFull / FetchStorageFull make.
func nbuOKHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the requested API version from the Accept header so the
	// Content-Type response matches (mirrors writeVersionResponse in
	// version_detection_integration_test.go).
	accept := r.Header.Get("Accept")
	version := "14.0"
	for _, v := range []string{"14.0", "13.0", "12.0", "10.0"} {
		if strings.Contains(accept, v) {
			version = v
			break
		}
	}
	ct := fmt.Sprintf(contentTypeNetBackupJSONFormat, version)

	switch {
	case strings.Contains(r.URL.Path, "/storage/storage-units"):
		// Encode an empty but valid storage response as raw JSON.
		w.Header().Set(contentTypeHeader, ct)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[],"meta":{"pagination":{}},"links":{}}`))
	default:
		// Handles /admin/jobs (version probe + actual job fetch)
		resp := &models.Jobs{}
		resp.Data = []models.JobData{}
		resp.Meta.Pagination.Next = ""
		w.Header().Set(contentTypeHeader, ct)
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}
}

// nbuErrorHandler returns HTTP 500 for every request.
func nbuErrorHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
}

// newNbuServerConfig builds a NbuServerConfig for a plain-http httptest.Server.
func newNbuServerConfig(site, serverURL, apiVersion string) models.NbuServerConfig {
	host, port := splitTestServerURL(serverURL)
	return models.NbuServerConfig{
		Site:               site,
		Host:               host,
		Port:               port,
		Scheme:             "http",
		URI:                "/netbackup",
		APIKey:             testAPIKey,
		APIVersion:         apiVersion,
		ContentType:        contentTypeJSON,
		InsecureSkipVerify: true,
	}
}

// TestCollectionLoop_collectOnce verifies that collectOnce produces a Snapshot
// with Up=true for a healthy server and Up=false for an unhealthy one.
func TestCollectionLoop_collectOnce(t *testing.T) {
	serverA := httptest.NewServer(http.HandlerFunc(nbuOKHandler))
	defer serverA.Close()

	serverB := httptest.NewServer(http.HandlerFunc(nbuErrorHandler))
	defer serverB.Close()

	// Use createTestConfig for the shared Server.* block (host, port, scraping interval).
	base := createTestConfig(serverA.URL, "14.0")

	entryA := newNbuServerConfig("a", serverA.URL, "14.0")
	entryB := newNbuServerConfig("b", serverB.URL, "14.0")

	tcA := NewTargetCollector(base, entryA)
	tcB := NewTargetCollector(base, entryB)

	store := &SnapshotStore{}
	loop := NewCollectionLoop([]*TargetCollector{tcA, tcB}, store, time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	loop.collectOnce(ctx)

	snap := store.Load()
	if snap == nil {
		t.Fatal("store.Load() returned nil after collectOnce")
	}

	siteA, ok := snap.Sites["a"]
	if !ok {
		t.Fatal("Snapshot missing site 'a'")
	}
	if !siteA.Up {
		t.Errorf("site 'a' Up = false, want true (storageErr=%v, jobsErr=%v)", siteA.StorageErr, siteA.JobsErr)
	}

	siteB, ok := snap.Sites["b"]
	if !ok {
		t.Fatal("Snapshot missing site 'b'")
	}
	if siteB.Up {
		t.Errorf("site 'b' Up = true, want false")
	}
}

// TestMaxDurationString verifies the job-window coupling: the larger of the two
// durations wins, with sane fallback when one is unparseable.
func TestMaxDurationString(t *testing.T) {
	cases := []struct{ a, b, want string }{
		{"1h", "5m", "1h"},   // scraping window already larger
		{"1m", "10m", "10m"}, // poll interval larger -> widen window to avoid gaps
		{"5m", "5m", "5m"},   // equal
		{"5m", "", "5m"},     // unparseable b -> a
		{"", "5m", "5m"},     // unparseable a -> b
	}
	for _, c := range cases {
		if got := maxDurationString(c.a, c.b); got != c.want {
			t.Errorf("maxDurationString(%q,%q) = %q, want %q", c.a, c.b, got, c.want)
		}
	}
}

// TestTargetCollector_BuffersSubMetrics verifies that an enabled opt-in
// sub-collector is run per-target and its metrics are buffered into the
// SiteSnapshot, carrying the target's site label. The SLO collector always
// emits exactly one series, so it is the simplest probe.
func TestTargetCollector_BuffersSubMetrics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(nbuOKHandler))
	defer server.Close()

	base := createTestConfig(server.URL, "14.0")
	base.Collectors.SLO.Enabled = true

	entry := newNbuServerConfig("paris", server.URL, "14.0")
	tc := NewTargetCollector(base, entry)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ss := tc.collect(ctx)
	if !ss.Up {
		t.Fatalf("site Up = false, want true (storageErr=%v jobsErr=%v)", ss.StorageErr, ss.JobsErr)
	}
	if len(ss.SubMetrics) != 1 {
		t.Fatalf("SubMetrics len = %d, want 1 (nbu_slo_count)", len(ss.SubMetrics))
	}

	var d dto.Metric
	if err := ss.SubMetrics[0].Write(&d); err != nil {
		t.Fatalf("metric write: %v", err)
	}
	if got := labelValue(&d, "site"); got != "paris" {
		t.Errorf("buffered sub-metric site label = %q, want paris", got)
	}
}
