package local_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sergeizaitcev/metrics/internal/metrics"
	"github.com/sergeizaitcev/metrics/internal/storage/local"
)

func TestStorage(t *testing.T) {
	ctx := context.Background()

	t.Run("set", func(t *testing.T) {
		storage := local.NewStorage()

		_, err := storage.Set(ctx, metrics.Metric{})
		require.Error(t, err)

		m1, err := storage.Set(ctx, metrics.Counter("test", 1))
		require.NoError(t, err)
		require.Empty(t, m1.Kind())

		_, err = storage.Set(ctx, metrics.Gauge("test", 1))
		require.Error(t, err)

		m2, err := storage.Set(ctx, metrics.Counter("test", 2))
		require.NoError(t, err)
		require.NotEmpty(t, m2.Kind())
	})

	t.Run("add", func(t *testing.T) {
		storage := local.NewStorage()

		_, err := storage.Add(ctx, metrics.Metric{})
		require.Error(t, err)

		m1, err := storage.Add(ctx, metrics.Counter("test", 1))
		require.NoError(t, err)
		require.NotEmpty(t, m1.Kind())

		_, err = storage.Add(ctx, metrics.Gauge("test", 1))
		require.Error(t, err)

		m2, err := storage.Add(ctx, metrics.Counter("test", 2))
		require.NoError(t, err)
		require.NotEmpty(t, m2.Kind())
	})

	t.Run("get", func(t *testing.T) {
		storage := local.NewStorage()

		_, err := storage.Set(ctx, metrics.Counter("test", 1))
		require.NoError(t, err)

		m1, err := storage.Get(ctx, "test")
		require.NoError(t, err)
		require.NotEmpty(t, m1.Kind())

		m2, err := storage.Get(ctx, "invalid")
		require.NoError(t, err)
		require.Empty(t, m2.Kind())
	})

	t.Run("get_all", func(t *testing.T) {
		storage := local.NewStorage()

		_, err := storage.Set(ctx, metrics.Counter("counter", 1))
		require.NoError(t, err)

		_, err = storage.Set(ctx, metrics.Gauge("gauge", 1))
		require.NoError(t, err)

		values, err := storage.GetAll(ctx)
		require.NoError(t, err)

		require.Len(t, values, 2)
	})

	t.Run("del", func(t *testing.T) {
		storage := local.NewStorage()

		_, err := storage.Set(ctx, metrics.Gauge("test", 1))
		require.NoError(t, err)

		m1, err := storage.Del(ctx, "test")
		require.NoError(t, err)
		require.NotEmpty(t, m1.Kind())

		m2, err := storage.Get(ctx, "test")
		require.NoError(t, err)
		require.Empty(t, m2.Kind())
	})
}
