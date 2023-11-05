package server

import (
	"compress/flate"

	"github.com/sergeizaitcev/metrics/internal/configs"
	"github.com/sergeizaitcev/metrics/pkg/logging"
	"github.com/sergeizaitcev/metrics/pkg/middleware"
	"github.com/sergeizaitcev/metrics/pkg/sign"
)

// NOTE: необходимо соблюдать порядок мидлварей в следующей последовательности
// gzip -> sign -> trace.
func newMiddlewares(config *configs.Server, opts *ServerOpts) []middleware.Middleware {
	var middlewares []middleware.Middleware

	middlewares = append(
		middlewares,
		middleware.Gzip(flate.BestCompression, "application/json", "text/html"),
	)

	if config.SHA256Key != "" {
		signer := sign.Signer(config.SHA256Key)
		middlewares = append(middlewares, middleware.Sign(signer))
	}

	paramsFunc := func(p *middleware.Params) {
		if p.Error != nil {
			opts.Logger.Log(logging.LevelError, p.Error.Error(),
				"method", p.Method,
				"path", p.Path,
				"status_code", p.StatusCode,
			)
		} else {
			opts.Logger.Log(logging.LevelInfo, "",
				"method", p.Method,
				"path", p.Path,
				"status_code", p.StatusCode,
				"elapsed", p.Elapsed.String(),
				"content_length", len(p.Body),
			)
		}
	}

	return append(middlewares, middleware.Trace(paramsFunc))
}
