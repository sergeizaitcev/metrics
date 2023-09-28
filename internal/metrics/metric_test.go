package metrics_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sergeizaitcev/metrics/internal/metrics"
)

func TestParseKind(t *testing.T) {
	testCases := []struct {
		kind string
		want metrics.Kind
	}{
		{"Counter", metrics.KindCounter},
		{"counter", metrics.KindCounter},
		{"COUNTER", metrics.KindCounter},
		{"Gauge", metrics.KindGauge},
		{"gauge", metrics.KindGauge},
		{"GAUGE", metrics.KindGauge},
		{"", metrics.KindUnknown},
		{"invalid", metrics.KindUnknown},
		{"unknown", metrics.KindUnknown},
	}

	for _, tc := range testCases {
		t.Run(tc.kind, func(t *testing.T) {
			got := metrics.ParseKind(tc.kind)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestMetric(t *testing.T) {
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

func TestMetric_Equal(t *testing.T) {
	testCases := []struct {
		a, b     metrics.Metric
		wantBool bool
	}{
		{
			a:        metrics.Metric{},
			b:        metrics.Metric{},
			wantBool: true,
		},
		{
			a:        metrics.Counter("counter", 1),
			b:        metrics.Counter("counter", 1),
			wantBool: true,
		},
		{
			a:        metrics.Gauge("gauge", 0.000005),
			b:        metrics.Gauge("gauge", 0.000005),
			wantBool: true,
		},
		{
			a:        metrics.Counter("counter", 1),
			b:        metrics.Counter("counter", 2),
			wantBool: false,
		},
		{
			a:        metrics.Gauge("gauge", 0.001),
			b:        metrics.Gauge("gauge", 0.0011),
			wantBool: false,
		},
		{
			a:        metrics.Counter("counter", 1),
			b:        metrics.Gauge("counter", 1),
			wantBool: false,
		},
		{
			a:        metrics.Counter("counter", 1),
			b:        metrics.Metric{},
			wantBool: false,
		},
		{
			a:        metrics.Gauge("gauge", 1),
			b:        metrics.Metric{},
			wantBool: false,
		},
	}

	for _, tc := range testCases {
		got := tc.a.Equal(tc.b)
		require.Equal(t, tc.wantBool, got)
	}
}
