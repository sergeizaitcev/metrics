package metrics

import (
	"encoding/json"
	"errors"
	"math"
	"strconv"
	"strings"
)

// Kind определяет тип метрики.
type Kind uint8

const (
	KindUnknown Kind = iota

	KindCounter
	KindGauge
)

var kindValues = map[Kind]string{
	KindUnknown: "unknown",
	KindCounter: "counter",
	KindGauge:   "gauge",
}

func (k Kind) String() string {
	v, ok := kindValues[k]
	if !ok {
		return kindValues[KindUnknown]
	}
	return v
}

// ParseKind парсит строку и возвращает тип метрики.
func ParseKind(s string) Kind {
	if s == "" {
		return KindUnknown
	}
	s0 := strings.ToLower(s)
	for k, v := range kindValues {
		if v == s0 {
			return k
		}
	}
	return KindUnknown
}

// value определяет значение метрики.
type value uint64

// counterValue возвращает значение метрики типа счётчик.
func counterValue(v int64) value {
	return value(v)
}

// gaugeValue возвращает значение метрики типа датчик.
func gaugeValue(v float64) value {
	return value(math.Float64bits(v))
}

// Int64 возвращает значение метрики как int64.
func (v value) Int64() int64 {
	return int64(v)
}

// Float64 возвращает значение метрики как float64.
func (v value) Float64() float64 {
	return math.Float64frombits(uint64(v))
}

var (
	_ json.Marshaler   = (*Metric)(nil)
	_ json.Unmarshaler = (*Metric)(nil)
)

// Metric определяет метрику.
type Metric struct {
	kind  Kind
	name  string
	value value
}

// Counter возращает метрику типа счётчик с именем name и значением value.
func Counter(name string, value int64) Metric {
	return Metric{
		kind:  KindCounter,
		name:  name,
		value: counterValue(value),
	}
}

// Gauge возращает метрику типа датчик с именем name и значением value.
func Gauge(name string, value float64) Metric {
	return Metric{
		kind:  KindGauge,
		name:  name,
		value: gaugeValue(value),
	}
}

// Kind возвращает тип метрики.
func (m *Metric) Kind() Kind {
	return m.kind
}

// Name возвращает наименование метрики.
func (m *Metric) Name() string {
	return m.name
}

// String возвращает строковое представление значения метрики.
func (m *Metric) String() string {
	switch m.kind {
	case KindCounter:
		return strconv.FormatInt(m.value.Int64(), 10)
	case KindGauge:
		return strconv.FormatFloat(m.value.Float64(), 'f', -1, 64)
	}
	return "<Unknown>"
}

// Int64 возвращает значение метрики как int64. Паникует, если тип метрики
// не является KindCounter.
func (m *Metric) Int64() int64 {
	return m.value.Int64()
}

// Float64 возвращает значение метрики как float64. Паникует, если тип метрики
// не является KindGauge.
func (m *Metric) Float64() float64 {
	return m.value.Float64()
}

// Equal возвращает true, если метрика равна x.
func (m *Metric) Equal(x Metric) bool {
	return m.kind == x.kind && m.name == x.name && m.value == x.value
}

// IsEmpty возвращает true, если метрика пуста.
func (m *Metric) IsEmpty() bool {
	return m.Equal(Metric{})
}

type metric struct {
	Kind  string   `json:"type"`            // тип метрики.
	ID    string   `json:"id"`              // имя метрики.
	Delta *int64   `json:"delta,omitempty"` // значение метрики counter.
	Value *float64 `json:"value,omitempty"` // значение метрики gauge.
}

func (m *Metric) MarshalJSON() ([]byte, error) {
	if m.Kind() == KindUnknown {
		return []byte("{}"), nil
	}

	obj := metric{
		Kind: m.Kind().String(),
		ID:   m.Name(),
	}

	switch m.Kind() {
	case KindCounter:
		v := m.Int64()
		obj.Delta = &v
	case KindGauge:
		v := m.Float64()
		obj.Value = &v
	}

	return json.Marshal(&obj)
}

func (m *Metric) UnmarshalJSON(data []byte) error {
	if string(data) == "{}" {
		*m = Metric{}
		return nil
	}

	var obj metric

	err := json.Unmarshal(data, &obj)
	if err != nil {
		return err
	}

	if obj.Kind == "" {
		return errors.New("metrics: the metric type should not be empty")
	}
	if obj.ID == "" {
		return errors.New("metrics: the metric id should not be empty")
	}

	switch ParseKind(obj.Kind) {
	case KindCounter:
		var v int64
		if obj.Delta != nil {
			v = *obj.Delta
		}
		*m = Counter(obj.ID, v)
	case KindGauge:
		var v float64
		if obj.Value != nil {
			v = *obj.Value
		}
		*m = Gauge(obj.ID, v)
	case KindUnknown:
		return errors.New("metrics: the metric type is unknown")
	}

	return nil
}
