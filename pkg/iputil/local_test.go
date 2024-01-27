package iputil_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sergeizaitcev/metrics/pkg/iputil"
)

func TestLocal(t *testing.T) {
	ip := iputil.Local()
	require.NotNil(t, ip)
}
