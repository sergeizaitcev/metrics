package local

import (
	"fmt"

	"github.com/sergeizaitcev/metrics/internal/metrics"
)

// memstorage определяет храналище метрик в памяти.
type memstorage map[string]metrics.Metric

// conflict возвращает ошибку, если метрика конфликтует с уже записанными
// метриками.
func (s memstorage) conflict(value metrics.Metric) error {
	actual, ok := s[value.Name()]
	if ok && actual.Kind() != value.Kind() {
		return fmt.Errorf("expected to get a metric kind %s, got %s",
			actual.Kind(), value.Kind(),
		)
	}
	return nil
}

// add увеличивает значение метрики и возвращает актуальное значение.
func (s memstorage) add(value metrics.Metric) metrics.Metric {
	oldValue, ok := s[value.Name()]
	if !ok {
		s[value.Name()] = value
		return value
	}

	value = metrics.Counter(value.Name(), value.Int64()+oldValue.Int64())
	s[value.Name()] = value

	return value
}

// set устанавливает новое значение метрики и возвращает предыдущее.
func (s memstorage) set(value metrics.Metric) metrics.Metric {
	oldValue := s[value.Name()]
	s[value.Name()] = value
	return oldValue
}

// get возвращает метрику.
func (s memstorage) get(name string) metrics.Metric {
	return s[name]
}

// getAll возвращает все метрики.
func (s memstorage) getAll() []metrics.Metric {
	values := make([]metrics.Metric, 0, len(s))
	for _, metric := range s {
		values = append(values, metric)
	}
	return values
}
