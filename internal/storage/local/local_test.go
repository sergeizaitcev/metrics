package local_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/sergeizaitcev/metrics/internal/metrics"
	"github.com/sergeizaitcev/metrics/internal/storage/local"
)

func filename(t *testing.T) string {
	f, err := os.CreateTemp(t.TempDir(), "test-*.wal")
	require.NoError(t, err)
	name := f.Name()
	require.NoError(t, f.Close())
	return name
}

func snapshot() (snap []metrics.Metric, want []metrics.Metric) {
	snaps := [...][]metrics.Metric{
		metrics.Snapshot(),
		metrics.Snapshot(),
		metrics.Snapshot(),
	}

	snap = make([]metrics.Metric, 0, len(snaps)*len(snaps[0]))
	want = make([]metrics.Metric, len(snaps[0]))

	for i := 0; i < len(snaps); i++ {
		snap = append(snap, snaps[i]...)
		for j := 0; j < len(snaps[i]); j++ {
			metric := snaps[i][j]
			if metric.Kind() == metrics.KindCounter {
				metric = metrics.Counter(metric.Name(), metric.Int64()+want[j].Int64())
			}
			want[j] = metric
		}
	}

	return snap, want
}

func testStorage(t *testing.T, synced bool, values ...metrics.Metric) (*local.Storage, string) {
	name := filename(t)

	opts := &local.StorageOpts{}
	if !synced {
		opts.StoreInterval = time.Second
	}

	storage, err := local.New(name, opts)
	if err != nil {
		t.Log(err)
		t.SkipNow()
	}

	t.Cleanup(func() { storage.Close() })
	ctx := context.Background()

	for _, value := range values {
		var err error
		switch value.Kind() {
		case metrics.KindCounter:
			_, err = storage.Add(ctx, value)
		case metrics.KindGauge:
			_, err = storage.Set(ctx, value)
		}
		require.NoError(t, err)
	}

	return storage, name
}

func TestStorage_reopen(t *testing.T) {
	snap, want := snapshot()
	storage, name := testStorage(t, true, snap...)

	require.NoError(t, storage.Ping(context.Background()))
	require.NoError(t, storage.Close())

	opened, err := local.Open(name, nil)
	if err != nil {
		t.SkipNow()
	}

	t.Cleanup(func() { opened.Close() })

	ctx := context.Background()

	for _, metric := range want {
		got, err := opened.Get(ctx, metric.Name())
		require.NoError(t, err)

		require.True(
			t,
			metric.Equal(got),
			"want: %s\ngot: %s",
			metric.String(),
			got.String(),
		)
	}

	require.NoError(t, opened.Close())
}

func TestStorage_New(t *testing.T) {
	ctx := context.Background()

	t.Run("set", func(t *testing.T) {
		storage, _ := testStorage(t, false)

		testCases := []struct {
			name       string
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
				metric:    metrics.Counter("gauge", 1),
				wantError: true,
			},
		}

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

	t.Run("add", func(t *testing.T) {
		storage, _ := testStorage(t, false)

		testCases := []struct {
			name       string
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
				name:       "counter_first",
				metric:     metrics.Counter("counter", 1),
				wantMetric: metrics.Counter("counter", 1),
			},
			{
				name:      "not equal kinds",
				metric:    metrics.Gauge("counter", 1),
				wantError: true,
			},
			{
				name:       "counter_second",
				metric:     metrics.Counter("counter", 1),
				wantMetric: metrics.Counter("counter", 2),
			},
			{
				name:       "gauge_add",
				metric:     metrics.Gauge("gauge", 1),
				wantMetric: metrics.Gauge("gauge", 1),
			},
		}

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

	t.Run("get", func(t *testing.T) {
		storage, _ := testStorage(
			t,
			false,
			metrics.Gauge("gauge", 1),
			metrics.Counter("counter", 1),
		)

		testCases := []struct {
			name       string
			metric     string
			wantMetric metrics.Metric
			wantError  bool
		}{
			{
				name:      "not found",
				metric:    "invalid",
				wantError: true,
			},
			{
				name:       "counter",
				metric:     "counter",
				wantMetric: metrics.Counter("counter", 1),
			},
			{
				name:       "gauge",
				metric:     "gauge",
				wantMetric: metrics.Gauge("gauge", 1),
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
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
			metrics.Gauge("gauge", 1),
			metrics.Counter("counter", 1),
		}
		storage, _ := testStorage(t, false, want...)

		got, err := storage.GetAll(ctx)
		require.NoError(t, err)

		require.Equal(t, len(want), len(got))
		require.True(t, want[0].Equal(got[1]))
		require.True(t, want[1].Equal(got[0]))
	})
}
