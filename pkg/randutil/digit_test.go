package randutil_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sergeizaitcev/metrics/pkg/randutil"
)

func TestFloat64(t *testing.T) {
	a := randutil.Float64()
	b := randutil.Float64()
	c := randutil.Float64()

	require.NotEqual(t, a, b)
	require.NotEqual(t, b, c)
	require.NotEqual(t, c, a)
}
