package metrics

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unsafe"
)

// Storager представляет интерфейс хранилища метрик.
type Storager interface {
	// Ping выполняет пинг к хранилищу.
	Ping(context.Context) error

	// Close закрывает хранилище.
	Close() error

	// Add увеличивает значение метрики и возвращает итоговый результат.
	Add(context.Context, Metric) (Metric, error)

	// Set устанавливает новое значение метрики и возвращает предыдущее.
	Set(context.Context, Metric) (Metric, error)

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

// String возвращает строковое представление метрики.
func (m *Metric) String() string {
	return fmt.Sprintf(
		"metric{kind=%s name=%s value=%s}",
		m.kind.String(),
		m.name,
		m.Str(),
	)
}

// Str возвращает значение метрики как string.
func (m *Metric) Str() string {
	switch m.kind {
	case KindCounter:
		return strconv.FormatInt(m.value.Int64(), 10)
	case KindGauge:
		return strconv.FormatFloat(m.value.Float64(), 'f', -1, 64)
	}
	return "<unknown>"
}

// Int64 возвращает значение метрики как int64.
func (m *Metric) Int64() int64 {
	return m.value.Int64()
}

// Float64 возвращает значение метрики как float64.
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
	if m.kind == KindUnknown {
		return []byte("{}"), nil
	}

	obj := metric{
		Kind: m.kind.String(),
		ID:   m.name,
	}

	switch m.kind {
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

func (m *Metric) MarshalBinary() ([]byte, error) {
	enc := base64.RawStdEncoding
	encodedLen := enc.EncodedLen(len(m.name))

	var buf [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(buf[:], uint64(encodedLen))

	data := make([]byte, 9+n+encodedLen)

	data[0] = byte(m.kind)
	binary.BigEndian.PutUint64(data[1:9], uint64(m.value))

	copy(data[9:], buf[:n])

	src := unsafe.Slice(unsafe.StringData(m.name), len(m.name))
	enc.Encode(data[9+n:], src)

	return data, nil
}

func (m *Metric) UnmarshalBinary(data []byte) error {
	if len(data) < 10 {
		return errors.New("metrics: data too small")
	}

	kind := Kind(data[0])
	data = data[1:]

	value := value(binary.BigEndian.Uint64(data[:8]))
	data = data[8:]

	size, n := binary.Uvarint(data)
	if n <= 0 || uint64(len(data)-n) < size {
		return errors.New("metrics: data is corrupted")
	}

	data = data[n:]
	data = data[:size]

	enc := base64.RawStdEncoding
	name := make([]byte, enc.DecodedLen(int(size)))

	_, err := enc.Decode(name, data)
	if err != nil {
		return fmt.Errorf("metrics: base64 decoding: %w", err)
	}

	*m = Metric{
		kind:  kind,
		name:  unsafe.String(unsafe.SliceData(name), len(name)),
		value: value,
	}

	return nil
}
