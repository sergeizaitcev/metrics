package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sergeizaitcev/metrics/internal/agent"
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

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	opts := &agent.AgentOpts{
		SHA256Key:      flagSHA256Key,
		ReportInterval: time.Duration(flagReportInterval) * time.Second,
		PollInterval:   time.Duration(flagPollInterval) * time.Second,
		RateLimit:      flagRateLimit,
	}

	a := agent.New(flagAddress, opts)

	err := a.Run(ctx)
	if err != nil {
		return fmt.Errorf("collecting and sending metrics to the server: %w", err)
	}

	return nil
}
