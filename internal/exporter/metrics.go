package exporter

// MetricKey represents a structured key for metrics instead of pipe-delimited strings.
type StorageMetricKey struct {
	Name string
	Type string
	Size string
}

// JobMetricKey represents a structured key for job metrics.
type JobMetricKey struct {
	Action     string
	PolicyType string
	Status     string
}

// JobStatusKey represents a structured key for job status metrics.
type JobStatusKey struct {
	Action string
	Status string
}

// String returns a string representation for map keys.
func (k StorageMetricKey) String() string {
	return k.Name + "|" + k.Type + "|" + k.Size
}

// String returns a string representation for map keys.
func (k JobMetricKey) String() string {
	return k.Action + "|" + k.PolicyType + "|" + k.Status
}

// String returns a string representation for map keys.
func (k JobStatusKey) String() string {
	return k.Action + "|" + k.Status
}

// Labels returns the metric labels as a slice.
func (k StorageMetricKey) Labels() []string {
	return []string{k.Name, k.Type, k.Size}
}

// Labels returns the metric labels as a slice.
func (k JobMetricKey) Labels() []string {
	return []string{k.Action, k.PolicyType, k.Status}
}

// Labels returns the metric labels as a slice.
func (k JobStatusKey) Labels() []string {
	return []string{k.Action, k.Status}
}
