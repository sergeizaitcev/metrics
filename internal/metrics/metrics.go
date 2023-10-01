package metrics

import (
	"context"
	"errors"
	"fmt"
)

// ErrNotFound возвращается, когда не найдена метрика.
var ErrNotFound = errors.New("metrics: not found")

// Storager представляет интерфейс хранилища метрик.
type Storager interface {
	// Set устанавливает новое значение метрики и возвращает предыдущее.
	Set(context.Context, Metric) (Metric, error)

	// Add увеличивает значение метрики и возвращает итоговый результат.
	Add(context.Context, Metric) (Metric, error)

	// Get возвращает метрику.
	Get(context.Context, string) (Metric, error)

	// GetAll возвращает все метрики.
	GetAll(context.Context) ([]Metric, error)
}

// Metrics определяет сервис для работы с метриками.
type Metrics struct {
	storage Storager
}

// NewService возвращает новый экземпляр metrics.
func NewMetrics(s Storager) *Metrics {
	return &Metrics{storage: s}
}

// Save сохраняет метрику.
func (m *Metrics) Save(ctx context.Context, metric Metric) (Metric, error) {
	var (
		actual Metric
		err    error
	)

	switch metric.Kind() {
	case KindCounter:
		actual, err = m.storage.Add(ctx, metric)
		if err != nil {
			return Metric{}, fmt.Errorf("metrics: adding a counter: %w", err)
		}
	case KindGauge:
		actual, err = m.storage.Set(ctx, metric)
		if err != nil {
			return Metric{}, fmt.Errorf("metrics: setting a gauge: %w", err)
		}
	default:
		return Metric{}, fmt.Errorf("metrics: unsupported kind: %s", metric.Kind())
	}

	return actual, nil
}

// Lookup выполняет поиск метрики по её имени.
func (m *Metrics) Lookup(ctx context.Context, name string) (Metric, error) {
	metric, err := m.storage.Get(ctx, name)
	if err != nil {
		return Metric{}, fmt.Errorf("metrics: lookup for a metric by name: %w", err)
	}
	if metric.Kind() == KindUnknown {
		return Metric{}, ErrNotFound
	}
	return metric, nil
}

// All возвращает все метрики.
func (m *Metrics) All(ctx context.Context) ([]Metric, error) {
	metrics, err := m.storage.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("metrics: getting all metrics: %w", err)
	}
	return metrics, nil
}
