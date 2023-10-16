package main

import (
	"compress/flate"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"

	"github.com/sergeizaitcev/metrics/internal/handlers"
	"github.com/sergeizaitcev/metrics/internal/storage/local"
	"github.com/sergeizaitcev/metrics/pkg/middleware"
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

	baseCtx := context.Background()

	ctx, cancel := signal.NotifyContext(baseCtx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	conn, err := pgx.Connect(ctx, flagDatabaseDSN)
	if err != nil {
		return err
	}
	defer conn.Close(baseCtx)

	storage, err := newStorage()
	if err != nil {
		return err
	}
	defer storage.Close()

	logger := zerolog.New(os.Stdout)
	paramsFunc := func(p *middleware.Params) {
		entry := logger.Info()
		entry.Str("method", p.Method)
		entry.Str("uri", p.URI)
		entry.Int("statusCode", p.StatusCode)
		entry.Dur("duration", p.Duration)
		entry.Int("size", len(p.Body))
		entry.Send()
	}

	handler := handlers.New(
		storage,
		middleware.Gzip(flate.BestCompression, "application/json", "text/html"),
		middleware.Trace(paramsFunc),
	)

	errg, errgCtx := errgroup.WithContext(ctx)
	errg.Go(func() error {
		return listenAndServe(errgCtx, handler)
	})

	if flagStoreInterval > 0 {
		errg.Go(func() error {
			return performEachTick(errgCtx, storage.Flush)
		})
	}

	return errg.Wait()
}

type storager interface {
	handlers.Storager

	Flush() error
	Close() error
}

func newStorage() (storager, error) {
	opts := &local.StorageOpts{
		Synced: flagStoreInterval == 0,
	}

	if flagRestore {
		return local.Open(flagFileStoragePath, opts)
	}

	return local.New(flagFileStoragePath, opts)
}

func performEachTick(ctx context.Context, f func() error) error {
	d := time.Duration(flagStoreInterval) * time.Second

	ticker := time.NewTicker(d)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := f(); err != nil {
				return err
			}
		}
	}
}

func listenAndServe(ctx context.Context, handler http.Handler) error {
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

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer shutdownCancel()

	err := server.Shutdown(shutdownCtx)
	<-errc

	return err
}
