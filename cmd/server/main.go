package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sergeizaitcev/metrics/internal/metrics"
	"github.com/sergeizaitcev/metrics/internal/storage/local"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	if err := parseFlags(); err != nil {
		return err
	}

	storage := local.NewStorage()
	metrics := metrics.NewMetrics(storage)
	router := newRouter(metrics)

	return listenAndServe(router)
}

func listenAndServe(handler http.Handler) error {
	baseCtx := context.Background()

	ctx, cancel := signal.NotifyContext(baseCtx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	server := http.Server{
		Addr:         flagAddress,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
		BaseContext:  func(net.Listener) context.Context { return ctx },
	}

	errc := make(chan error)
	go func() { errc <- server.ListenAndServe() }()

	select {
	case <-ctx.Done():
	case err := <-errc:
		return err
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(baseCtx, 3*time.Second)
	defer shutdownCancel()

	err := server.Shutdown(shutdownCtx)
	<-errc

	return err
}
