package storage

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"sort"
	"time"
	"unicode/utf8"

	"github.com/sergeizaitcev/metrics/internal/metrics"
)

// LocalOpts определяет необязательные параметры для локального хранилища
// метрик.
type LocalOpts struct {
	// Интервал сброса данных в WAL.
	//
	// При StoreInterval == 0, сброс данных в WAL происходит синхронно.
	StoreInterval time.Duration

	// Восстановление данных из WAL.
	Restore bool
}

var _ Storage = (*Local)(nil)

// Local определяет локальное храналище метрик, записывающее метрики на диск
// и хранящее кеш в памяти.
type Local struct {
	metrics memstorage
	wal     *wal
	synced  bool // Индикатор синхронной записи.

	sem chan struct{}

	term         chan struct{}
	singleflight chan struct{}
}

// NewLocal возвращает локальное хранилище метрик.
func NewLocal(filename string, opts *LocalOpts) (*Local, error) {
	flags := os.O_RDWR | os.O_CREATE
	if opts == nil || !opts.Restore {
		flags |= os.O_TRUNC
	}

	fd, err := os.OpenFile(filename, flags, 0644)
	if err != nil {
		return nil, fmt.Errorf("local: %w", err)
	}

	local := &Local{
		metrics: make(memstorage),
		wal:     &wal{fd: fd},
		sem:     make(chan struct{}, 1),
	}
	local.unlock()

	if opts != nil && opts.Restore {
		if err = local.load(); err != nil {
			local.Close()
			return nil, err
		}
	}

	var interval time.Duration

	if opts != nil {
		interval = opts.StoreInterval
		local.synced = interval == 0
	}
	if interval > 0 {
		go local.flushing(interval)
	}

	return local, nil
}

// Ping реализует интерфейс Storage.
func (l *Local) Ping(ctx context.Context) error {
	err := l.lockContext(ctx)
	if err != nil {
		return err
	}
	l.unlock()
	return nil
}

// Close реализует интерфейс Storage.
func (l *Local) Close() error {
	err := l.lock()
	if err != nil {
		return err
	}

	l.wal.close()

	// NOTE: если был запущен flushing в фоне, то Close будет ждать
	// завершения flushing.
	if l.term != nil {
		close(l.term)
	}
	if l.singleflight != nil {
		<-l.singleflight
	}

	close(l.sem)
	return nil
}

// Save реализует интерфейс Storage.
func (l *Local) Save(ctx context.Context, values ...metrics.Metric) ([]metrics.Metric, error) {
	if len(values) == 0 {
		return nil, errors.New("metrics is empty")
	}

	err := l.lockContext(ctx)
	if err != nil {
		return nil, err
	}
	defer l.unlock()

	actuals := make([]metrics.Metric, len(values))
	var written bool

	for i, value := range values {
		if value.IsEmpty() {
			continue
		}

		err = l.metrics.conflict(value)
		if err != nil {
			return nil, fmt.Errorf("local: conflicting metrics: %w", err)
		}

		switch value.Kind() {
		case metrics.KindCounter:
			err = l.write(operationAdd, value)
			if err != nil {
				return nil, fmt.Errorf("local: writing an add operation: %w", err)
			}
			actuals[i] = l.metrics.add(value)
		case metrics.KindGauge:
			err = l.write(operationUpdate, value)
			if err != nil {
				return nil, fmt.Errorf("local: writing an update operation: %w", err)
			}
			actuals[i] = l.metrics.update(value)
		}

		written = true
	}

	if l.synced && written {
		err = l.wal.flush()
		if err != nil {
			return nil, fmt.Errorf("local: synchronous writing to a file: %w", err)
		}
	}

	return actuals, nil
}

// Get реализует интерфейс Storage.
func (l *Local) Get(ctx context.Context, name string) (metrics.Metric, error) {
	// FIXME: при большом количестве чтения необходимо
	// реализовать RLOCK.
	err := l.lockContext(ctx)
	if err != nil {
		return metrics.Metric{}, err
	}

	actual := l.metrics.get(name)
	l.unlock()

	if actual.IsEmpty() {
		return metrics.Metric{}, ErrNotFound
	}

	return actual, nil
}

// GetAll реализует интерфейс Storage.
func (l *Local) GetAll(ctx context.Context) ([]metrics.Metric, error) {
	// FIXME: при большом количестве чтения необходимо
	// реализовать RLOCK.
	err := l.lockContext(ctx)
	if err != nil {
		return nil, err
	}

	values := l.metrics.getAll()
	l.unlock()

	sort.SliceStable(values, func(i, j int) bool {
		return values[i].Name() < values[j].Name()
	})

	return values, nil
}

func (l *Local) lockContext(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case _, ok := <-l.sem:
		if !ok {
			return ErrStorageClosed
		}
	}
	return nil
}

func (l *Local) lock() error {
	return l.lockContext(context.Background())
}

func (l *Local) unlock() {
	l.sem <- struct{}{}
}

// load загружает метрики из файла в кеш.
func (l *Local) load() error {
	err := l.lock()
	if err != nil {
		return err
	}
	defer l.unlock()

	err = l.wal.readAll(l.read)
	if err != nil {
		return fmt.Errorf("local: reading records from a file: %w", err)
	}

	return nil
}

// read читает метрику в файла и записывает в кеш.
func (l *Local) read(e record) error {
	err := l.metrics.conflict(e.metric)
	if err != nil {
		return fmt.Errorf("conflicting metrics: %w", err)
	}

	switch e.op {
	case operationAdd:
		l.metrics.add(e.metric)
	case operationUpdate:
		l.metrics.update(e.metric)
	}

	return nil
}

// write записывает метрику на диск.
func (l *Local) write(op operation, value metrics.Metric) error {
	e := record{op, value}

	err := l.wal.append(e)
	if err != nil {
		return fmt.Errorf("adding an entry to the buffer: %w", err)
	}

	return nil
}

// flush сбрасывает буфер с метриками на диск.
func (l *Local) flush() error {
	err := l.lock()
	if err != nil {
		return err
	}
	defer l.unlock()

	err = l.wal.flush()
	if err != nil {
		return fmt.Errorf("local: writing buffered records to a file: %w", err)
	}

	return nil
}

func (l *Local) flushing(d time.Duration) {
	err := l.lock()
	if err != nil {
		return
	}

	if l.term == nil {
		l.term = make(chan struct{})
	}
	if l.singleflight == nil {
		l.singleflight = make(chan struct{})
	}

	l.unlock()

	ticker := time.NewTicker(d)
	defer ticker.Stop()

	for {
		select {
		case <-l.term:
			// NOTE: дополнительного вызова flush не требуется,
			// т.к. хранилище при закрытии выполняет flush.
			close(l.singleflight)
			return
		case <-ticker.C:
			l.flush()
		}
	}
}

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

// update обновляет значение метрики и возвращает предыдущее.
func (s memstorage) update(value metrics.Metric) metrics.Metric {
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

var (
	separator    = '\xb1'
	separatorLen = utf8.RuneLen(separator)
)

type operation uint8

const (
	operationUnknown operation = iota
	operationAdd
	operationUpdate
)

var operations = []operation{
	operationUnknown,
	operationAdd,
	operationUpdate,
}

func validate(op operation) error {
	if op > 0 && int(op) < len(operations) {
		v := operations[op]
		if v != operationUnknown {
			return nil
		}
	}
	return errors.New("operation is unknown")
}

type record struct {
	op     operation
	metric metrics.Metric
}

func (r record) MarshalBinary() ([]byte, error) {
	data, err := r.metric.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("converting a metric to bytes: %w", err)
	}

	var buf [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(buf[:], uint64(len(data)))

	start := 4
	end := start + n + len(data) + 1

	b := make([]byte, start, end)
	b = append(b, byte(r.op))
	b = append(b, buf[:n]...)
	b = append(b, data...)

	crc := crc32.NewIEEE()
	crc.Write(b[start:])
	binary.BigEndian.PutUint32(b, crc.Sum32())

	return b, nil
}

func (r *record) UnmarshalBinary(data []byte) error {
	if len(data) < 6 {
		return errors.New("record too small")
	}

	op := operation(data[4])
	if err := validate(op); err != nil {
		return err
	}

	size, n := binary.Uvarint(data[5:])
	if size <= 0 || len(data[5:])-n < int(size) {
		return errors.New("record is corrupted")
	}

	start := 4
	end := start + n + int(size) + 1

	sum := binary.BigEndian.Uint32(data[:start])
	crc := crc32.NewIEEE()
	crc.Write(data[start:end])

	if sum != crc.Sum32() {
		return errors.New("invalid record")
	}

	var metric metrics.Metric

	err := metric.UnmarshalBinary(data[end-int(size) : end])
	if err != nil {
		return err
	}

	*r = record{
		op:     op,
		metric: metric,
	}

	return nil
}

// wal определяет файл для упреждающей журнализации.
type wal struct {
	buf bytes.Buffer
	fd  *os.File
}

// close сбрасывает содержимое буфера в конец файла и закрывает его.
func (w *wal) close() error {
	err := w.flush()
	if err != nil {
		return fmt.Errorf("writing buffered record to a file: %w", err)
	}

	err = w.fd.Close()
	if err != nil {
		return fmt.Errorf("closing a file: %w", err)
	}

	return nil
}

// flush сбрасывает содержимое буфера в конец файла.
func (w *wal) flush() error {
	if w.buf.Len() == 0 {
		return nil
	}

	_, err := w.fd.Seek(0, io.SeekEnd)
	if err != nil {
		return fmt.Errorf("offset to the end: %w", err)
	}

	_, err = w.buf.WriteTo(w.fd)
	if err != nil {
		return fmt.Errorf("writing to a file: %w", err)
	}

	w.buf.Reset()

	err = w.fd.Sync()
	if err != nil {
		return fmt.Errorf("file synchronization: %w", err)
	}

	return nil
}

// append добавляет запись в конец буфера.
func (w *wal) append(r record) error {
	b, err := r.MarshalBinary()
	if err != nil {
		return fmt.Errorf("converting a record to bytes: %w", err)
	}

	w.buf.Write(b)
	w.buf.WriteRune(separator)

	return nil
}

// readAll считывает хранимые в файле записи и передает их в f.
func (w *wal) readAll(f func(record) error) error {
	_, err := w.fd.Seek(0, io.SeekStart)
	if err != nil {
		return fmt.Errorf("offset to the beginning: %w", err)
	}

	err = scan(w.fd, f)
	if err != nil {
		return fmt.Errorf("scanning a file: %w", err)
	}

	return nil
}

func scan(r io.Reader, f func(record) error) error {
	scanner := bufio.NewScanner(r)
	scanner.Split(split)

	for scanner.Scan() {
		var e record

		err := e.UnmarshalBinary(scanner.Bytes())
		if err != nil {
			return err
		}

		if err = f(e); err != nil {
			return err
		}
	}

	return scanner.Err()
}

func split(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return
	}
	n := bytes.IndexRune(data, separator)
	if n > 0 {
		return n + separatorLen, data[0:n], nil
	}
	if atEOF {
		return len(data), data, nil
	}
	return
}
