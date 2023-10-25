package storage

import (
	"context"

	"github.com/sergeizaitcev/metrics/internal/metrics"
)

// Storager представляет интерфейс хранилища метрик.
type Storager interface {
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
