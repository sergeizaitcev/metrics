package metrics

import (
	"encoding/json"
	"errors"
	"fmt"
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

var kindNames = []Kind{
	KindUnknown,
	KindCounter,
	KindGauge,
}

var kindValues = []string{
	"Unknown",
	"Counter",
	"Gauge",
}

func (k Kind) String() string {
	if int(k) < len(kindValues) {
		return kindValues[k]
	}
	return kindValues[0]
}

// ParseKind парсит строку с типом метрики.
func ParseKind(s string) Kind {
	if s == "" {
		return KindUnknown
	}
	s0 := strings.ToLower(s)
	for i := range kindValues {
		v := strings.ToLower(kindValues[i])
		if v == s0 {
			return kindNames[i]
		}
	}
	return KindUnknown
}

// kindValue определяет тип значения метрики.
type kindValue uint8

const (
	kindValueUnknown kindValue = iota

	kindValueInt64
	kindValueFloat64
)

var kindValueValues = []string{
	"Unknown",
	"Int64",
	"Float64",
}

func (k kindValue) String() string {
	if int(k) < len(kindValueValues) {
		return kindValueValues[k]
	}
	return kindValueValues[0]
}

// value определяет значение метрики.
type value struct {
	kind kindValue
	num  uint64
}

// Equal возвращает true, если значение метрики равно x.
func (v value) Equal(x value) bool {
	return v.kind == x.kind && v.num == x.num
}

// counterValue возвращает значение метрики типа счётчик.
func counterValue(v int64) value {
	return value{kind: kindValueInt64, num: uint64(v)}
}

// gaugeValue возвращает значение метрики типа датчик.
func gaugeValue(v float64) value {
	return value{kind: kindValueFloat64, num: math.Float64bits(v)}
}

// String возвращает строковое представление значения метрики.
func (v value) String() string {
	switch v.kind {
	case kindValueInt64:
		return strconv.FormatInt(int64(v.num), 10)
	case kindValueFloat64:
		return strconv.FormatFloat(v.float(), 'f', -1, 64)
	default:
		return ""
	}
}

// Int64 возвращает значение метрики как int64.
func (v value) Int64() int64 {
	if got, want := v.kind, kindValueInt64; got != want {
		panic(fmt.Sprintf("metrics: expected kind of value %s, got %s", want, got))
	}
	return int64(v.num)
}

// Float64 возвращает значение метрики как float64.
func (v value) Float64() float64 {
	if got, want := v.kind, kindValueFloat64; got != want {
		panic(fmt.Sprintf("metrics: expected kind of value %s, got %s", want, got))
	}
	return v.float()
}

func (v value) float() float64 {
	return math.Float64frombits(v.num)
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
	return m.value.String()
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
	return m.kind == x.kind && m.name == x.name && m.value.Equal(x.value)
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
		Kind: strings.ToLower(m.Kind().String()),
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

	kind := ParseKind(obj.Kind)
	if kind == KindUnknown {
		return errors.New("metrics: the metric type is unknown")
	}

	switch kind {
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
	}

	return nil
}
