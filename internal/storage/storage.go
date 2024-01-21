package storage

import (
	"context"
	"time"

	"github.com/sergeizaitcev/metrics/internal/configs"
	"github.com/sergeizaitcev/metrics/internal/metrics"
)

// Storage представляет интерфейс хранилища метрик.
type Storage interface {
	// Ping возвращает ошибку, если не удалось выполнить пинг к хранилищу.
	Ping(context.Context) error

	// Close закрывает хранилище.
	Close() error

	// Save сохраняет значения метрик и возвращает актуальные значения.
	Save(context.Context, ...metrics.Metric) ([]metrics.Metric, error)

	// Get возвращает метрику name.
	Get(context.Context, string) (metrics.Metric, error)

	// GetAll возвращает все метрики.
	GetAll(context.Context) ([]metrics.Metric, error)
}

// NewStorage возвращает новый экземпляр хранилища метрик.
func NewStorage(config *configs.Server) (Storage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if config.DatabaseDSN != "" {
		return initPostgres(ctx, config)
	}

	return initLocal(ctx, config)
}

func initPostgres(ctx context.Context, config *configs.Server) (*Postgres, error) {
	s, err := NewPostgres(config.DatabaseDSN)
	if err != nil {
		return nil, err
	}

	err = s.Ping(ctx)
	if err != nil {
		return nil, err
	}

	err = s.MigrateUp(ctx)
	if err != nil {
		return nil, err
	}

	return s, nil
}

func initLocal(ctx context.Context, config *configs.Server) (*Local, error) {
	opts := &LocalOpts{
		StoreInterval: config.StoreInterval,
		Restore:       config.Restore,
	}

	s, err := NewLocal(config.FileStoragePath, opts)
	if err != nil {
		return nil, err
	}

	err = s.Ping(ctx)
	if err != nil {
		return nil, err
	}

	return s, nil
}
