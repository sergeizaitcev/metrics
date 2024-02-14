package interceptors_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	pb "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/sergeizaitcev/metrics/pkg/interceptors"
)

func TestTrace(t *testing.T) {
	paramsCh := make(chan *interceptors.Params, 1)
	trace := func(params *interceptors.Params) {
		paramsCh <- params
	}

	testServer(t, interceptors.Trace(trace), func(check checkFunc) {
		res, err := check(context.Background(), &pb.HealthCheckRequest{Service: "test"})
		require.NoError(t, err)
		require.Equal(t, pb.HealthCheckResponse_SERVING, res.Status)
	})

	params := <-paramsCh
	require.Equal(t, "/grpc.health.v1.Health/Check", params.FullMethod)
	require.NoError(t, params.Error)
}
