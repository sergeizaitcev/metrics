package postgres_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/sergeizaitcev/metrics/internal/metrics"
	"github.com/sergeizaitcev/metrics/internal/storage/postgres"
)

func testContext(t *testing.T) context.Context {
	deadline, ok := t.Deadline()
	if !ok {
		deadline = time.Now().Add(15 * time.Second)
	}

	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	t.Cleanup(cancel)

	return ctx
}

func testStorage(t *testing.T) (*postgres.Storage, context.Context) {
	const (
		dsn = "postgres://postgres:postgres@localhost:5432/practicum?sslmode=disable"
	)

	t.Helper()

	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)

	ctx := testContext(t)

	storage := postgres.New(db)
	t.Cleanup(func() { storage.Close() })

	err = storage.Ping(ctx)
	if err != nil {
		t.Log(err)
		t.SkipNow()
	}

	require.NoError(t, storage.Up(ctx))
	t.Cleanup(func() { storage.Down(ctx) })

	return storage, ctx
}

func TestStorage(t *testing.T) {
	t.Run("add", func(t *testing.T) {
		testCases := []struct {
			name       string
			metric     metrics.Metric
			wantMetric metrics.Metric
			wantError  bool
		}{
			{
				name:      "empty",
				wantError: true,
			},
			{
				name:      "gauge",
				metric:    metrics.Gauge("gauge", 1),
				wantError: true,
			},
			{
				name:       "insert",
				metric:     metrics.Counter("counter", 1),
				wantMetric: metrics.Counter("counter", 1),
			},
			{
				name:       "first update",
				metric:     metrics.Counter("counter", 1),
				wantMetric: metrics.Counter("counter", 2),
			},
			{
				name:       "second update",
				metric:     metrics.Counter("counter", 2),
				wantMetric: metrics.Counter("counter", 4),
			},
		}

		storage, ctx := testStorage(t)

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
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

	t.Run("set", func(t *testing.T) {
		testCases := []struct {
			name       string
			metric     metrics.Metric
			wantMetric metrics.Metric
			wantError  bool
		}{
			{
				name:      "empty",
				wantError: true,
			},
			{
				name:      "counter",
				metric:    metrics.Counter("counter", 1),
				wantError: true,
			},
			{
				name:   "insert",
				metric: metrics.Gauge("gauge", 1),
			},
			{
				name:       "first update",
				metric:     metrics.Gauge("gauge", 2),
				wantMetric: metrics.Gauge("gauge", 1),
			},
			{
				name:       "second update",
				metric:     metrics.Gauge("gauge", 4),
				wantMetric: metrics.Gauge("gauge", 2),
			},
		}

		storage, ctx := testStorage(t)

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
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

	t.Run("get", func(t *testing.T) {
		testCases := []struct {
			name       string
			wantMetric metrics.Metric
			wantError  bool
		}{
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

		err := storage.SaveMany(ctx, []metrics.Metric{
			metrics.Counter("counter", 1),
			metrics.Gauge("gauge", 1),
		})
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

		for _, metric := range want {
			var err error
			switch metric.Kind() {
			case metrics.KindCounter:
				_, err = storage.Add(ctx, metric)
			case metrics.KindGauge:
				_, err = storage.Set(ctx, metric)
			}
			require.NoError(t, err)
		}

		values, err := storage.GetAll(ctx)
		require.NoError(t, err)

		require.Len(t, values, len(want))
		require.Equal(t, want, values)
	})
}
