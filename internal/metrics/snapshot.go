package metrics

import (
	"crypto/rand"
	"encoding/binary"
	"io"
	mathrand "math/rand"
	"runtime"
	"sync/atomic"
)

var snapCnt atomic.Int64

var rnd = func() *mathrand.Rand {
	buf := make([]byte, 8)
	_, err := io.ReadFull(rand.Reader, buf)
	if err != nil {
		panic(err)
	}
	src := mathrand.NewSource(int64(binary.LittleEndian.Uint64(buf)))
	return mathrand.New(src)
}()

// Snapshot возвращает снимок метрик всей системы.
func Snapshot() []Metric {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	snapCnt.Add(1)

	return []Metric{
		Gauge("Alloc", float64(ms.Alloc)),
		Gauge("BuckHashSys", float64(ms.BuckHashSys)),
		Gauge("Frees", float64(ms.Frees)),
		Gauge("NumForcedGC", float64(ms.NumForcedGC)),
		Gauge("GCCPUFraction", ms.GCCPUFraction),
		Gauge("GCSys", float64(ms.GCSys)),
		Gauge("HeapAlloc", float64(ms.HeapAlloc)),
		Gauge("HeapIdle", float64(ms.HeapIdle)),
		Gauge("HeapInuse", float64(ms.HeapInuse)),
		Gauge("HeapObjects", float64(ms.HeapObjects)),
		Gauge("HeapReleased", float64(ms.HeapReleased)),
		Gauge("HeapSys", float64(ms.HeapSys)),
		Gauge("LastGC", float64(ms.LastGC)),
		Gauge("Lookups", float64(ms.Lookups)),
		Gauge("Mallocs", float64(ms.Mallocs)),
		Gauge("MCacheInuse", float64(ms.MCacheInuse)),
		Gauge("MCacheSys", float64(ms.MCacheSys)),
		Gauge("MSpanInuse", float64(ms.MSpanInuse)),
		Gauge("MSpanSys", float64(ms.MSpanSys)),
		Gauge("NextGC", float64(ms.NextGC)),
		Gauge("NumGC", float64(ms.NumGC)),
		Gauge("OtherSys", float64(ms.OtherSys)),
		Gauge("PauseTotalNs", float64(ms.PauseTotalNs)),
		Counter("PollCount", snapCnt.Load()),
		Gauge("RandomValue", rnd.Float64()),
		Gauge("StackInuse", float64(ms.StackInuse)),
		Gauge("StackSys", float64(ms.StackSys)),
		Gauge("Sys", float64(ms.Sys)),
		Gauge("TotalAlloc", float64(ms.TotalAlloc)),
	}
}
