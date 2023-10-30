package postgres_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sergeizaitcev/metrics/internal/metrics"
	"github.com/sergeizaitcev/metrics/internal/storage/postgres"
	"github.com/sergeizaitcev/metrics/pkg/testutil"
)

func testStorage(t *testing.T) (*postgres.Storage, context.Context) {
	t.Helper()

	const dsn = "postgres://postgres:postgres@localhost:5432/practicum?sslmode=disable"

	storage, err := postgres.New(dsn)
	require.NoError(t, err)
	t.Cleanup(func() { storage.Close() })

	ctx := testutil.Context(t)

	if err = storage.Ping(ctx); err != nil {
		t.Log(err)
		t.SkipNow()
	}

	require.NoError(t, storage.MigrateUp(ctx))
	t.Cleanup(func() { storage.MigrateDown(ctx) })

	return storage, ctx
}

func TestStorage(t *testing.T) {
	t.Run("save", func(t *testing.T) {
		testCases := []struct {
			name        string
			metrics     []metrics.Metric
			wantMetrics []metrics.Metric
			wantError   bool
		}{
			{
				name:      "empty",
				wantError: true,
			},
			{
				name:        "first insert",
				metrics:     []metrics.Metric{metrics.Counter("counter", 1), {}},
				wantMetrics: []metrics.Metric{metrics.Counter("counter", 1), {}},
			},
			{
				name:        "second insert",
				metrics:     []metrics.Metric{{}, metrics.Counter("counter", 1)},
				wantMetrics: []metrics.Metric{{}, metrics.Counter("counter", 2)},
			},
			{
				name:        "first update",
				metrics:     []metrics.Metric{metrics.Gauge("gauge", 1)},
				wantMetrics: []metrics.Metric{{}},
			},
			{
				name:        "second update",
				metrics:     []metrics.Metric{metrics.Gauge("gauge", 2)},
				wantMetrics: []metrics.Metric{metrics.Gauge("gauge", 1)},
			},
		}

		storage, ctx := testStorage(t)

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				got, err := storage.Save(ctx, tc.metrics...)
				if tc.wantError {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
					require.Len(t, got, len(tc.wantMetrics))
					for i, want := range tc.wantMetrics {
						require.True(t, want.Equal(got[i]))
					}
				}
			})
		}
	})

	t.Run("get", func(t *testing.T) {
		testCases := []struct {
			name       string
			wantMetric metrics.Metric
			wantError  bool
		}{
			{
				name:       "not found",
				wantMetric: metrics.Counter("invalid", 0),
				wantError:  true,
			},
			{
				name:       "counter",
				wantMetric: metrics.Counter("counter", 1),
			},
			{
				name:       "gauge",
				wantMetric: metrics.Gauge("gauge", 1),
			},
			{
				name:      "invalid",
				wantError: true,
			},
		}

		storage, ctx := testStorage(t)

		_, err := storage.Save(ctx,
			metrics.Counter("counter", 1),
			metrics.Gauge("gauge", 1),
		)
		require.NoError(t, err)

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				got, err := storage.Get(ctx, tc.name)
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
		storage, ctx := testStorage(t)
		want := []metrics.Metric{
			metrics.Counter("counter", 1),
			metrics.Gauge("gauge", 1),
		}

		_, err := storage.Save(ctx, want...)
		require.NoError(t, err)

		values, err := storage.GetAll(ctx)
		require.NoError(t, err)

		require.Len(t, values, len(want))
		require.Equal(t, want, values)
	})

	t.Run("not_found", func(t *testing.T) {
		storage, ctx := testStorage(t)
		_, err := storage.GetAll(ctx)
		require.Error(t, err)
	})
}
