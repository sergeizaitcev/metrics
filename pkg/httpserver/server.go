package httpserver

import (
	"context"
	"net"
	"net/http"
	"time"
)

// ServerOpts определяет не обязательные параметры сервера.
type ServerOpts struct {
	// Пользовательское прослушивание соединения.
	Listener net.Listener

	// Тайм-аут чтения запроса.
	ReadTimeout time.Duration

	// Тайм-аут записи ответа.
	WriteTimeout time.Duration

	// Время жизни не используемого `keep-alive` соединения.
	IdleTimeout time.Duration
}

// Server определяет HTTP-сервер.
type Server struct {
	lis net.Listener
	srv http.Server
}

// New возвращает новый экземпляр Server.
func New(h http.Handler, opts *ServerOpts) *Server {
	if opts == nil {
		opts = &ServerOpts{}
	}

	server := &Server{
		lis: opts.Listener,
		srv: http.Server{
			Handler:      h,
			ReadTimeout:  opts.ReadTimeout,
			WriteTimeout: opts.WriteTimeout,
			IdleTimeout:  opts.IdleTimeout,
		},
	}

	return server
}

// ListenAndServe слушает входящие запросы и блокируется до тех пор, пока
// не сработает контекст, не сработает метод Close или функция не вернёт ошибку.
func (s *Server) ListenAndServe(ctx context.Context) error {
	errc := make(chan error)
	go func() { errc <- s.Serve(ctx) }()

	select {
	case <-ctx.Done():
	case err := <-errc:
		return err
	}

	err := s.Close()
	<-errc

	return err
}

// Close завершает работу HTTP-сервера.
func (s *Server) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return s.srv.Shutdown(ctx)
}

// Serve слушает входящие запросы и блокируется до тех пор, пока не сработает
// метод Close или функция не вернёт ошибку.
func (s *Server) Serve(ctx context.Context) error {
	var (
		lis = s.lis
		err error
	)

	s.srv.BaseContext = func(net.Listener) context.Context { return ctx }

	if lis == nil {
		lis, err = net.Listen("tcp", "")
		if err != nil {
			return err
		}
		defer lis.Close()
	}

	return s.srv.Serve(lis)
}
