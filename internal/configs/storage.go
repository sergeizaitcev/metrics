package configs

import (
	"errors"
	"time"
)

var DefaultStorage = &Storage{
	DatabaseDSN:     "",
	FileStoragePath: "/tmp/metrics-db.wal",
	StoreInterval:   300 * time.Second,
	Restore:         true,
}

// Storage определяет конфиг для хранилища метрик.
type Storage struct {
	// Строка подключения к postgres.
	DatabaseDSN string `env:"DATABASE_DSN"`

	// Путь к файлу с метриками.
	//
	// По умолчанию "/tmp/metrics-db.wal".
	FileStoragePath string `env:"FILE_STORAGE_PATH"`

	// Интервал сохранения данных на диск.
	//
	// По умолчанию 300s.
	StoreInterval time.Duration `env:"STORE_INTERVAL"`

	// Индикатор восстановления данных с диска.
	//
	// По умолчанию true.
	Restore bool `env:"RESTORE"`
}

func (s *Storage) Clone() *Storage {
	s2 := *s
	return &s2
}

func (s *Storage) validate() error {
	if s.StoreInterval < 0 {
		return errors.New("store interval must be is greater than or equal to zero")
	}
	return nil
}
