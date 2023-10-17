package local

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/sergeizaitcev/metrics/internal/metrics"
	"github.com/sergeizaitcev/metrics/internal/storage"
)

// StorageOpts определяет необязательные параметры для локального хранилища
// метрик.
type StorageOpts struct {
	StoreInterval time.Duration
}

// Storage определяет локальное храналище метрик, записывающее метрики на диск.
type Storage struct {
	metrics memstorage
	wal     *wal
	synced  bool

	sem chan struct{}

	term         chan struct{}
	singleflight chan struct{}
}

func create(filename string, flags int, opts *StorageOpts) (*Storage, error) {
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

	var interval time.Duration

	if opts != nil {
		interval = opts.StoreInterval
		s.synced = interval == 0
	}
	if interval > 0 {
		go s.flushing(interval)
	}

	return s, nil
}

// New создает локальный файл по пути filename и возвращает локальное хранилище
// метрик.
func New(filename string, opts *StorageOpts) (*Storage, error) {
	return create(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, opts)
}

// Open открывает локальный файл по пути filename и возвращает локальное
// хранилище метрик.
func Open(filename string, opts *StorageOpts) (*Storage, error) {
	s, err := create(filename, os.O_RDWR|os.O_CREATE, opts)
	if err != nil {
		return nil, err
	}

	if err = s.load(); err != nil {
		s.Close()
		return nil, err
	}

	return s, nil
}

func (s *Storage) lockContext(ctx context.Context) error {
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

func (s *Storage) lock() error {
	return s.lockContext(context.Background())
}

func (s *Storage) unlock() {
	s.sem <- struct{}{}
}

// load загружает метрики из файла в память.
func (s *Storage) load() error {
	err := s.lock()
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

	return nil
}

func (s *Storage) flush() error {
	err := s.lock()
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

func (s *Storage) flushing(d time.Duration) {
	err := s.lock()
	if err != nil {
		return
	}

	if s.term == nil {
		s.term = make(chan struct{})
	}
	if s.singleflight == nil {
		s.singleflight = make(chan struct{})
	}

	s.unlock()

	ticker := time.NewTicker(d)
	defer ticker.Stop()

	for {
		select {
		case <-s.term:
			close(s.singleflight)
			return
		case <-ticker.C:
			s.flush()
		}
	}
}

// Ping возвращает ошибку, если не удалось выполнить пинг хранилища.
func (s *Storage) Ping(ctx context.Context) error {
	err := s.lockContext(ctx)
	if err != nil {
		return err
	}
	s.unlock()
	return nil
}

// Close закрывает локальное хранилище метрик.
func (s *Storage) Close() error {
	err := s.lock()
	if err != nil {
		return err
	}

	s.wal.close()

	if s.term != nil {
		close(s.term)
	}
	if s.singleflight != nil {
		<-s.singleflight
	}

	close(s.sem)
	return nil
}

// SaveMany устанавливает или увеличивает значения метрики.
func (s *Storage) SaveMany(ctx context.Context, values []metrics.Metric) error {
	if len(values) == 0 {
		return nil
	}

	err := s.lockContext(ctx)
	if err != nil {
		return err
	}
	defer s.unlock()

	var writed bool

	for _, value := range values {
		if value.IsEmpty() {
			continue
		}

		writed = true

		err = s.metrics.conflict(value)
		if err != nil {
			return fmt.Errorf("local: conflicting metrics: %w", err)
		}

		switch value.Kind() {
		case metrics.KindCounter:
			err = s.write(operationAdd, value)
			if err != nil {
				return fmt.Errorf("local: writing an operation: %w", err)
			}
			s.metrics.add(value)
		case metrics.KindGauge:
			err = s.write(operationSet, value)
			if err != nil {
				return fmt.Errorf("local: writing an operation: %w", err)
			}
			s.metrics.set(value)
		}
	}

	if s.synced && writed {
		err = s.wal.flush()
		if err != nil {
			return fmt.Errorf("local: synchronous writing to a file: %w", err)
		}
	}

	return nil
}

// Add увеличивает значение метрики и возвращает итоговый результат.
func (s *Storage) Add(ctx context.Context, value metrics.Metric) (metrics.Metric, error) {
	if value.IsEmpty() {
		return metrics.Metric{}, errors.New("local: metric is empty")
	}

	err := s.lockContext(ctx)
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

	if s.synced {
		err = s.wal.flush()
		if err != nil {
			return metrics.Metric{}, fmt.Errorf("local: synchronous writing to a file: %w", err)
		}
	}

	return s.metrics.add(value), nil
}

// Set устанавливает новое значение метрики и возвращает предыдущее.
func (s *Storage) Set(ctx context.Context, value metrics.Metric) (metrics.Metric, error) {
	if value.IsEmpty() {
		return metrics.Metric{}, errors.New("local: metric is empty")
	}

	err := s.lockContext(ctx)
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

	if s.synced {
		err = s.wal.flush()
		if err != nil {
			return metrics.Metric{}, fmt.Errorf("local: synchronous writing to a file: %w", err)
		}
	}

	return s.metrics.set(value), nil
}

// Get возвращает метрику.
func (s *Storage) Get(ctx context.Context, name string) (metrics.Metric, error) {
	err := s.lockContext(ctx)
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
	err := s.lockContext(ctx)
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
