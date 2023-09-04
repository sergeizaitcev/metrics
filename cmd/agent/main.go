package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"time"

	"github.com/sergeizaitcev/metrics/internal/metrics"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	fs, err := parseFlags()
	if err != nil {
		return err
	}

	pollTicker := time.NewTicker(fs.pollInterval)
	defer pollTicker.Stop()

	reportTicker := time.NewTicker(fs.reportInterval)
	defer reportTicker.Stop()

	for {
		var report bool

		select {
		case <-pollTicker.C:
		case <-reportTicker.C:
			report = true
		}

		snapshot := metrics.Snapshot()
		if !report {
			continue
		}

		for _, metric := range snapshot {
			if err := reportMetric(fs.addr, metric); err != nil {
				fmt.Fprintln(os.Stderr, err)
				break
			}
		}
	}
}

func reportMetric(addr string, m metrics.Metric) error {
	u := &url.URL{
		Scheme: "http",
		Host:   addr,
	}

	switch m.Kind() {
	case metrics.KindCounter:
		u.Path = path.Join("update", "counter", m.Name(), m.String())
	case metrics.KindGauge:
		u.Path = path.Join("update", "gauge", m.Name(), m.String())
	default:
		return fmt.Errorf("unknown metric kind: %s", m.Kind())
	}

	res, err := http.Post(u.String(), "text/plain; charset=utf-8", nil)
	if err != nil {
		return err
	}

	io.Copy(io.Discard, res.Body)
	return res.Body.Close()
}
