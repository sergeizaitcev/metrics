package metrics_test

import (
	"context"
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

				m := metrics.NewMetrics(storage)

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

				m := metrics.NewMetrics(storage)

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

				m := metrics.NewMetrics(storage)

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
