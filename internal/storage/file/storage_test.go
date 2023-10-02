package file_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sergeizaitcev/metrics/internal/metrics"
	"github.com/sergeizaitcev/metrics/internal/storage/file"
)

func TestFile(t *testing.T) {
	fd, err := os.CreateTemp(t.TempDir(), "test*.json")
	require.NoError(t, err)

	filename := fd.Name()
	require.NoError(t, fd.Close())

	f, err := file.OpenSync(filename)
	require.NoError(t, err)

	want := []metrics.Metric{
		metrics.Counter("counter", 1),
		metrics.Gauge("gauge", 1),
	}

	for _, metric := range want {
		require.NoError(t, f.Append(metric))
	}

	require.NoError(t, f.Flush())

	got, err := f.ReadAll()
	require.NoError(t, err)
	require.Equal(t, want, got)

	require.NoError(t, f.Close())
}
