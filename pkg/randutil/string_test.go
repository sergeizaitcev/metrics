package randutil_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sergeizaitcev/metrics/pkg/randutil"
)

func TestString(t *testing.T) {
	testCases := []struct {
		n    int
		want int
	}{
		{0, 0},
		{-1, 0},
		{5, 5},
		{20, 20},
	}

	for _, tc := range testCases {
		s := randutil.String(tc.n)
		t.Log(s)
		require.Len(t, s, tc.want)
	}
}
