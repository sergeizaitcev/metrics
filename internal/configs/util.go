package configs

import (
	"time"
)

func second(d time.Duration) int64 {
	return int64(d / time.Second)
}

func duration(v int64) time.Duration {
	return time.Duration(v) * time.Second
}
