package flagutil

import (
	"errors"
	"flag"
	"strconv"
	"time"
)

var _ flag.Value = (*Second)(nil)

type Second time.Duration

func (s Second) String() string {
	return s.Duration().String()
}

func (s Second) Duration() time.Duration {
	return time.Duration(s) * time.Second
}

func (s *Second) Set(value string) error {
	v, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return err
	}
	if v < 0 {
		return errors.New("value must be is greater or equal than zero")
	}
	*s = Second(v)
	return nil
}
