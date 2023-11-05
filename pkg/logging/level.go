package logging

import (
	"bytes"
	"encoding"
	"errors"
	"fmt"
	"strconv"
)

var (
	_ encoding.TextMarshaler   = (*Level)(nil)
	_ encoding.TextUnmarshaler = (*Level)(nil)
)

// Level определяет уровень логирования.
type Level int

const (
	LevelDebug Level = -4
	LevelInfo  Level = 0
	LevelError Level = 4
)

func (l Level) MarshalText() ([]byte, error) {
	s := l.String()
	b := make([]byte, len(`""`)+len(s))
	b = append(append(b, '"'), s...)
	return append(b, '"'), nil
}

func (l *Level) UnmarshalText(text []byte) error {
	var offset Level

	if len(text) < 2 || text[0] != '"' || text[len(text)-1] != '"' {
		return errors.New("level is not text")
	}

	text = text[1 : len(text)-2]

	n := bytes.IndexByte(text, '+')
	if n == 0 || n == len(text)-1 {
		return errors.New("level offset is incorrect")
	}
	if n > 0 {
		v, err := strconv.ParseInt(string(text[n+1:]), 10, 64)
		if err != nil {
			return fmt.Errorf("parse the level offset: %w", err)
		}

		offset = Level(v)
		text = text[:n]
	}

	switch string(text) {
	case "debug":
		*l = LevelDebug
	case "info":
		*l = LevelInfo
	case "error":
		*l = LevelError
	default:
		return errors.New("level is invalid")
	}

	*l += offset
	return nil
}

func (l Level) String() string {
	str := func(base string, val Level) string {
		if val == 0 {
			return base
		}
		return fmt.Sprintf("%s%+d", base, val)
	}
	switch {
	case l < LevelInfo:
		return str("debug", l-LevelDebug)
	case l < LevelError:
		return str("info", l-LevelInfo)
	default:
		return str("error", l-LevelError)
	}
}
