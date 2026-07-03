package exporter

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func TestNewBuildInfoCollector(t *testing.T) {
	registry := prometheus.NewRegistry()
	registry.MustRegister(NewBuildInfoCollector("v1.2.3", "go1.99"))

	const expected = `
# HELP nbu_exporter_build_info Exporter build information; constant 1, with the running version and Go version in the ` + "`version`" + ` and ` + "`goversion`" + ` labels.
# TYPE nbu_exporter_build_info gauge
nbu_exporter_build_info{goversion="go1.99",version="v1.2.3"} 1
`

	require.NoError(t, testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"nbu_exporter_build_info",
	))
}
