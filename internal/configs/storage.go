package configs

import (
	"errors"
	"flag"
	"time"

	"github.com/sergeizaitcev/metrics/pkg/commands"
)

var DefaultStorage = &Storage{
	DatabaseDSN:     "",
	FileStoragePath: "/tmp/metrics-db.wal",
	StoreInterval:   300 * time.Second,
	Restore:         true,
}

var _ commands.Config = (*Storage)(nil)

// Storage определяет конфиг для хранилища метрик.
type Storage struct {
	commands.UnimplementedConfig

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

func (s *Storage) SetFlags(fs *flag.FlagSet) {
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

func (s *Storage) Validate() error {
	if s.storeInterval != nil {
		s.StoreInterval = duration(*s.storeInterval)
	}
	if s.StoreInterval < 0 {
		return errors.New("store interval must be is greater than or equal to zero")
	}
	return nil
}
