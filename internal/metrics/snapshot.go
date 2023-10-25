package metrics

import (
	"runtime"
	"sync/atomic"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"

	"github.com/sergeizaitcev/metrics/pkg/randutil"
)

var snapCnt atomic.Int64

// Snapshot возвращает снимок метрик всей системы.
func Snapshot() []Metric {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	memstat, err := mem.VirtualMemory()
	if err != nil {
		panic(err)
	}

	cpustat, err := cpu.Percent(0, false)
	if err != nil {
		panic(err)
	}

	snapCnt.Add(1)

	return []Metric{
		Gauge("Alloc", float64(ms.Alloc)),
		Gauge("BuckHashSys", float64(ms.BuckHashSys)),
		Gauge("CPUutilization1", cpustat[0]),
		Gauge("Frees", float64(ms.Frees)),
		Gauge("FreeMemory", float64(memstat.Free)),
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
		Gauge("NumForcedGC", float64(ms.NumForcedGC)),
		Gauge("NumGC", float64(ms.NumGC)),
		Gauge("OtherSys", float64(ms.OtherSys)),
		Gauge("PauseTotalNs", float64(ms.PauseTotalNs)),
		Counter("PollCount", snapCnt.Load()),
		Gauge("RandomValue", randutil.Float64()),
		Gauge("StackInuse", float64(ms.StackInuse)),
		Gauge("StackSys", float64(ms.StackSys)),
		Gauge("Sys", float64(ms.Sys)),
		Gauge("TotalAlloc", float64(ms.TotalAlloc)),
		Gauge("TotalMemory", float64(memstat.Total)),
	}
}
