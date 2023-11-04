package main

import (
	"compress/flate"
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sergeizaitcev/metrics/internal/handlers"
	"github.com/sergeizaitcev/metrics/internal/storage"
	"github.com/sergeizaitcev/metrics/internal/storage/local"
	"github.com/sergeizaitcev/metrics/internal/storage/postgres"
	"github.com/sergeizaitcev/metrics/pkg/logging"
	"github.com/sergeizaitcev/metrics/pkg/middleware"
	"github.com/sergeizaitcev/metrics/pkg/server"
	"github.com/sergeizaitcev/metrics/pkg/sign"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	if err := parseFlags(); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}

	storage, err := newStorage()
	if err != nil {
		return fmt.Errorf("creating a storage: %w", err)
	}
	defer storage.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	err = storage.Ping(ctx)
	if err != nil {
		return fmt.Errorf("ping to a storage: %w", err)
	}

	// NOTE: если выбран postgres в качестве хранилища метрик,
	// то накатывается миграция.
	if pg, ok := storage.(*postgres.Storage); ok {
		err = pg.MigrateUp(ctx)
		if err != nil {
			return fmt.Errorf("failed migration: %w", err)
		}
	}

	lis, err := net.Listen("tcp", flagAddress)
	if err != nil {
		return fmt.Errorf("listening to %s: %w", flagAddress, err)
	}
	defer lis.Close()

	opts := &server.ServerOpts{
		Listener: lis,
	}

	handler := handlers.New(storage, newMiddlewares()...)
	s := server.New(handler, opts)

	return s.ListenAndServe(ctx)
}

func newStorage() (storage.Storager, error) {
	if flagDatabaseDSN != "" {
		return postgres.New(flagDatabaseDSN)
	}

	opts := &local.StorageOpts{
		StoreInterval: time.Duration(flagStoreInterval) * time.Second,
		Restore:       flagRestore,
	}

	return local.New(flagFileStoragePath, opts)
}

func newMiddlewares() []middleware.Middleware {
	middlewares := make([]middleware.Middleware, 0, 3)

	if flagSHA256Key != "" {
		signer := sign.Signer(flagSHA256Key)
		middlewares = append(middlewares, middleware.Sign(signer))
	}

	middlewares = append(
		middlewares,
		middleware.Gzip(flate.BestCompression, "application/json", "text/html"),
	)

	logger := logging.New(os.Stdout, logging.LevelInfo)

	paramsFunc := func(p *middleware.Params) {
		if p.Error != nil {
			logger.Log(logging.LevelError, p.Error.Error(),
				"method", p.Method,
				"uri", p.URI,
				"status_code", p.StatusCode,
			)
			return
		}
		logger.Log(logging.LevelInfo, "",
			"method", p.Method,
			"uri", p.URI,
			"status_code", p.StatusCode,
			"duration", p.Duration,
			"size", len(p.Body),
		)
	}
	middlewares = append(middlewares, middleware.Trace(paramsFunc))

	return middlewares
}
