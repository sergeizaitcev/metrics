package interceptors

import (
	"context"
	"time"

	"google.golang.org/grpc"
)

// Params определяет параметры запроса.
type Params struct {
	FullMethod string
	Elapsed    time.Duration
	Error      error
}

// Trace передает параметры запроса в paramsFunc.
func Trace(paramsFunc func(*Params)) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		start := time.Now()
		resp, err = handler(ctx, req)
		elapsed := time.Since(start)
		paramsFunc(&Params{
			FullMethod: info.FullMethod,
			Elapsed:    elapsed,
			Error:      err,
		})
		return resp, err
	}
}
