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
	"strings"
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

	pollTicker := time.NewTicker(time.Duration(flagPollInterval) * time.Second)
	defer pollTicker.Stop()

	reportTicker := time.NewTicker(time.Duration(flagReportInterval) * time.Second)
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

		err := sendMetrics(ctx, snapshot)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
}

func sendMetrics(ctx context.Context, values []metrics.Metric) error {
	u := url.URL{
		Scheme: "http",
		Host:   flagAddress,
		Path:   "/updates/",
	}

	if len(values) == 0 {
		return errors.New("metrics is empty")
	}

	var buf bytes.Buffer

	err := json.NewEncoder(&buf).Encode(&values)
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

	for i := 0; i < 3; i++ {
		res, err := http.DefaultClient.Do(req)
		if err == nil {
			io.Copy(io.Discard, res.Body)
			return res.Body.Close()
		}

		if errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
			return nil
		}

		if strings.Contains(err.Error(), "connection refused") {
			delay := time.Duration(2*i-1) * time.Second
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(delay):
				continue
			}
		}

		return err
	}

	return errors.New("failed to send metrics to the server")
}
