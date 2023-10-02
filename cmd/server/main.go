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

	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"

	"github.com/sergeizaitcev/metrics/internal/metrics"
	"github.com/sergeizaitcev/metrics/internal/storage/file"
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
	baseCtx := context.Background()

	if err := parseFlags(); err != nil {
		return err
	}

	fileStorage, err := newFileStorage()
	if err != nil {
		return err
	}
	defer fileStorage.Close()

	var values []metrics.Metric
	if flagRestore {
		values, err = fileStorage.ReadAll()
		if err != nil {
			return err
		}
	}

	storage := local.NewStorage(values...)
	metrics := metrics.NewMetrics(storage, fileStorage)

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

	router := newRouter(
		metrics,
		middleware.Gzip(flate.BestCompression, "application/json", "text/html"),
		middleware.Trace(paramsFunc),
	)

	ctx, cancel := signal.NotifyContext(baseCtx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	errg, errgCtx := errgroup.WithContext(ctx)
	errg.Go(func() error { return listenAndServe(errgCtx, router) })

	if flagStoreInterval > 0 {
		errg.Go(func() error {
			ticker := time.NewTicker(flagStoreInterval.Duration())
			defer ticker.Stop()

			for {
				select {
				case <-errgCtx.Done():
					return nil
				case <-ticker.C:
					err := fileStorage.Flush()
					if err != nil {
						return err
					}
				}
			}
		})
	}

	return errg.Wait()
}

func newFileStorage() (*file.Storage, error) {
	if flagStoreInterval == 0 {
		return file.OpenSync(flagFileStoragePath)
	}
	return file.Open(flagFileStoragePath)
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
