package metrics_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/sergeizaitcev/metrics/internal/metrics"
	"github.com/sergeizaitcev/metrics/internal/metrics/mocks"
)

func TestMetrics(t *testing.T) {
	ctx := context.Background()

	t.Run("save", func(t *testing.T) {
		testCases := []struct {
			name      string
			method    string
			metric    metrics.Metric
			mockError error
			wantError bool
		}{
			{
				name:   "add counter",
				method: "Add",
				metric: metrics.Counter("counter", 1),
			},
			{
				name:      "add counter error",
				method:    "Add",
				metric:    metrics.Counter("counter", 1),
				mockError: errors.New("error"),
				wantError: true,
			},
			{
				name:   "set gauge",
				method: "Set",
				metric: metrics.Gauge("gauge", 1),
			},
			{
				name:      "set gauge error",
				method:    "Set",
				metric:    metrics.Gauge("gauge", 1),
				mockError: errors.New("error"),
				wantError: true,
			},
			{
				name:      "unknown kind",
				metric:    metrics.Metric{},
				wantError: true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				storage := mocks.NewMockStorage()
				if tc.method != "" {
					storage.On(tc.method, mock.Anything, tc.metric).
						Return(metrics.Metric{}, tc.mockError)
				}

				fileStorage := mocks.NewMockFileStorage()
				if tc.method != "" {
					fileStorage.On("Append", tc.metric).Return(nil)
				}

				m := metrics.NewMetrics(storage, fileStorage)

				_, err := m.Save(ctx, tc.metric)
				if tc.wantError {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			})
		}
	})

	t.Run("lookup", func(t *testing.T) {
		testCases := []struct {
			name       string
			metric     string
			mockMetric metrics.Metric
			mockError  error
			wantError  bool
		}{
			{
				name:       "found",
				metric:     "counter",
				mockMetric: metrics.Counter("counter", 1),
			},
			{
				name:      "not found",
				metric:    "counter",
				wantError: true,
			},
			{
				name:      "error",
				metric:    "counter",
				mockError: errors.New("error"),
				wantError: true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				storage := mocks.NewMockStorage()
				storage.On("Get", ctx, tc.metric).Return(tc.mockMetric, tc.mockError)

				m := metrics.NewMetrics(storage, nil)

				got, err := m.Lookup(ctx, tc.metric)
				if tc.wantError {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
					require.True(t, tc.mockMetric.Equal(got))
				}
			})
		}
	})

	t.Run("all", func(t *testing.T) {
		testCases := []struct {
			name        string
			mockMetrics []metrics.Metric
			mockError   error
			wantError   bool
		}{
			{
				name: "success",
				mockMetrics: []metrics.Metric{
					metrics.Counter("counter", 1),
					metrics.Gauge("gauge", 2),
				},
			},
			{
				name:      "error",
				mockError: errors.New("error"),
				wantError: true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				storage := mocks.NewMockStorage()
				storage.On("GetAll", ctx).Return(tc.mockMetrics, tc.mockError)

				m := metrics.NewMetrics(storage, nil)

				got, err := m.All(ctx)
				if tc.wantError {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
					require.Equal(t, len(tc.mockMetrics), len(got))

					for i := range tc.mockMetrics {
						require.True(t, tc.mockMetrics[i].Equal(got[i]))
					}
				}
			})
		}
	})
}

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
