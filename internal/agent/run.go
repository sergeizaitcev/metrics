package agent

import (
	"context"
	"os"
	"time"

	"github.com/sergeizaitcev/metrics/internal/configs"
	"github.com/sergeizaitcev/metrics/pkg/logging"
)

// Run инициализирует агент сбора метрик и запускает его.
func Run(ctx context.Context, c *configs.Agent) error {
	opts := &AgentOpts{
		Logger:  logging.New(os.Stdout, c.Level),
		Timeout: 5 * time.Second,
	}
	agent := New(c, opts)
	return agent.Run(ctx)
}
