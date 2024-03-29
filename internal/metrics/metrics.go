package metrics

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unsafe"

	pb "github.com/sergeizaitcev/metrics/api/proto/metrics"
)

// Kind определяет тип метрики.
type Kind uint8

const (
	KindUnknown Kind = iota

	KindCounter
	KindGauge
)

var kindValues = []string{
	"unknown",
	"counter",
	"gauge",
}

func (k Kind) String() string {
	if k >= 1 && int(k) < len(kindValues) {
		return kindValues[k]
	}
	return kindValues[KindUnknown]
}

// ParseKind парсит строку и возвращает тип метрики.
func ParseKind(s string) Kind {
	if s == "" {
		return KindUnknown
	}
	s0 := strings.ToLower(s)
	for i := 0; i < len(kindValues); i++ {
		v := kindValues[i]
		if v == s0 {
			return Kind(i)
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

// Counter возвращает метрику типа счётчик с именем name и значением value.
func Counter(name string, value int64) Metric {
	return Metric{
		kind:  KindCounter,
		name:  name,
		value: counterValue(value),
	}
}

// Gauge возвращает метрику типа датчик с именем name и значением value.
func Gauge(name string, value float64) Metric {
	return Metric{
		kind:  KindGauge,
		name:  name,
		value: gaugeValue(value),
	}
}

// FromProto конвертирует *pb.UpdateRequest_Metrics в метрику и возвращает её.
func FromProto(value *pb.Metric) Metric {
	var m Metric

	switch value.GetType() {
	case pb.MetricType_COUNTER:
		m = Counter(value.GetName(), int64(value.GetValue()))
	case pb.MetricType_GAUGE:
		m = Gauge(value.GetName(), value.GetValue())
	}

	return m
}

// Proto конвертирует метрику в *pb.UpdateRequest_Metrics.
func (m *Metric) Proto() *pb.Metric {
	value := new(pb.Metric)

	switch m.Kind() {
	case KindCounter:
		value.Type = pb.MetricType_COUNTER
		value.Name = m.Name()
		value.Value = float64(m.Int64())
	case KindGauge:
		value.Type = pb.MetricType_GAUGE
		value.Name = m.Name()
		value.Value = m.Float64()
	}

	return value
}

// Kind возвращает тип метрики.
func (m *Metric) Kind() Kind {
	return m.kind
}

// Name возвращает наименование метрики.
func (m *Metric) Name() string {
	return m.name
}

// GoString возвращает строковое представление метрики.
func (m *Metric) GoString() string {
	return fmt.Sprintf(
		"metric{kind=%s name=%s value=%s}",
		m.kind.String(),
		m.name,
		m.String(),
	)
}

// String возвращает значение метрики как string.
func (m *Metric) String() string {
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
