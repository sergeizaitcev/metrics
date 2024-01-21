package agent

import (
	"context"
	"crypto/rsa"
	"os"
	"time"

	"github.com/sergeizaitcev/metrics/internal/configs"
	"github.com/sergeizaitcev/metrics/pkg/logging"
	"github.com/sergeizaitcev/metrics/pkg/rsautil"
)

// Run инициализирует агент сбора метрик и запускает его.
func Run(ctx context.Context, c *configs.Agent) (err error) {
	var key *rsa.PublicKey
	if c.PublicKeyPath != "" {
		key, err = rsautil.Public(c.PublicKeyPath)
		if err != nil {
			return err
		}
	}
	opts := &AgentOpts{
		Logger:  logging.New(os.Stdout, c.Level),
		Timeout: 5 * time.Second,
		Key:     key,
	}
	agent := New(c, opts)
	return agent.Run(ctx)
}
