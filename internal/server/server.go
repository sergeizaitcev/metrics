package server

import (
	"context"
	"fmt"
	"net"

	"github.com/sergeizaitcev/metrics/internal/configs"
	"github.com/sergeizaitcev/metrics/internal/storage"
	"github.com/sergeizaitcev/metrics/pkg/closer"
	"github.com/sergeizaitcev/metrics/pkg/httpserver"
	"github.com/sergeizaitcev/metrics/pkg/logging"
)

var defaultOpts = &ServerOpts{
	Logger: logging.Discard(),
}

// ServerOpts определяет не обязательные параметры для Server.
type ServerOpts struct {
	Logger *logging.Logger
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

	storage, err := storage.NewStorage(s.config.Storage)
	if err != nil {
		return fmt.Errorf("init storage: %w", err)
	}
	gracefulClose.Add(ctx, storage.Close)

	lis, err := net.Listen("tcp", s.config.Address)
	if err != nil {
		return fmt.Errorf("listen to %s: %w", s.config.Address, err)
	}
	gracefulClose.Add(ctx, lis.Close)

	mws := newMiddlewares(s.config, s.opts)
	handler := NewHandler(storage, mws...)

	srv := httpserver.New(handler, &httpserver.ServerOpts{
		Listener: lis,
	})
	gracefulClose.Add(ctx, srv.Close)

	return srv.ListenAndServe(ctx)
}
