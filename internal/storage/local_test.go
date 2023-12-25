package storage_test

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/sergeizaitcev/metrics/internal/metrics"
	"github.com/sergeizaitcev/metrics/internal/storage"
	"github.com/sergeizaitcev/metrics/pkg/testutil"
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

func testLocal(
	t *testing.T,
	synced bool,
	values ...metrics.Metric,
) (*storage.Local, string) {
	name := filename(t)

	opts := &storage.LocalOpts{}
	if !synced {
		opts.StoreInterval = 50 * time.Millisecond
	}

	storage, err := storage.NewLocal(name, opts)
	if err != nil {
		t.Log(err)
		t.SkipNow()
	}

	ctx := testutil.Context(t)

	t.Cleanup(func() { storage.Close() })
	storage.Save(ctx, values...)

	return storage, name
}

func TestStorage_reopen(t *testing.T) {
	snap, want := snapshot()
	store, name := testLocal(t, true, snap...)

	ctx := testutil.Context(t)

	require.NoError(t, store.Ping(ctx))
	require.NoError(t, store.Close())

	opened, err := storage.NewLocal(name, &storage.LocalOpts{Restore: true})
	if err != nil {
		t.SkipNow()
	}

	t.Cleanup(func() { opened.Close() })

	for _, metric := range want {
		got, err := opened.Get(ctx, metric.Name())
		require.NoError(t, err)

		require.True(
			t,
			metric.Equal(got),
			"want: %s\ngot: %s",
			metric.GoString(),
			got.GoString(),
		)
	}

	require.NoError(t, opened.Close())
}

func TestLocal(t *testing.T) {
	ctx := testutil.Context(t)

	t.Run("save", func(t *testing.T) {
		storage, _ := testLocal(t, false)

		testCases := []struct {
			name        string
			metrics     []metrics.Metric
			wantMetrics []metrics.Metric
			wantError   bool
		}{
			{
				name:      "empty",
				metrics:   nil,
				wantError: true,
			},
			{
				name:        "set",
				metrics:     []metrics.Metric{metrics.Gauge("gauge", 1), {}},
				wantMetrics: []metrics.Metric{{}, {}},
			},
			{
				name:        "re-set",
				metrics:     []metrics.Metric{{}, metrics.Gauge("gauge", 2)},
				wantMetrics: []metrics.Metric{{}, metrics.Gauge("gauge", 1)},
			},
			{
				name:      "unknown kind",
				metrics:   []metrics.Metric{metrics.Counter("gauge", 1)},
				wantError: true,
			},
			{
				name:        "add",
				metrics:     []metrics.Metric{metrics.Counter("counter", 1), {}},
				wantMetrics: []metrics.Metric{metrics.Counter("counter", 1), {}},
			},
			{
				name:        "add_2",
				metrics:     []metrics.Metric{{}, metrics.Counter("counter", 1)},
				wantMetrics: []metrics.Metric{{}, metrics.Counter("counter", 2)},
			},
		}

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
		storage, _ := testLocal(
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
		storage, _ := testLocal(t, false, want...)

		got, err := storage.GetAll(ctx)
		require.NoError(t, err)

		require.Equal(t, len(want), len(got))
		require.True(t, want[0].Equal(got[1]))
		require.True(t, want[1].Equal(got[0]))
	})
}
