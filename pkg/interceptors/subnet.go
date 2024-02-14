package interceptors

import (
	"context"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/sergeizaitcev/metrics/pkg/interceptors/md"
)

// Subnet проверяет IP-адрес входящего запроса на вхождение в доверенную подсеть.
func Subnet(subnet *net.IPNet) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (
		resp any, err error,
	) {
		ip := md.GetRealIP(ctx)
		if ip == "" || !subnet.Contains(net.ParseIP(ip)) {
			return nil, status.Error(
				codes.Internal,
				"real IP address is not contained in the subnet",
			)
		}
		return handler(ctx, req)
	}
}
