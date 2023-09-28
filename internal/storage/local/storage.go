package local

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/sergeizaitcev/metrics/internal/metrics"
)

// Storage определяет локальное храналище метрик.
type Storage struct {
	mu      sync.RWMutex
	metrics map[string]metrics.Metric
}

// NewStorage возвращает новый экземпляр локального хранилища метрик.
func NewStorage() *Storage {
	return &Storage{
		metrics: make(map[string]metrics.Metric),
	}
}

// Set устанавливает новое значение метрики и возвращает предыдущее.
func (s *Storage) Set(ctx context.Context, value metrics.Metric) (metrics.Metric, error) {
	if value.Kind() == metrics.KindUnknown {
		return metrics.Metric{}, errors.New("local: unknown metric kind")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	oldValue, ok := s.metrics[value.Name()]
	if ok && oldValue.Kind() != value.Kind() {
		return metrics.Metric{}, fmt.Errorf("local: expected metric kind %s, got %s",
			oldValue.Kind(), value.Kind(),
		)
	}

	s.metrics[value.Name()] = value

	return oldValue, nil
}

// Add увеличивает значение метрики и возвращает итоговый результат.
func (s *Storage) Add(ctx context.Context, value metrics.Metric) (metrics.Metric, error) {
	if value.Kind() == metrics.KindUnknown {
		return metrics.Metric{}, errors.New("local: unknown metric kind")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	oldValue, ok := s.metrics[value.Name()]
	if !ok {
		s.metrics[value.Name()] = value
		return value, nil
	}
	if oldValue.Kind() != value.Kind() {
		return metrics.Metric{}, fmt.Errorf("local: expected metric kind %s, got %s",
			oldValue.Kind(), value.Kind(),
		)
	}

	switch value.Kind() {
	case metrics.KindCounter:
		value = metrics.Counter(value.Name(), oldValue.Int64()+value.Int64())
	case metrics.KindGauge:
		value = metrics.Gauge(value.Name(), oldValue.Float64()+value.Float64())
	}

	s.metrics[value.Name()] = value

	return value, nil
}

// Get возвращает метрику.
func (s *Storage) Get(ctx context.Context, name string) (metrics.Metric, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	metric := s.metrics[name]

	return metric, nil
}

// GetAll возвращает все метрики.
func (s *Storage) GetAll(ctx context.Context) ([]metrics.Metric, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	values := make([]metrics.Metric, 0, len(s.metrics))
	for _, metric := range s.metrics {
		values = append(values, metric)
	}

	sort.SliceStable(values, func(i, j int) bool {
		return values[i].Name() < values[j].Name()
	})

	return values, nil
}

// Del удаляет метрику.
func (s *Storage) Del(ctx context.Context, name string) (metrics.Metric, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	oldValue := s.metrics[name]
	delete(s.metrics, name)

	return oldValue, nil
}
