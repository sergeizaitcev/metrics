package server

import (
	"context"
	"os"

	"github.com/sergeizaitcev/metrics/internal/configs"
	"github.com/sergeizaitcev/metrics/pkg/logging"
)

// Run инициализирует сервер сбора метрик и запускает его.
func Run(ctx context.Context, c *configs.Server) error {
	opts := &ServerOpts{
		Logger: logging.New(os.Stdout, c.Level),
	}
	server := New(c, opts)
	return server.Run(ctx)
}
