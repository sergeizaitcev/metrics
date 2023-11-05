package server

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/sergeizaitcev/metrics/internal/configs"
	"github.com/sergeizaitcev/metrics/pkg/logging"
)

// Run инициализирует сервер сбора метрик и запускает его.
func Run() error {
	config, err := configs.ParseServer()
	if err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	opts := &ServerOpts{
		Logger: logging.New(os.Stdout, config.Level),
	}

	server, err := New(config, opts)
	if err != nil {
		return fmt.Errorf("create server: %w", err)
	}
	defer server.Close()

	return server.Run(ctx)
}
