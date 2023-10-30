package server

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
	//
	// По умолчанию 0.
	ReadTimeout time.Duration

	// Тайм-аут записи ответа.
	//
	// По умолчанию 0.
	WriteTimeout time.Duration

	// Время жизни не используемого `keep-alive` соединения.
	//
	// По умолчанию 0.
	IdleTimeout time.Duration
}

func (o *ServerOpts) clone() *ServerOpts {
	o2 := *o
	return &o2
}

// Server определяет HTTP-сервер.
type Server struct {
	handler http.Handler
	opts    *ServerOpts
}

// New возвращает новый экземпляр Server.
func New(h http.Handler, opts *ServerOpts) *Server {
	if opts == nil {
		opts = &ServerOpts{}
	}

	o2 := opts.clone()

	return &Server{
		handler: h,
		opts:    o2,
	}
}

// ListenAndServe слушает входящие запросы и блокируется до тех пор, пока
// не сработает контекст или функция не вернёт ошибку.
func (s *Server) ListenAndServe(ctx context.Context) error {
	var (
		lis = s.opts.Listener
		err error
	)

	if lis == nil {
		lis, err = net.Listen("tcp", "")
		if err != nil {
			return err
		}
		defer lis.Close()
	}

	server := http.Server{
		Handler:      s.handler,
		ReadTimeout:  s.opts.ReadTimeout,
		WriteTimeout: s.opts.WriteTimeout,
		IdleTimeout:  s.opts.IdleTimeout,
		BaseContext:  func(net.Listener) context.Context { return ctx },
	}

	errc := make(chan error)
	go func() { errc <- server.Serve(lis) }()

	select {
	case <-ctx.Done():
	case err := <-errc:
		return err
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer shutdownCancel()

	err = server.Shutdown(shutdownCtx)
	<-errc

	return err
}
