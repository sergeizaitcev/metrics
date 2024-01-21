package configs

import (
	"encoding/json"
	"errors"
	"flag"
	"io"
	"time"

	"github.com/sergeizaitcev/metrics/pkg/commands"
	"github.com/sergeizaitcev/metrics/pkg/logging"
)

var DefaultServer = &Server{
	Level:           logging.LevelInfo,
	ConfigPath:      "",
	Address:         "localhost:8080",
	SHA256Key:       "",
	PrivateKeyPath:  "",
	DatabaseDSN:     "",
	FileStoragePath: "/tmp/metrics-db.wal",
	StoreInterval:   300 * time.Second,
	Restore:         true,
}

var _ commands.Config = (*Server)(nil)

// Server определяет конфиг для сервера.
type Server struct {
	commands.UnimplementedConfig

	// Путь к файлу конфигурации.
	ConfigPath commands.ConfigPath `env:"CONFIG" json:"-"`

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

	// Строка подключения к postgres.
	DatabaseDSN string `env:"DATABASE_DSN" json:"database_dsn"`

	// Путь к файлу с метриками.
	//
	// По умолчанию "/tmp/metrics-db.wal".
	FileStoragePath string `env:"FILE_STORAGE_PATH" json:"file_storage_path"`

	// Интервал сохранения данных на диск.
	//
	// По умолчанию 300s.
	StoreInterval time.Duration `env:"STORE_INTERVAL" json:"store_interval"`

	// Индикатор восстановления данных с диска.
	//
	// По умолчанию true.
	Restore bool `env:"RESTORE" json:"restore"`

	storeInterval *int64
}

func (s *Server) ReadFrom(r io.Reader) (int64, error) {
	dec := json.NewDecoder(r)
	err := dec.Decode(s)
	if err != nil {
		return 0, err
	}
	return dec.InputOffset(), nil
}

func (s *Server) Validate() error {
	if s.storeInterval != nil {
		s.StoreInterval = duration(*s.storeInterval)
	}
	if s.Address == "" {
		return errors.New("address must be not empty")
	}
	if s.StoreInterval < 0 {
		return errors.New("store interval must be is greater than or equal to zero")
	}
	return nil
}

func (s *Server) SetFlags(fs *flag.FlagSet) {
	fs.Var(&s.ConfigPath, "c", "path to config")
	fs.TextVar(&s.Level, "v", DefaultServer.Level, "logging level")
	fs.StringVar(&s.Address, "a", DefaultServer.Address, "server address")
	fs.StringVar(&s.SHA256Key, "k", DefaultServer.SHA256Key, "secret sha256 key")
	fs.StringVar(
		&s.PrivateKeyPath,
		"private-key",
		DefaultServer.PrivateKeyPath,
		"path to private key",
	)
	fs.StringVar(&s.DatabaseDSN, "d", DefaultServer.DatabaseDSN, "database dsn")
	fs.StringVar(
		&s.FileStoragePath,
		"f",
		DefaultServer.FileStoragePath,
		"file storage path",
	)
	fs.BoolVar(&s.Restore, "r", DefaultServer.Restore, "restore")
	s.storeInterval = fs.Int64(
		"i",
		second(DefaultServer.StoreInterval),
		"store interval in seconds",
	)
}
