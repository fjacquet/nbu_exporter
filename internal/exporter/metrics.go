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
