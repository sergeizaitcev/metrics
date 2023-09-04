package metrics_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sergeizaitcev/metrics/internal/metrics"
)

func TestMetrics(t *testing.T) {
	t.Run("unknown", func(t *testing.T) {
		var metric metrics.Metric

		require.Empty(t, metric.Name())
		require.Empty(t, metric.Kind())

		require.Panics(t, func() { metric.Int64() })
		require.Panics(t, func() { metric.Float64() })
	})

	t.Run("counter", func(t *testing.T) {
		counter := metrics.Counter("test", 1)

		require.Equal(t, "Counter", counter.Kind().String())
		require.Equal(t, "test", counter.Name())
		require.Equal(t, "1", counter.String())

		require.NotPanics(t, func() { counter.Int64() })
		require.Panics(t, func() { counter.Float64() })
		require.EqualValues(t, 1, counter.Int64())
	})

	t.Run("gauge", func(t *testing.T) {
		gauge := metrics.Gauge("test", 1)

		require.Equal(t, "Gauge", gauge.Kind().String())
		require.Equal(t, "test", gauge.Name())
		require.Equal(t, "1", gauge.String())

		require.NotPanics(t, func() { gauge.Float64() })
		require.Panics(t, func() { gauge.Int64() })
		require.EqualValues(t, 1, gauge.Float64())
	})
}
