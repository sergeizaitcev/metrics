package server

import (
	"google.golang.org/grpc"

	"github.com/sergeizaitcev/metrics/pkg/interceptors"
	"github.com/sergeizaitcev/metrics/pkg/logging"
)

func (s *Server) interceptors() []grpc.UnaryServerInterceptor {
	var values []grpc.UnaryServerInterceptor

	paramsFunc := func(p *interceptors.Params) {
		if p.Error != nil {
			s.opts.Logger.Log(logging.LevelError, p.Error.Error(),
				"method", p.FullMethod,
			)
		} else {
			s.opts.Logger.Log(logging.LevelInfo, "",
				"method", p.FullMethod,
				"elapsed", p.Elapsed.String(),
			)
		}
	}

	values = append(values, interceptors.Trace(paramsFunc))

	if subnet := s.config.CIDR(); subnet != nil {
		values = append(values, interceptors.Subnet(subnet))
	}

	return values
}
