package metrics

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"

	"github.com/sergeizaitcev/metrics/pkg/randutil"
)

var snapCnt atomic.Int64

var getStats = func() func() hwstats {
	var mu sync.Mutex
	var stats hwstats
	var created time.Time

	return func() hwstats {
		mu.Lock()
		defer mu.Unlock()

		if now := time.Now(); now.Sub(created) >= time.Second {
			hw, err := newStats()
			if err != nil {
				panic(err)
			}
			created = now
			stats = hw
		}

		return stats
	}
}()

type hwstats struct {
	FreeMemory  float64
	TotalMemory float64
	CPU         float64
}

func newStats() (hwstats, error) {
	var hw hwstats

	vm, err := mem.VirtualMemory()
	if err != nil {
		return hw, fmt.Errorf("metrics: collecting memory statistics: %w", err)
	}

	percentage, err := cpu.Percent(0, false)
	if err != nil {
		return hw, fmt.Errorf("metrics: collecting cpu statistics: %w", err)
	}

	hw.FreeMemory = float64(vm.Free)
	hw.TotalMemory = float64(vm.Total)
	hw.CPU = percentage[0]

	return hw, nil
}

// Snapshot возвращает снимок метрик всей системы.
func Snapshot() []Metric {
	var ms runtime.MemStats

	runtime.ReadMemStats(&ms)
	stats := getStats()

	snapCnt.Add(1)

	return []Metric{
		Gauge("Alloc", float64(ms.Alloc)),
		Gauge("BuckHashSys", float64(ms.BuckHashSys)),
		Gauge("CPUutilization1", stats.CPU),
		Gauge("Frees", float64(ms.Frees)),
		Gauge("FreeMemory", stats.FreeMemory),
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
		Gauge("TotalMemory", stats.TotalMemory),
	}
}
