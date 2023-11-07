package agent

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sergeizaitcev/metrics/internal/configs"
	"github.com/sergeizaitcev/metrics/pkg/logging"
)

// Run инициализирует агент сбора метрик и запускает его.
func Run() error {
	config, err := configs.ParseAgent()
	if err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	opts := &AgentOpts{
		Logger:  logging.New(os.Stdout, config.Level),
		Timeout: 5 * time.Second,
	}

	agent := New(config, opts)

	return agent.Run(ctx)
}
