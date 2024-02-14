package interceptors_test

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	pb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"

	"github.com/sergeizaitcev/metrics/pkg/interceptors"
	"github.com/sergeizaitcev/metrics/pkg/interceptors/md"
)

func newLocalListener() *bufconn.Listener {
	const bufSize int = 4 << 10
	return bufconn.Listen(bufSize)
}

func contextDialer(lis *bufconn.Listener) func(context.Context, string) (net.Conn, error) {
	return func(ctx context.Context, _ string) (net.Conn, error) {
		return lis.DialContext(ctx)
	}
}

type checkFunc func(ctx context.Context, in *pb.HealthCheckRequest, opts ...grpc.CallOption) (*pb.HealthCheckResponse, error)

func testServer(
	t *testing.T,
	interceptor grpc.UnaryServerInterceptor,
	check func(checkFunc),
) {
	t.Helper()

	hsrv := health.NewServer()
	hsrv.SetServingStatus("test", pb.HealthCheckResponse_SERVING)

	srv := grpc.NewServer(grpc.UnaryInterceptor(interceptor))
	srv.RegisterService(&pb.Health_ServiceDesc, hsrv)

	lis := newLocalListener()
	t.Cleanup(func() { lis.Close() })

	errc := make(chan error, 1)
	go func() { errc <- srv.Serve(lis) }()

	conn, err := grpc.Dial(lis.Addr().Network(),
		grpc.WithContextDialer(contextDialer(lis)),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })

	check(pb.NewHealthClient(conn).Check)

	srv.Stop()
	require.NoError(t, <-errc)
}

func TestSubnet(t *testing.T) {
	testCases := []struct {
		name      string
		context   context.Context
		subnet    *net.IPNet
		wantError bool
	}{
		{
			name:    "ipv4",
			context: md.SetRealIP(context.Background(), "127.0.0.1"),
			subnet: func() *net.IPNet {
				_, subnet, _ := net.ParseCIDR("127.0.0.1/24")
				return subnet
			}(),
			wantError: false,
		},
		{
			name:    "ipv6",
			context: md.SetRealIP(context.Background(), "::ffff:127.0.0.1"),
			subnet: func() *net.IPNet {
				_, subnet, _ := net.ParseCIDR("::ffff:127.0.0.1/124")
				return subnet
			}(),
			wantError: false,
		},
		{
			name: "empty",
			context: func() context.Context {
				ctx := context.Background()
				md := metadata.New(nil)
				return metadata.NewOutgoingContext(ctx, md)
			}(),
			subnet: func() *net.IPNet {
				_, subnet, _ := net.ParseCIDR("127.0.0.1/24")
				return subnet
			}(),
			wantError: true,
		},
		{
			name:    "no contains",
			context: md.SetRealIP(context.Background(), "127.0.1.1"),
			subnet: func() *net.IPNet {
				_, subnet, _ := net.ParseCIDR("127.0.0.1/24")
				return subnet
			}(),
			wantError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testServer(t, interceptors.Subnet(tc.subnet), func(check checkFunc) {
				res, err := check(tc.context, &pb.HealthCheckRequest{Service: "test"})
				if tc.wantError {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
					require.Equal(t, pb.HealthCheckResponse_SERVING, res.Status)
				}
			})
		})
	}
}
