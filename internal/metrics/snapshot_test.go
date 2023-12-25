package metrics_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sergeizaitcev/metrics/internal/metrics"
)

func TestSnapshot(t *testing.T) {
	snapshot := metrics.Snapshot()
	require.NotEmpty(t, snapshot)

	for _, metric := range snapshot {
		switch metric.Name() {
		case "PollCount":
			require.NotPanics(t, func() { metric.Int64() })
			require.EqualValues(t, 1, metric.Int64())
		default:
			require.NotEmpty(t, metric.Kind())
			require.NotEmpty(t, metric.Name())
		}
	}
}

func BenchmarkSnapshot(b *testing.B) {
	var snapshot []metrics.Metric
	for i := 0; i < b.N; i++ {
		snapshot = metrics.Snapshot()
	}
	_ = snapshot
}
