package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
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
		Path:   "/update",
	}

	if m.Kind() == metrics.KindUnknown {
		return fmt.Errorf("unknown metric kind: %s", m.Kind())
	}

	var buf bytes.Buffer

	err := json.NewEncoder(&buf).Encode(&m)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), &buf)
	if err != nil {
		return err
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Accept-Encoding", "gzip")
	req.Header.Add("Content-Type", "application/json")

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
