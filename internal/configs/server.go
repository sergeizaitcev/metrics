package configs

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/caarlos0/env/v10"

	"github.com/sergeizaitcev/metrics/pkg/logging"
)

var DefaultServer = &Server{
	Level:     logging.LevelInfo,
	Address:   "localhost:8080",
	SHA256Key: "",
	Storage:   DefaultStorage,
}

// Server определяет конфиг для сервера.
type Server struct {
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

	// Конфигурация хранилища метрик.
	*Storage
}

func (s *Server) Clone() *Server {
	s2 := *s
	s2.Storage = s.Storage.Clone()
	return &s2
}

func (s *Server) validate() error {
	if s.Address == "" {
		return errors.New("address must be not empty")
	}
	return s.Storage.validate()
}

// ParseServer возвращает новый конфиг для сервера.
func ParseServer() (*Server, error) {
	server := parseServerFlags()

	opts := env.Options{
		FuncMap: customParsers,
	}

	err := env.ParseWithOptions(server, opts)
	if err != nil {
		return nil, fmt.Errorf("parse env: %w", err)
	}

	err = server.validate()
	if err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}

	return server, nil
}

func parseServerFlags() *Server {
	s := DefaultServer.Clone()

	flags := flag.NewFlagSet("server", flag.ExitOnError)

	flags.TextVar(&s.Level, "v", DefaultServer.Level, "logging level")
	flags.StringVar(&s.Address, "a", DefaultServer.Address, "server address")
	flags.StringVar(&s.DatabaseDSN, "d", DefaultServer.DatabaseDSN, "database dsn")
	flags.StringVar(
		&s.FileStoragePath,
		"f",
		DefaultServer.FileStoragePath,
		"file storage path",
	)
	flags.StringVar(&s.SHA256Key, "k", DefaultServer.SHA256Key, "secret sha256 key")
	flags.BoolVar(&s.Restore, "r", DefaultServer.Restore, "restore")

	flagStoreInterval := flags.Int64(
		"i",
		second(DefaultServer.StoreInterval),
		"store interval in seconds",
	)

	err := flags.Parse(os.Args[1:])
	if err != nil {
		flags.Usage()
	}

	if flagStoreInterval != nil {
		s.StoreInterval = duration(*flagStoreInterval)
	}

	return s
}
