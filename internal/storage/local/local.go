package local

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"

	"github.com/sergeizaitcev/metrics/internal/metrics"
	"github.com/sergeizaitcev/metrics/internal/storage"
)

// StorageOpts определяет необязательные параметры для локального хранилища
// метрик.
type StorageOpts struct {
	Synced bool // Синхронная запись.
}

// Storage определяет локальное храналище метрик, записывающее метрики на диск.
type Storage struct {
	metrics memstorage
	wal     *wal
	synced  bool

	sem chan struct{}
}

func open(filename string, flags int) (*Storage, error) {
	fd, err := os.OpenFile(filename, flags, 0644)
	if err != nil {
		return nil, fmt.Errorf("local: %w", err)
	}

	s := &Storage{
		metrics: make(memstorage),
		wal:     &wal{fd: fd},
		sem:     make(chan struct{}, 1),
	}
	s.unlock()

	return s, nil
}

// New создает локальный файл по пути filename и возвращает локальное хранилище
// метрик.
func New(filename string, opts *StorageOpts) (*Storage, error) {
	s, err := open(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC)
	if err != nil {
		return nil, err
	}

	if opts != nil {
		s.synced = opts.Synced
	}

	return s, nil
}

// Open открывает локальный файл по пути filename и возвращает локальное
// хранилище метрик.
func Open(filename string, opts *StorageOpts) (*Storage, error) {
	s, err := open(filename, os.O_RDWR|os.O_CREATE)
	if err != nil {
		return nil, err
	}

	if opts != nil {
		s.synced = opts.Synced
	}

	if err = s.load(); err != nil {
		s.Close()
		return nil, err
	}

	return s, nil
}

func (s *Storage) lock(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case _, ok := <-s.sem:
		if !ok {
			return storage.ErrStorageClosed
		}
	}

	return nil
}

func (s *Storage) unlock() {
	s.sem <- struct{}{}
}

// load загружает метрики из файла в память.
func (s *Storage) load() error {
	err := s.lock(context.Background())
	if err != nil {
		return err
	}
	defer s.unlock()

	err = s.wal.readAll(s.read)
	if err != nil {
		return fmt.Errorf("local: reading records from a file: %w", err)
	}

	return nil
}

func (s *Storage) read(e record) error {
	err := s.metrics.conflict(e.metric)
	if err != nil {
		return fmt.Errorf("conflicting metrics: %w", err)
	}

	switch e.op {
	case operationAdd:
		s.metrics.add(e.metric)
	case operationSet:
		s.metrics.set(e.metric)
	}

	return nil
}

func (s *Storage) write(op operation, value metrics.Metric) error {
	e := record{op, value}

	err := s.wal.append(e)
	if err != nil {
		return fmt.Errorf("adding an entry to the buffer: %w", err)
	}

	if !s.synced {
		return nil
	}

	err = s.wal.flush()
	if err != nil {
		return fmt.Errorf("synchronous writing to a file: %w", err)
	}

	return nil
}

// Close закрывает локальное хранилище метрик.
func (s *Storage) Close() error {
	err := s.lock(context.Background())
	if err != nil {
		return err
	}

	err = s.wal.close()
	if err != nil {
		s.unlock()
		return fmt.Errorf("local: closing a storage: %w", err)
	}

	close(s.sem)
	return nil
}

// Flush записывает буферезированные записи метрик в файл.
func (s *Storage) Flush() error {
	err := s.lock(context.Background())
	if err != nil {
		return err
	}
	defer s.unlock()

	err = s.wal.flush()
	if err != nil {
		return fmt.Errorf("local: writing buffered records to a file: %w", err)
	}

	return nil
}

// Add увеличивает значение метрики и возвращает итоговый результат.
func (s *Storage) Add(ctx context.Context, value metrics.Metric) (metrics.Metric, error) {
	if value.IsEmpty() {
		return metrics.Metric{}, errors.New("local: metric is empty")
	}

	err := s.lock(ctx)
	if err != nil {
		return metrics.Metric{}, err
	}
	defer s.unlock()

	err = s.metrics.conflict(value)
	if err != nil {
		return metrics.Metric{}, fmt.Errorf("local: conflicting metrics: %w", err)
	}

	err = s.write(operationAdd, value)
	if err != nil {
		return metrics.Metric{}, fmt.Errorf("local: writing an operation: %w", err)
	}

	return s.metrics.add(value), nil
}

// Set устанавливает новое значение метрики и возвращает предыдущее.
func (s *Storage) Set(ctx context.Context, value metrics.Metric) (metrics.Metric, error) {
	if value.IsEmpty() {
		return metrics.Metric{}, errors.New("local: metric is empty")
	}

	err := s.lock(ctx)
	if err != nil {
		return metrics.Metric{}, err
	}
	defer s.unlock()

	err = s.metrics.conflict(value)
	if err != nil {
		return metrics.Metric{}, fmt.Errorf("local: conflicting metrics: %w", err)
	}

	err = s.write(operationSet, value)
	if err != nil {
		return metrics.Metric{}, fmt.Errorf("local: writing an operation: %w", err)
	}

	return s.metrics.set(value), nil
}

// Get возвращает метрику.
func (s *Storage) Get(ctx context.Context, name string) (metrics.Metric, error) {
	err := s.lock(ctx)
	if err != nil {
		return metrics.Metric{}, err
	}

	actual := s.metrics.get(name)
	s.unlock()

	if actual.IsEmpty() {
		return metrics.Metric{}, storage.ErrNotFound
	}

	return actual, nil
}

// GetAll возвращает все метрики.
func (s *Storage) GetAll(ctx context.Context) ([]metrics.Metric, error) {
	err := s.lock(ctx)
	if err != nil {
		return nil, err
	}

	values := s.metrics.getAll()
	s.unlock()

	sort.SliceStable(values, func(i, j int) bool {
		return values[i].Name() < values[j].Name()
	})

	return values, nil
}
