package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/sergeizaitcev/metrics/internal/metrics"
	"github.com/sergeizaitcev/metrics/pkg/middleware"
	"github.com/sergeizaitcev/metrics/pkg/sign"
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

	metricChan := rateLimit(ctx, generator(ctx))

	errg, errgCtx := errgroup.WithContext(ctx)
	errg.SetLimit(flagRateLimit)

	for i := 0; i < flagRateLimit; i++ {
		errg.Go(func() error {
			for {
				select {
				case <-errgCtx.Done():
					return nil
				case values := <-metricChan:
					err := sendMetrics(ctx, values)
					if err != nil {
						return err
					}
				}
			}
		})
	}

	return errg.Wait()
}

func generator(ctx context.Context) <-chan []metrics.Metric {
	metricChan := make(chan []metrics.Metric, 1)

	go func() {
		ticker := time.NewTicker(time.Duration(flagPollInterval) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}

			select {
			case <-metricChan:
			default:
			}

			select {
			case <-ctx.Done():
				return
			case metricChan <- metrics.Snapshot():
			}
		}
	}()

	return metricChan
}

func rateLimit(ctx context.Context, in <-chan []metrics.Metric) <-chan []metrics.Metric {
	metricChan := make(chan []metrics.Metric)

	go func() {
		ticker := time.NewTicker(time.Duration(flagReportInterval) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}

			select {
			case <-ctx.Done():
				return
			case metricChan <- <-in:
			}
		}
	}()

	return metricChan
}

func sendMetrics(ctx context.Context, values []metrics.Metric) error {
	if len(values) == 0 {
		return errors.New("metrics is empty")
	}

	req, err := prepare(ctx, values)
	if err != nil {
		return err
	}

	return send(req)
}

func prepare(ctx context.Context, values []metrics.Metric) (*http.Request, error) {
	u := url.URL{
		Scheme: "http",
		Host:   flagAddress,
		Path:   "/updates/",
	}

	var buf bytes.Buffer

	err := json.NewEncoder(&buf).Encode(&values)
	if err != nil {
		return nil, fmt.Errorf("encoding metrics: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), &buf)
	if err != nil {
		return nil, fmt.Errorf("create a new request: %w", err)
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Accept-Encoding", "gzip")
	req.Header.Add("Content-Type", "application/json")

	if flagSHA256Key != "" {
		s := sign.Signer(flagSHA256Key)
		signed := s.Sign(buf.Bytes())
		hash := base64.RawURLEncoding.EncodeToString(signed)
		req.Header.Add(middleware.SignHeader, hash)
	}

	return req, nil
}

func send(req *http.Request) error {
	ctx := req.Context()

retry:
	for i := 1; i < 4; i++ {
		res, err := http.DefaultClient.Do(req)
		if err == nil {
			io.Copy(io.Discard, res.Body)
			return res.Body.Close()
		}
		if errors.Is(err, io.EOF) {
			return nil
		}

		ne, ok := err.(net.Error)
		if ok && ne.Timeout() {
			delay := time.Duration(2*i-1) * time.Second
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(delay):
				continue retry
			}
		}

		return err
	}

	return errors.New("failed to send metrics to the server")
}
