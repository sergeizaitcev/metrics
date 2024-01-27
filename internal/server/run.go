package server

import (
	"context"
	"crypto/rsa"
	"os"

	"github.com/sergeizaitcev/metrics/internal/configs"
	"github.com/sergeizaitcev/metrics/pkg/logging"
	"github.com/sergeizaitcev/metrics/pkg/rsautil"
)

// Run инициализирует сервер сбора метрик и запускает его.
func Run(ctx context.Context, c *configs.Server) (err error) {
	var key *rsa.PrivateKey
	if c.PrivateKeyPath != "" {
		key, err = rsautil.PrivateKeyFrom(c.PrivateKeyPath)
		if err != nil {
			return err
		}
	}
	opts := &ServerOpts{
		Logger: logging.New(os.Stdout, c.Level),
		Key:    key,
	}
	server := New(c, opts)
	return server.Run(ctx)
}
