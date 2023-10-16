package local

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"unicode/utf8"

	"github.com/sergeizaitcev/metrics/internal/metrics"
)

var (
	separator    = '\xb1'
	separatorLen = utf8.RuneLen(separator)
)

type operation uint8

const (
	operationUnknown operation = iota
	operationAdd
	operationSet
)

var operations = []operation{
	operationUnknown,
	operationAdd,
	operationSet,
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

// wal определяет файловое хранилище метрик.
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
