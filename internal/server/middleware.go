package server

import (
	"compress/flate"

	"github.com/sergeizaitcev/metrics/pkg/logging"
	"github.com/sergeizaitcev/metrics/pkg/middleware"
	"github.com/sergeizaitcev/metrics/pkg/sign"
)

// NOTE: необходимо соблюдать порядок мидлварей в следующей последовательности
// rsa -> gzip -> sign -> trace -> subnet.
func (s *Server) middlewares() []middleware.Middleware {
	var middlewares []middleware.Middleware

	if s.opts.Key != nil {
		middlewares = append(middlewares, middleware.RSA(s.opts.Key))
	}

	middlewares = append(
		middlewares,
		middleware.Gzip(flate.BestCompression, "application/json", "text/html"),
	)

	if s.config.SHA256Key != "" {
		signer := sign.Signer(s.config.SHA256Key)
		middlewares = append(middlewares, middleware.Sign(signer))
	}

	paramsFunc := func(p *middleware.Params) {
		if p.Error != nil {
			s.opts.Logger.Log(logging.LevelError, p.Error.Error(),
				"method", p.Method,
				"path", p.Path,
				"status_code", p.StatusCode,
			)
		} else {
			s.opts.Logger.Log(logging.LevelInfo, "",
				"method", p.Method,
				"path", p.Path,
				"status_code", p.StatusCode,
				"elapsed", p.Elapsed.String(),
				"content_length", len(p.Body),
			)
		}
	}

	middlewares = append(middlewares, middleware.Trace(paramsFunc))

	if subnet := s.config.CIDR(); subnet != nil {
		middlewares = append(middlewares, middleware.Subnet(subnet))
	}

	return middlewares
}
