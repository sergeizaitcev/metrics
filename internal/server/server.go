package server

import (
	"context"
	"fmt"
	"net"

	"github.com/sergeizaitcev/metrics/internal/configs"
	"github.com/sergeizaitcev/metrics/internal/storage"
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
	lis     net.Listener
	srv     *httpserver.Server
	storage storage.Storage
}

// New возвращает новый экземпляр Server.
func New(config *configs.Server, opts *ServerOpts) (*Server, error) {
	if opts == nil {
		opts = defaultOpts
	}
	if opts.Logger == nil {
		opts.Logger = defaultOpts.Logger
	}

	s := new(Server)

	err := s.init(config, opts)
	if err != nil {
		return nil, fmt.Errorf("init server: %w", err)
	}

	return s, nil
}

func (s *Server) init(config *configs.Server, opts *ServerOpts) error {
	var err error

	s.storage, err = storage.NewStorage(config.Storage)
	if err != nil {
		return fmt.Errorf("init storage: %w", err)
	}

	s.lis, err = net.Listen("tcp", config.Address)
	if err != nil {
		return fmt.Errorf("listen to %s: %w", config.Address, err)
	}

	mws := newMiddlewares(config, opts)
	handler := NewHandler(s.storage, mws...)

	srvOpts := &httpserver.ServerOpts{
		Listener: s.lis,
	}
	s.srv = httpserver.New(handler, srvOpts)

	return nil
}

// Close завершает работу сервера.
func (s *Server) Close() error {
	var firstErr error

	err := s.srv.Close()
	if err != nil {
		firstErr = err
	}

	err = s.lis.Close()
	if err != nil && firstErr == nil {
		firstErr = err
	}

	err = s.storage.Close()
	if err != nil && firstErr == nil {
		firstErr = err
	}

	return firstErr
}

// Run запускает сервер сбора метрик и блокируется до тех пор, пока
// не сработает контекст или функция не вернёт ошибку.
func (s *Server) Run(ctx context.Context) error {
	return s.srv.ListenAndServe(ctx)
}
