package configs

import (
	"errors"
	"flag"

	"github.com/sergeizaitcev/metrics/pkg/commands"
	"github.com/sergeizaitcev/metrics/pkg/logging"
)

var DefaultServer = &Server{
	Level:          logging.LevelInfo,
	Address:        "localhost:8080",
	SHA256Key:      "",
	PrivateKeyPath: "server.rsa",
	Storage:        DefaultStorage,
}

var _ commands.Config = (*Server)(nil)

// Server определяет конфиг для сервера.
type Server struct {
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

	// Приватный ключ для асиметричного шифрования.
	PrivateKeyPath string `env:"PRIVATE_KEY_PATH" json:"private_key_path"`

	// Конфигурация хранилища метрик.
	*Storage
}

func (s *Server) Validate() error {
	if s.Address == "" {
		return errors.New("address must be not empty")
	}
	if s.PrivateKeyPath == "" {
		return errors.New("private key path must be not empty")
	}
	return s.Storage.Validate()
}

func (s *Server) SetFlags(fs *flag.FlagSet) {
	fs.TextVar(&s.Level, "v", DefaultServer.Level, "logging level")
	fs.StringVar(&s.Address, "a", DefaultServer.Address, "server address")
	fs.StringVar(&s.SHA256Key, "k", DefaultServer.SHA256Key, "secret sha256 key")
	fs.StringVar(
		&s.PrivateKeyPath,
		"private-key",
		DefaultServer.PrivateKeyPath,
		"path to private key",
	)
	s.Storage.SetFlags(fs)
}
