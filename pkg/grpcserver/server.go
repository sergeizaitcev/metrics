package grpcserver

import (
	"context"
	"net"

	"google.golang.org/grpc"
)

// Server определяет надстройку над gRPC-сервером.
type Server struct {
	addr string
	srv  *grpc.Server
}

// New возвращает новый экземпляр Server.
func New(addr string, srv *grpc.Server) *Server {
	return &Server{
		addr: addr,
		srv:  srv,
	}
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

	s.Close()
	<-errc

	return nil
}

// Close завершает работу gRPC-сервера.
func (s *Server) Close() {
	s.srv.GracefulStop()
}

// Serve слушает входящие запросы и блокируется до тех пор, пока не сработает
// метод Close или функция не вернёт ошибку.
func (s *Server) Serve(ctx context.Context) error {
	lis, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	defer lis.Close()
	return s.srv.Serve(lis)
}
