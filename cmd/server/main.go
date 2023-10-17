package main

import (
	"compress/flate"
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"github.com/rs/zerolog"

	"github.com/sergeizaitcev/metrics/internal/handlers"
	"github.com/sergeizaitcev/metrics/internal/metrics"
	"github.com/sergeizaitcev/metrics/internal/storage/local"
	"github.com/sergeizaitcev/metrics/internal/storage/postgres"
	"github.com/sergeizaitcev/metrics/pkg/middleware"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	baseCtx := context.Background()

	if err := parseFlags(); err != nil {
		return err
	}

	storage, err := newStorage()
	if err != nil {
		return err
	}
	defer storage.Close()

	ctx, cancel := signal.NotifyContext(baseCtx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	err = storage.Ping(ctx)
	if err != nil {
		return err
	}

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

	return listenAndServe(ctx, handler)
}

func newStorage() (metrics.Storager, error) {
	switch {
	case flagDatabaseDSN != "":
		db, err := sql.Open("postgres", flagDatabaseDSN)
		if err != nil {
			return nil, err
		}

		storage := postgres.New(db)

		return storage, storage.Up(context.TODO())
	case flagRestore:
		opts := &local.StorageOpts{
			StoreInterval: time.Duration(flagStoreInterval) * time.Second,
		}
		return local.Open(flagFileStoragePath, opts)
	default:
		opts := &local.StorageOpts{
			StoreInterval: time.Duration(flagStoreInterval) * time.Second,
		}
		return local.New(flagFileStoragePath, opts)
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
