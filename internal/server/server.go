package server

import (
	"context"
	"crypto/rsa"
	"fmt"
	"net/http"

	"google.golang.org/grpc"
	_ "google.golang.org/grpc/encoding/gzip"

	pb "github.com/sergeizaitcev/metrics/api/proto/metrics"
	"github.com/sergeizaitcev/metrics/internal/configs"
	"github.com/sergeizaitcev/metrics/internal/storage"
	"github.com/sergeizaitcev/metrics/pkg/closer"
	"github.com/sergeizaitcev/metrics/pkg/grpcserver"
	"github.com/sergeizaitcev/metrics/pkg/httpserver"
	"github.com/sergeizaitcev/metrics/pkg/logging"
)

var defaultOpts = &ServerOpts{
	Logger: logging.Discard(),
}

// ServerOpts определяет не обязательные параметры для Server.
type ServerOpts struct {
	Logger *logging.Logger
	Key    *rsa.PrivateKey
}

// Server определяет сервер сбора метрик.
type Server struct {
	config *configs.Server
	opts   *ServerOpts
}

// New возвращает новый экземпляр Server.
func New(config *configs.Server, opts *ServerOpts) *Server {
	if opts == nil {
		opts = defaultOpts
	}
	if opts.Logger == nil {
		opts.Logger = defaultOpts.Logger
	}

	return &Server{
		config: config,
		opts:   opts,
	}
}

// Run запускает сервер сбора метрик и блокируется до тех пор, пока
// не сработает контекст или функция не вернёт ошибку.
func (s *Server) Run(ctx context.Context) (err error) {
	gracefulClose := closer.New()
	defer func() {
		firstErr := gracefulClose.Close()
		if firstErr != nil && err == nil {
			err = firstErr
		}
	}()

	storage, err := storage.NewStorage(s.config)
	if err != nil {
		return fmt.Errorf("init storage: %w", err)
	}
	gracefulClose.Add(ctx, storage.Close)

	httpSrv := s.httpServer(ctx, storage)
	gracefulClose.Add(ctx, httpSrv.Close)

	grpcSrv := s.grpcServer(ctx, storage)
	gracefulClose.Add(ctx, grpcSrv.Close)

	errChan := make(chan error, 2)

	go func() { errChan <- httpSrv.ListenAndServe(ctx) }()
	go func() { errChan <- grpcSrv.ListenAndServe(ctx) }()

	select {
	case <-ctx.Done():
	case err = <-errChan:
		return err
	}

	return nil
}

func (s *Server) httpServer(ctx context.Context, storage storage.Storage) *httpserver.Server {
	srv := &http.Server{
		Addr:    s.config.Address,
		Handler: NewHandler(storage, s.middlewares()...),
	}
	return httpserver.New(srv)
}

func (s *Server) grpcServer(ctx context.Context, storage storage.Storage) *grpcserver.Server {
	srv := grpc.NewServer(grpc.ChainUnaryInterceptor(s.interceptors()...))
	pb.RegisterMetricsServer(srv, newUpdateServer(s.config, storage))
	return grpcserver.New(s.config.StreamAddress, srv)
}
