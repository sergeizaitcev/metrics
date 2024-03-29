package httpserver

import (
	"context"
	"net"
	"net/http"
	"time"
)

// Server определяет надстройку над HTTP-сервером.
type Server struct {
	srv *http.Server
}

// New возвращает новый экземпляр Server.
func New(srv *http.Server) *Server {
	return &Server{srv: srv}
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
	lis, err := net.Listen("tcp", s.srv.Addr)
	if err != nil {
		return err
	}
	defer lis.Close()
	s.srv.BaseContext = func(net.Listener) context.Context { return ctx }
	return s.srv.Serve(lis)
}
