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
	})

	t.Run("counter", func(t *testing.T) {
		counter := metrics.Counter("test", 1)

		require.Equal(t, "counter", counter.Kind().String())
		require.Equal(t, "test", counter.Name())
		require.Equal(t, "1", counter.String())

		require.NotPanics(t, func() { counter.Int64() })
		require.EqualValues(t, 1, counter.Int64())
	})

	t.Run("gauge", func(t *testing.T) {
		gauge := metrics.Gauge("test", 1)

		require.Equal(t, "gauge", gauge.Kind().String())
		require.Equal(t, "test", gauge.Name())
		require.Equal(t, "1", gauge.String())

		require.NotPanics(t, func() { gauge.Float64() })
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

func TestMetric_MarshalJSON(t *testing.T) {
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

func TestMetric_UnmarshalJSON(t *testing.T) {
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

func TestMetric_MarshalBinary(t *testing.T) {
	testCases := []struct {
		name string
		want metrics.Metric
	}{
		{
			name: "empty",
			want: metrics.Metric{},
		},
		{
			name: "counter",
			want: metrics.Counter("\n\t\x1btest", 101),
		},
		{
			name: "gauge",
			want: metrics.Gauge("\n\t\x1btest", 1e-5),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := tc.want.MarshalBinary()
			require.NoError(t, err)

			var got metrics.Metric

			err = got.UnmarshalBinary(data)
			require.NoError(t, err)

			require.True(t, tc.want.Equal(got))
		})
	}
}

func marshal(t testing.TB, m metrics.Metric) []byte {
	b, err := m.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestMetric_UnmarshalBinary(t *testing.T) {
	testCases := []struct {
		name       string
		data       []byte
		wantMetric metrics.Metric
		wantError  bool
	}{
		{
			name:      "too small",
			wantError: true,
		},
		{
			name: "corrupt",
			data: func() []byte {
				b := [19]byte{
					9: 0x80, 10: 0x80, 11: 0x80,
					12: 0x80, 13: 0x80, 14: 0x80,
					15: 0x80, 16: 0x80, 17: 0x80,
					18: 0x7f,
				}
				return b[:]
			}(),
			wantError: true,
		},
		{
			name: "base64",
			data: func() []byte {
				b := [11]byte{9: 1, 10: 0x80}
				return b[:]
			}(),
			wantError: true,
		},
		{
			name: "empty",
			data: marshal(t, metrics.Metric{}),
		},
		{
			name:       "counter",
			data:       marshal(t, metrics.Counter("\xb1\b\ttest", 1e3)),
			wantMetric: metrics.Counter("\xb1\b\ttest", 1e3),
		},
		{
			name:       "gauge",
			data:       marshal(t, metrics.Gauge("\xb1\b\t\ftest", 1e-5)),
			wantMetric: metrics.Gauge("\xb1\b\t\ftest", 1e-5),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var got metrics.Metric

			err := got.UnmarshalBinary(tc.data)
			if tc.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.True(t, tc.wantMetric.Equal(got))
			}
		})
	}
}

func BenchmarkParseKind(b *testing.B) {
	b.Run("gauge", func(b *testing.B) {
		const value = "gauge"
		var kind metrics.Kind
		for i := 0; i < b.N; i++ {
			kind = metrics.ParseKind(value)
		}
		_ = kind
	})

	b.Run("counter", func(b *testing.B) {
		const value = "counter"
		var kind metrics.Kind
		for i := 0; i < b.N; i++ {
			kind = metrics.ParseKind(value)
		}
		_ = kind
	})
}

func BenchmarkMetric_MarshalBinary(b *testing.B) {
	var data []byte
	var err error

	counter := metrics.Counter("\n\t\x1btest", 101)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		data, err = counter.MarshalBinary()
		if err != nil {
			b.Fatal(err)
		}
	}

	_ = data
	_ = err
}

func BenchmarkMetric_UnmarshalBinary(b *testing.B) {
	var counter metrics.Metric
	var err error

	data := marshal(b, metrics.Counter("\n\t\x1btest", 101))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err = counter.UnmarshalBinary(data); err != nil {
			b.Fatal(err)
		}
	}

	_ = counter
	_ = err
}
