package local_test

import (
	"context"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sergeizaitcev/metrics/internal/metrics"
	"github.com/sergeizaitcev/metrics/internal/storage/local"
)

func testStorage(t *testing.T, ms ...metrics.Metric) *local.Storage {
	t.Helper()

	s := local.NewStorage()

	for _, m := range ms {
		if m.Kind() == metrics.KindUnknown {
			continue
		}
		if _, err := s.Set(context.Background(), m); err != nil {
			t.Fatal(err)
		}
	}

	return s
}

func TestStorage(t *testing.T) {
	ctx := context.Background()

	t.Run("set", func(t *testing.T) {
		testCases := []struct {
			name       string
			preload    []metrics.Metric
			metric     metrics.Metric
			wantMetric metrics.Metric
			wantError  bool
		}{
			{
				name:       "set",
				metric:     metrics.Gauge("gauge", 1),
				wantMetric: metrics.Metric{},
			},
			{
				name:       "re-set",
				preload:    []metrics.Metric{metrics.Gauge("gauge", 1)},
				metric:     metrics.Gauge("gauge", 2),
				wantMetric: metrics.Gauge("gauge", 1),
			},
			{
				name:      "unknown kind",
				metric:    metrics.Metric{},
				wantError: true,
			},
			{
				name:      "not equal kinds",
				preload:   []metrics.Metric{metrics.Gauge("gauge", 1)},
				metric:    metrics.Counter("gauge", 1),
				wantError: true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				storage := testStorage(t, tc.preload...)

				got, err := storage.Set(ctx, tc.metric)
				if tc.wantError {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
					require.True(t, tc.wantMetric.Equal(got))
				}
			})
		}
	})

	t.Run("add", func(t *testing.T) {
		testCases := []struct {
			name       string
			preload    []metrics.Metric
			metric     metrics.Metric
			wantMetric metrics.Metric
			wantError  bool
		}{
			{
				name:      "unknown kind",
				metric:    metrics.Metric{},
				wantError: true,
			},
			{
				name:      "not equal kinds",
				preload:   []metrics.Metric{metrics.Gauge("counter", 1)},
				metric:    metrics.Counter("counter", 1),
				wantError: true,
			},
			{
				name:       "counter_first",
				metric:     metrics.Counter("counter", 1),
				wantMetric: metrics.Counter("counter", 1),
			},
			{
				name:       "counter_second",
				preload:    []metrics.Metric{metrics.Counter("counter", 1)},
				metric:     metrics.Counter("counter", 1),
				wantMetric: metrics.Counter("counter", 2),
			},
			{
				name:       "gauge_add",
				preload:    []metrics.Metric{metrics.Gauge("gauge", 1)},
				metric:     metrics.Gauge("gauge", 1),
				wantMetric: metrics.Gauge("gauge", 2),
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				storage := testStorage(t, tc.preload...)

				got, err := storage.Add(ctx, tc.metric)
				if tc.wantError {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
					require.True(t, tc.wantMetric.Equal(got))
				}
			})
		}
	})

	t.Run("get", func(t *testing.T) {
		preload := []metrics.Metric{
			metrics.Counter("counter", 1),
			metrics.Gauge("gauge", 1),
		}

		testCases := []struct {
			name       string
			preload    []metrics.Metric
			metric     string
			wantMetric metrics.Metric
			wantError  bool
		}{
			{
				name:       "not found",
				metric:     "invalid",
				wantMetric: metrics.Metric{},
			},
			{
				name:       "not found in content",
				preload:    preload,
				metric:     "invalid",
				wantMetric: metrics.Metric{},
			},
			{
				name:       "counter",
				preload:    preload,
				metric:     "counter",
				wantMetric: metrics.Counter("counter", 1),
			},
			{
				name:       "gauge",
				preload:    preload,
				metric:     "gauge",
				wantMetric: metrics.Gauge("gauge", 1),
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				storage := testStorage(t, tc.preload...)

				got, err := storage.Get(ctx, tc.metric)
				if tc.wantError {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
					require.True(t, tc.wantMetric.Equal(got))
				}
			})
		}
	})

	t.Run("get_all", func(t *testing.T) {
		want := []metrics.Metric{
			metrics.Gauge("gauge", 1),     // 1
			metrics.Counter("counter", 1), // 0
		}

		storage := testStorage(t, want...)
		sort.SliceStable(want, func(i, j int) bool {
			return want[i].Name() < want[j].Name()
		})

		got, err := storage.GetAll(ctx)
		require.NoError(t, err)

		require.Equal(t, len(want), len(got))
		for i := range got {
			require.True(t, want[i].Equal(got[i]))
		}
	})

	t.Run("del", func(t *testing.T) {
		testCases := []struct {
			name       string
			preload    []metrics.Metric
			metric     string
			wantMetric metrics.Metric
			wantError  bool
		}{
			{
				name:       "success",
				preload:    []metrics.Metric{metrics.Counter("counter", 1)},
				metric:     "counter",
				wantMetric: metrics.Counter("counter", 1),
			},
			{
				name:       "not found",
				preload:    []metrics.Metric{metrics.Counter("counter", 1)},
				metric:     "invalid",
				wantMetric: metrics.Metric{},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				storage := testStorage(t, tc.preload...)

				got, err := storage.Del(ctx, tc.metric)
				if tc.wantError {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
					require.True(t, tc.wantMetric.Equal(got))
				}
			})
		}
	})
}
