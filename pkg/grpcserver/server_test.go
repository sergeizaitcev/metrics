package grpcserver_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/sergeizaitcev/metrics/pkg/grpcserver"
	"github.com/sergeizaitcev/metrics/pkg/tcputil"
	"github.com/sergeizaitcev/metrics/pkg/testutil"
)

func grpcServer(t *testing.T) (srv *grpcserver.Server, host string) {
	port, err := tcputil.FreePort()
	require.NoError(t, err)
	host = net.JoinHostPort("localhost", port)

	healthSrv := health.NewServer()
	healthSrv.SetServingStatus("test", healthpb.HealthCheckResponse_SERVING)

	grpcSrv := grpc.NewServer()
	healthpb.RegisterHealthServer(grpcSrv, healthSrv)

	return grpcserver.New(host, grpcSrv), host
}

func grpcClient(t *testing.T, host string) healthpb.HealthClient {
	creds := grpc.WithTransportCredentials(insecure.NewCredentials())
	conn, err := grpc.Dial(host, creds)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	return healthpb.NewHealthClient(conn)
}

func TestServer(t *testing.T) {
	srv, host := grpcServer(t)
	errc := make(chan error)

	ctx, cancel := context.WithTimeout(testutil.Context(t), 3*time.Second)
	t.Cleanup(cancel)

	go func() { errc <- srv.ListenAndServe(ctx) }()
	go func() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
			client := grpcClient(t, host)
			res, err := client.Check(ctx, &healthpb.HealthCheckRequest{
				Service: "test",
			})
			require.NoError(t, err)
			require.Equal(t, healthpb.HealthCheckResponse_SERVING, res.Status)
		}
	}()

	select {
	case <-ctx.Done():
	case err := <-errc:
		require.NoError(t, err)
	}

	cancel()
	require.NoError(t, <-errc)
}
