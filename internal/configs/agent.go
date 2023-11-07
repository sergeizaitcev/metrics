package configs

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/caarlos0/env/v10"

	"github.com/sergeizaitcev/metrics/pkg/logging"
)

var DefaultAgent = &Agent{
	Level:          logging.LevelInfo,
	Address:        "localhost:8080",
	SHA256Key:      "",
	PollInterval:   10 * time.Second,
	ReportInterval: 2 * time.Second,
	RateLimit:      1,
}

// Agent определяет конфиг для агента.
type Agent struct {
	// Уровень логирования.
	//
	// По умолчанию "info".
	Level logging.Level

	// Адрес сервера.
	//
	// По умолчанию "localhost:8080".
	Address string `env:"ADDRESS"`

	// Ключ подписи данных. Если ключ пуст, то данные не подписываются.
	SHA256Key string `env:"KEY"`

	// Интервал отправки метрик на сервер.
	//
	// По умолчанию 10s.
	PollInterval time.Duration `env:"POLL_INTERVAL"`

	// Интервал сбора метрик.
	//
	// По умолчанию 2s.
	ReportInterval time.Duration `env:"REPORT_INTERVAL"`

	// Количество одновременных запросов на сервер.
	//
	// По умолчанию 1.
	RateLimit int `env:"RATE_LIMIT"`
}

func (a *Agent) Clone() *Agent {
	a2 := *a
	return &a2
}

func (a *Agent) validate() error {
	if a.Address == "" {
		return errors.New("address must be not empty")
	}
	if a.PollInterval <= 0 {
		return errors.New("poll interval must be is greater than zero")
	}
	if a.ReportInterval <= 0 {
		return errors.New("report interval must be is greater than zero")
	}
	if a.RateLimit < 1 {
		return errors.New("rate limit must be is greater than zero")
	}
	return nil
}

// ParseAgent возвращает новый конфиг для агента.
func ParseAgent() (*Agent, error) {
	agent := parseAgentFlags()

	opts := env.Options{
		FuncMap: customParsers,
	}

	err := env.ParseWithOptions(agent, opts)
	if err != nil {
		return nil, fmt.Errorf("parse env: %w", err)
	}

	err = agent.validate()
	if err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}

	return agent, nil
}

func parseAgentFlags() *Agent {
	a := DefaultAgent.Clone()

	flags := flag.NewFlagSet("agent", flag.ExitOnError)

	flags.TextVar(&a.Level, "v", DefaultAgent.Level, "logging level")
	flags.StringVar(&a.Address, "a", DefaultAgent.Address, "server address")
	flags.StringVar(&a.SHA256Key, "k", DefaultAgent.SHA256Key, "secret sha256 key")
	flags.IntVar(&a.RateLimit, "l", DefaultAgent.RateLimit, "rate limit")

	flagPollInterval := flags.Int64(
		"p",
		second(DefaultAgent.PollInterval),
		"poll interval in seconds",
	)
	flagReportInterval := flags.Int64(
		"r",
		second(DefaultAgent.ReportInterval),
		"report interval in seconds",
	)

	err := flags.Parse(os.Args[1:])
	if err != nil {
		flags.Usage()
	}

	if flagPollInterval != nil {
		a.PollInterval = duration(*flagPollInterval)
	}
	if flagReportInterval != nil {
		a.ReportInterval = duration(*flagReportInterval)
	}

	return a
}
