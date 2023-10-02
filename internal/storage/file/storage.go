package file

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"sync"

	"github.com/sergeizaitcev/metrics/internal/metrics"
)

// Storage определяет файловое хранилище метрик.
type Storage struct {
	mu     sync.Mutex
	fd     *os.File
	enc    *json.Encoder
	buf    bytes.Buffer
	synced bool
}

// Open открывает файл filename и возвращает экземпляр Storage.
func Open(filename string) (*Storage, error) {
	fd, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}

	f := &Storage{fd: fd}
	f.enc = json.NewEncoder(&f.buf)

	return f, nil
}

// OpenSync открывает файл filename и возвращает экземпляр синхронный Storage.
func OpenSync(filename string) (*Storage, error) {
	f, err := Open(filename)
	if err != nil {
		return nil, err
	}
	f.synced = true
	return f, nil
}

// Close сохраняет содержимое буфера в файл и закрывает его.
func (s *Storage) Close() error {
	err := s.Flush()
	if err != nil {
		return err
	}
	return s.fd.Close()
}

// Flush сохраняет содержимое буфера в файл.
func (s *Storage) Flush() error {
	if s.buf.Len() == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return s.flush()
}

func (s *Storage) flush() error {
	_, err := s.fd.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}

	_, err = s.buf.WriteTo(s.fd)
	if err != nil {
		return err
	}

	s.buf.Truncate(s.buf.Len())

	return s.fd.Sync()
}

// Append добавляет метрики в конец буфера.
func (s *Storage) Append(metric metrics.Metric) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := s.enc.Encode(&metric)
	if err != nil {
		return err
	}

	if s.synced {
		return s.flush()
	}

	return nil
}

// ReadAll считывает хранимые в файле метрики и возвращает их.
func (s *Storage) ReadAll() ([]metrics.Metric, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.fd.Seek(0, io.SeekStart)
	if err != nil {
		return nil, err
	}

	batch := make([]metrics.Metric, 0, 64)

	scanner := bufio.NewScanner(s.fd)
	for scanner.Scan() {
		var metric metrics.Metric

		err = json.Unmarshal(scanner.Bytes(), &metric)
		if err != nil {
			return nil, err
		}

		batch = append(batch, metric)
	}

	err = scanner.Err()
	if err != nil {
		return nil, err
	}

	return batch, nil
}
