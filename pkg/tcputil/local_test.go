package tcputil_test

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sergeizaitcev/metrics/pkg/tcputil"
)

func TestLocal(t *testing.T) {
	ip := tcputil.Local()
	require.NotNil(t, ip)
}

func TestFreePort(t *testing.T) {
	port, err := tcputil.FreePort()
	require.NoError(t, err)

	addr := net.JoinHostPort("", port)

	l, err := net.Listen("tcp", addr)
	require.NoError(t, err)
	require.NoError(t, l.Close())
}
