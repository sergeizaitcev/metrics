package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path"
	"syscall"
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
	if err := parseFlags(); err != nil {
		return err
	}

	baseCtx := context.Background()

	ctx, cancel := signal.NotifyContext(baseCtx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	pollTicker := time.NewTicker(time.Duration(flagPollInterval))
	defer pollTicker.Stop()

	reportTicker := time.NewTicker(time.Duration(flagReportInterval))
	defer reportTicker.Stop()

	for {
		var report bool

		select {
		case <-ctx.Done():
			return nil
		case <-pollTicker.C:
		case <-reportTicker.C:
			report = true
		}

		snapshot := metrics.Snapshot()
		if !report {
			continue
		}

		for _, metric := range snapshot {
			if err := sendMetric(ctx, metric); err != nil {
				fmt.Fprintln(os.Stderr, err)
				break
			}
		}
	}
}

func sendMetric(ctx context.Context, m metrics.Metric) error {
	u := url.URL{
		Scheme: "http",
		Host:   flagAddress,
	}

	switch m.Kind() {
	case metrics.KindCounter:
		u.Path = path.Join("update", "counter", m.Name(), m.String())
	case metrics.KindGauge:
		u.Path = path.Join("update", "gauge", m.Name(), m.String())
	default:
		return fmt.Errorf("unknown metric kind: %s", m.Kind())
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), nil)
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "text/plain; charset=utf-8")

	res, err := http.DefaultClient.Do(req)
	if errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
		return nil
	}
	if err != nil {
		return err
	}

	io.Copy(io.Discard, res.Body)
	return res.Body.Close()
}
