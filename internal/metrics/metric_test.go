package metrics_test

import (
	"encoding/json"
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

func TestMetric_IsEmpty(t *testing.T) {
	testCases := []struct {
		name   string
		metric metrics.Metric
		want   bool
	}{
		{
			name:   "empty",
			metric: metrics.Metric{},
			want:   true,
		},
		{
			name:   "counter not empty",
			metric: metrics.Counter("", 0),
			want:   false,
		},
		{
			name:   "gauge not empty",
			metric: metrics.Gauge("", 0),
			want:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.metric.IsEmpty()
			require.Equal(t, tc.want, got)
		})
	}
}

func TestMetric_Marshal(t *testing.T) {
	testCases := []struct {
		name      string
		metric    metrics.Metric
		wantData  string
		wantError bool
	}{
		{
			name:     "empty",
			metric:   metrics.Metric{},
			wantData: "{}",
		},
		{
			name:     "counter",
			metric:   metrics.Counter("test", 1),
			wantData: `{"type":"counter","id":"test","delta":1}`,
		},
		{
			name:     "gauge",
			metric:   metrics.Gauge("test", 0.00005),
			wantData: `{"type":"gauge","id":"test","value":0.00005}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(&tc.metric)

			if tc.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.wantData, string(data))
			}
		})
	}
}

func TestMetric_Unmarshal(t *testing.T) {
	testCases := []struct {
		name       string
		data       []byte
		wantMetric metrics.Metric
		wantError  bool
	}{
		{
			name: "empty",
			data: []byte("{}"),
		},
		{
			name:       "counter1",
			data:       []byte(`{"type":"counter","id":"test","delta":1}`),
			wantMetric: metrics.Counter("test", 1),
		},
		{
			name:       "counter2",
			data:       []byte(`{"type":"counter","id":"test"}`),
			wantMetric: metrics.Counter("test", 0),
		},
		{
			name:       "gauge1",
			data:       []byte(`{"type":"gauge","id":"test","value":0.00005}`),
			wantMetric: metrics.Gauge("test", 0.00005),
		},
		{
			name:       "gauge2",
			data:       []byte(`{"type":"gauge","id":"test"}`),
			wantMetric: metrics.Gauge("test", 0),
		},
		{
			name:      "type is blank",
			data:      []byte(`{"type":""}`),
			wantError: true,
		},
		{
			name:      "id is blank",
			data:      []byte(`{"type":"counter","id":""}`),
			wantError: true,
		},
		{
			name:      "type is unknown",
			data:      []byte(`{"type":"unknown","id":"test","delta":1}`),
			wantError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var got metrics.Metric
			err := json.Unmarshal(tc.data, &got)

			if tc.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.True(t, tc.wantMetric.Equal(got))
			}
		})
	}
}
