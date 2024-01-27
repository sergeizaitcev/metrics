package metrics_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sergeizaitcev/metrics/internal/metrics"
)

func TestSign(t *testing.T) {
	hash := metrics.Sign("test", []metrics.Metric{
		metrics.Counter("counter", 1),
		metrics.Gauge("gauge", 1.0),
	})
	require.Equal(t, "PUzgLXpZnaZbeIIi2ey0JVTMxUqr7b2lKpLkweXXYSA", hash)
}
