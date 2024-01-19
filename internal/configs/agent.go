package configs

import (
	"errors"
	"flag"
	"time"

	"github.com/sergeizaitcev/metrics/pkg/commands"
	"github.com/sergeizaitcev/metrics/pkg/logging"
)

var DefaultAgent = &Agent{
	Level:          logging.LevelInfo,
	Address:        "localhost:8080",
	SHA256Key:      "",
	PublicKeyPath:  "server.rsa.pub",
	PollInterval:   10 * time.Second,
	ReportInterval: 2 * time.Second,
	RateLimit:      1,
}

var _ commands.Config = (*Agent)(nil)

// Agent определяет конфиг для агента.
type Agent struct {
	commands.UnimplementedConfig

	// Уровень логирования.
	//
	// По умолчанию "info".
	Level logging.Level `env:"LEVEL" json:"level"`

	// Адрес сервера.
	//
	// По умолчанию "localhost:8080".
	Address string `env:"ADDRESS" json:"address"`

	// Ключ подписи данных. Если ключ пуст, то данные не подписываются.
	SHA256Key string `env:"KEY" json:"key"`

	// Открытый ключ для асиметричного шифрования.
	PublicKeyPath string `env:"PUBLIC_KEY_PATH" json:"public_key_path"`

	// Интервал отправки метрик на сервер.
	//
	// По умолчанию 10s.
	PollInterval time.Duration `env:"POLL_INTERVAL" json:"poll_interval"`

	// Интервал сбора метрик.
	//
	// По умолчанию 2s.
	ReportInterval time.Duration `env:"REPORT_INTERVAL" json:"report_interval"`

	// Количество одновременных запросов на сервер.
	//
	// По умолчанию 1.
	RateLimit int `env:"RATE_LIMIT" json:"rate_limit"`

	pollInternval, reportInterval *int64
}

func (a *Agent) SetFlags(fs *flag.FlagSet) {
	fs.TextVar(&a.Level, "v", DefaultAgent.Level, "logging level")
	fs.StringVar(&a.Address, "a", DefaultAgent.Address, "server address")
	fs.StringVar(&a.SHA256Key, "k", DefaultAgent.SHA256Key, "secret sha256 key")
	fs.StringVar(
		&a.PublicKeyPath,
		"public-key",
		DefaultAgent.PublicKeyPath,
		"path to public key",
	)
	fs.IntVar(&a.RateLimit, "l", DefaultAgent.RateLimit, "rate limit")

	a.pollInternval = fs.Int64(
		"p",
		second(DefaultAgent.PollInterval),
		"poll interval in seconds",
	)
	a.reportInterval = fs.Int64(
		"r",
		second(DefaultAgent.ReportInterval),
		"report interval in seconds",
	)
}

func (a *Agent) Validate() error {
	if a.pollInternval != nil {
		a.PollInterval = duration(*a.pollInternval)
	}
	if a.reportInterval != nil {
		a.ReportInterval = duration(*a.reportInterval)
	}
	if a.Address == "" {
		return errors.New("address must be not empty")
	}
	if a.PublicKeyPath == "" {
		return errors.New("public key must be not empty")
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
