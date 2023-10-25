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
	// Интервал сброса данных в WAL.
	//
	// При StoreInterval == 0, сброс данных в WAL происходит синхронно.
	StoreInterval time.Duration

	// Восстановление данных из WAL.
	Restore bool
}

var _ storage.Storager = (*Storage)(nil)

// Storage определяет локальное храналище метрик, записывающее метрики на диск.
type Storage struct {
	metrics memstorage
	wal     *wal
	synced  bool

	sem chan struct{}

	term         chan struct{}
	singleflight chan struct{}
}

// New возвращает локальное хранилище метрик.
func New(filename string, opts *StorageOpts) (*Storage, error) {
	flags := os.O_RDWR | os.O_CREATE
	if opts == nil || !opts.Restore {
		flags |= os.O_TRUNC
	}

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

	if opts != nil && opts.Restore {
		if err = s.load(); err != nil {
			s.Close()
			return nil, err
		}
	}

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
	case operationUpdate:
		s.metrics.update(e.metric)
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

// Ping реализует интерфейс storage.Storager.
func (s *Storage) Ping(ctx context.Context) error {
	err := s.lockContext(ctx)
	if err != nil {
		return err
	}
	s.unlock()
	return nil
}

// Close реализует интерфейс storage.Storager.
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

// Save реализует интерфейс storage.Storager.
func (s *Storage) Save(ctx context.Context, values ...metrics.Metric) ([]metrics.Metric, error) {
	if len(values) == 0 {
		return nil, errors.New("metrics is empty")
	}

	err := s.lockContext(ctx)
	if err != nil {
		return nil, err
	}
	defer s.unlock()

	actuals := make([]metrics.Metric, len(values))
	var writed bool

	for i, value := range values {
		if value.IsEmpty() {
			continue
		}

		err = s.metrics.conflict(value)
		if err != nil {
			return nil, fmt.Errorf("local: conflicting metrics: %w", err)
		}

		switch value.Kind() {
		case metrics.KindCounter:
			err = s.write(operationAdd, value)
			if err != nil {
				return nil, fmt.Errorf("local: writing an add operation: %w", err)
			}
			actuals[i] = s.metrics.add(value)
		case metrics.KindGauge:
			err = s.write(operationUpdate, value)
			if err != nil {
				return nil, fmt.Errorf("local: writing an update operation: %w", err)
			}
			actuals[i] = s.metrics.update(value)
		}

		writed = true
	}

	if s.synced && writed {
		err = s.wal.flush()
		if err != nil {
			return nil, fmt.Errorf("local: synchronous writing to a file: %w", err)
		}
	}

	return actuals, nil
}

// Get реализует интерфейс storage.Storager.
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

// GetAll реализует интерфейс storage.Storager.
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
