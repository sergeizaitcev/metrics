package configs

import (
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/caarlos0/env/v10"
)

// NOTE: во флагах и переменных окружения автотестов для интервалов передаются
// значения без единиц измерений, из-за этого стандартный time.ParseDuration
// не работает.
var durationType reflect.Type = reflect.TypeOf((*time.Duration)(nil)).Elem()

var customParsers = map[reflect.Type]env.ParserFunc{
	durationType: func(s string) (any, error) {
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse int: %w", err)
		}
		return duration(v), nil
	},
}

func second(d time.Duration) int64 {
	return int64(d / time.Second)
}

func duration(v int64) time.Duration {
	return time.Duration(v) * time.Second
}
