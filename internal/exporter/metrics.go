// Package exporter provides metric key structures for organizing and aggregating
// NetBackup metrics before exposition to Prometheus.
package exporter

// StorageMetricKey represents a unique identifier for storage unit metrics.
// It combines the storage unit name, server type, and size dimension (free/used)
// to create a composite key for metric aggregation in Prometheus.
//
// Example: StorageMetricKey{Name: "disk-pool-1", Type: "MEDIA_SERVER", Size: "free"}
type StorageMetricKey struct {
	Name string // Storage unit name (e.g., "disk-pool-1")
	Type string // Storage server type (e.g., "MEDIA_SERVER")
	Size string // Size dimension: "free" or "used"
}

// JobMetricKey represents a unique identifier for job-level metrics.
// It combines job action, policy type, and status code to create a composite
// key for aggregating job statistics by type and outcome.
//
// Example: JobMetricKey{Action: "BACKUP", PolicyType: "VMWARE", Status: "0"}
type JobMetricKey struct {
	Action     string // Job action type (e.g., "BACKUP", "RESTORE")
	PolicyType string // Policy type (e.g., "VMWARE", "STANDARD")
	Status     string // Job status code as string (e.g., "0" for success, "150" for failure)
}

// JobStatusKey represents a simplified identifier for job status metrics.
// It combines job action and status code for high-level status aggregation
// across all policy types.
//
// Example: JobStatusKey{Action: "BACKUP", Status: "0"}
type JobStatusKey struct {
	Action string // Job action type (e.g., "BACKUP", "RESTORE")
	Status string // Job status code as string (e.g., "0" for success)
}

// String returns a pipe-delimited string representation suitable for use as a map key.
// Format: "name|type|size"
func (k StorageMetricKey) String() string {
	return k.Name + "|" + k.Type + "|" + k.Size
}

// String returns a pipe-delimited string representation suitable for use as a map key.
// Format: "action|policyType|status"
func (k JobMetricKey) String() string {
	return k.Action + "|" + k.PolicyType + "|" + k.Status
}

// String returns a pipe-delimited string representation suitable for use as a map key.
// Format: "action|status"
func (k JobStatusKey) String() string {
	return k.Action + "|" + k.Status
}

// Labels returns the metric labels as a slice in the order expected by Prometheus descriptors.
// Returns: [name, type, size]
func (k StorageMetricKey) Labels() []string {
	return []string{k.Name, k.Type, k.Size}
}

// Labels returns the metric labels as a slice in the order expected by Prometheus descriptors.
// Returns: [action, policyType, status]
func (k JobMetricKey) Labels() []string {
	return []string{k.Action, k.PolicyType, k.Status}
}

// Labels returns the metric labels as a slice in the order expected by Prometheus descriptors.
// Returns: [action, status]
func (k JobStatusKey) Labels() []string {
	return []string{k.Action, k.Status}
}

// JobStateKey identifies job counts by action and lifecycle state
// (e.g. ACTIVE, QUEUED, DONE) for the nbu_jobs_state_count metric.
type JobStateKey struct {
	Action string // Job action type (e.g. "BACKUP", "RESTORE")
	State  string // Job lifecycle state (e.g. "ACTIVE", "QUEUED", "DONE")
}

// Labels returns the metric labels in descriptor order: [action, state].
func (k JobStateKey) Labels() []string { return []string{k.Action, k.State} }

// JobPolicyKey identifies job aggregates by action and policy type, without a
// status dimension. Used for files-count, dedup-ratio, and duration metrics.
type JobPolicyKey struct {
	Action     string // Job action type (e.g. "BACKUP", "RESTORE")
	PolicyType string // Policy type (e.g. "VMWARE", "STANDARD")
}

// Labels returns the metric labels in descriptor order: [action, policy_type].
func (k JobPolicyKey) Labels() []string { return []string{k.Action, k.PolicyType} }

// JobQueueKey identifies queued-job counts by action and queue reason code
// for the nbu_jobs_queued_count metric.
type JobQueueKey struct {
	Action string // Job action type (e.g. "BACKUP", "RESTORE")
	Reason string // NetBackup job queue reason code as string
}

// Labels returns the metric labels in descriptor order: [action, reason].
func (k JobQueueKey) Labels() []string { return []string{k.Action, k.Reason} }

// JobDurationBuckets are the cumulative upper bounds (in seconds) for the
// nbu_job_duration_seconds histogram. They span one minute to one day, matching
// the range of typical NetBackup backup windows.
var JobDurationBuckets = []float64{60, 300, 900, 1800, 3600, 7200, 14400, 28800, 86400}

// DurationAccum accumulates observations for a const histogram. BucketCounts are
// cumulative (count of observations <= the bucket's upper bound), as required by
// prometheus.MustNewConstHistogram.
type DurationAccum struct {
	BucketCounts []uint64 // cumulative counts aligned to JobDurationBuckets
	Count        uint64   // total number of observations
	Sum          float64  // sum of all observed durations in seconds
}

// newDurationAccum returns a DurationAccum sized to JobDurationBuckets.
func newDurationAccum() *DurationAccum {
	return &DurationAccum{BucketCounts: make([]uint64, len(JobDurationBuckets))}
}

// Observe records a single duration (in seconds) into the histogram.
func (d *DurationAccum) Observe(seconds float64) {
	d.Count++
	d.Sum += seconds
	for i, ub := range JobDurationBuckets {
		if seconds <= ub {
			d.BucketCounts[i]++
		}
	}
}

// Buckets returns the cumulative bucket map keyed by upper bound, ready for
// prometheus.MustNewConstHistogram.
func (d *DurationAccum) Buckets() map[float64]uint64 {
	m := make(map[float64]uint64, len(JobDurationBuckets))
	for i, ub := range JobDurationBuckets {
		m[ub] = d.BucketCounts[i]
	}
	return m
}

// JobAggregator accumulates every job-derived metric across paginated pages in a
// single pass. A pointer is threaded through FetchJobDetails so each page folds
// into the same maps. Keys are comparable structs for direct map lookups.
type JobAggregator struct {
	Size        map[JobMetricKey]float64        // bytes transferred per action/policy/status
	Count       map[JobMetricKey]float64        // job count per action/policy/status
	StatusCount map[JobStatusKey]float64        // job count per action/status
	StateCount  map[JobStateKey]float64         // job count per action/state
	FilesCount  map[JobPolicyKey]float64        // sum of files per action/policy
	DedupSum    map[JobPolicyKey]float64        // sum of dedup ratios per action/policy
	DedupCount  map[JobPolicyKey]float64        // job count contributing to dedup mean
	QueuedCount map[JobQueueKey]float64         // queued job count per action/reason
	Duration    map[JobPolicyKey]*DurationAccum // completed-job duration histogram per action/policy
}

// NewJobAggregator returns a JobAggregator with all maps initialized and capacity
// hints applied to reduce reallocation during aggregation.
func NewJobAggregator() *JobAggregator {
	return &JobAggregator{
		Size:        make(map[JobMetricKey]float64, expectedJobMetricKeys),
		Count:       make(map[JobMetricKey]float64, expectedJobMetricKeys),
		StatusCount: make(map[JobStatusKey]float64, expectedStatusMetricKeys),
		StateCount:  make(map[JobStateKey]float64, expectedStatusMetricKeys),
		FilesCount:  make(map[JobPolicyKey]float64, expectedJobMetricKeys),
		DedupSum:    make(map[JobPolicyKey]float64, expectedJobMetricKeys),
		DedupCount:  make(map[JobPolicyKey]float64, expectedJobMetricKeys),
		QueuedCount: make(map[JobQueueKey]float64, expectedStatusMetricKeys),
		Duration:    make(map[JobPolicyKey]*DurationAccum, expectedJobMetricKeys),
	}
}

// observeDuration folds a completed job's duration into the histogram for its
// action/policy, creating the accumulator on first use.
func (a *JobAggregator) observeDuration(key JobPolicyKey, seconds float64) {
	acc, ok := a.Duration[key]
	if !ok {
		acc = newDurationAccum()
		a.Duration[key] = acc
	}
	acc.Observe(seconds)
}

// StorageUnitInfo holds per-storage-unit attributes that are exposed as their own
// metrics (capacity, concurrency, fragment size) and as a single info metric.
// It is derived from the same storage API response as StorageMetricValue and is
// cached alongside it.
type StorageUnitInfo struct {
	Name               string  // Storage unit name
	Type               string  // Storage server type
	SubType            string  // Storage sub type
	TotalCapacityBytes float64 // Authoritative total capacity (API TotalCapacityBytes)
	MaxConcurrentJobs  float64 // Max concurrent jobs the unit accepts
	MaxFragmentBytes   float64 // Max fragment size in bytes (API value is megabytes)
	IsCloud            bool    // Cloud-backed storage unit
	WormCapable        bool    // Supports WORM (immutability)
	UseWorm            bool    // WORM currently enabled
	ReplicationCapable bool    // Supports replication
	InstantAccess      bool    // Instant Access enabled
}

// boolLabel renders a bool as a stable "true"/"false" label value.
func boolLabel(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// InfoLabels returns the label values for nbu_storage_info in descriptor order:
// [name, type, subtype, is_cloud, worm_capable, use_worm, replication_capable, instant_access].
func (s StorageUnitInfo) InfoLabels() []string {
	return []string{
		s.Name,
		s.Type,
		s.SubType,
		boolLabel(s.IsCloud),
		boolLabel(s.WormCapable),
		boolLabel(s.UseWorm),
		boolLabel(s.ReplicationCapable),
		boolLabel(s.InstantAccess),
	}
}

// StorageMetricValue pairs a StorageMetricKey with its metric value.
// Used for type-safe metric collection without string parsing.
type StorageMetricValue struct {
	Key   StorageMetricKey
	Value float64
}

// JobMetricValue pairs a JobMetricKey with its metric value.
// Used for type-safe metric collection without string parsing.
type JobMetricValue struct {
	Key   JobMetricKey
	Value float64
}

// JobStatusMetricValue pairs a JobStatusKey with its metric value.
// Used for type-safe metric collection without string parsing.
type JobStatusMetricValue struct {
	Key   JobStatusKey
	Value float64
}
