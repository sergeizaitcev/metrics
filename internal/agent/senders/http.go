package senders

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/sergeizaitcev/metrics/internal/metrics"
	"github.com/sergeizaitcev/metrics/pkg/logging"
	"github.com/sergeizaitcev/metrics/pkg/middleware"
	"github.com/sergeizaitcev/metrics/pkg/rsautil"
	"github.com/sergeizaitcev/metrics/pkg/sign"
)

// SenderHTTP определяет агент для сбора и отправки метрик на HTTP-сервер.
type SenderHTTP struct {
	addr   string
	client *http.Client
	opts   commonOptions
}

// HTTP возвращает новый экземпляр Sender для HTTP-сервера.
func HTTP(addr string, opts ...Option) *SenderHTTP {
	sender := &SenderHTTP{
		addr:   addr,
		client: http.DefaultClient,
	}
	for _, opt := range opts {
		opt(&sender.opts)
	}
	return sender
}

func (s *SenderHTTP) Send(ctx context.Context, values []metrics.Metric) error {
	req, err := s.prepareRequest(ctx, values)
	if err != nil {
		return fmt.Errorf("request preparation: %w", err)
	}

	err = s.sendRequest(req)
	if err != nil {
		return fmt.Errorf("sending a request: %w", err)
	}

	return nil
}

func (s *SenderHTTP) prepareRequest(
	ctx context.Context,
	values []metrics.Metric,
) (*http.Request, error) {
	u := url.URL{
		Scheme: "http",
		Host:   s.addr,
		Path:   "/updates/",
	}

	body, err := s.newBody(values)
	if err != nil {
		return nil, fmt.Errorf("create a new body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), body)
	if err != nil {
		return nil, fmt.Errorf("create a new request: %w", err)
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Accept-Encoding", "gzip")
	req.Header.Add("Content-Encoding", "gzip")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add(middleware.IPHeader, s.opts.ip)

	if s.opts.sha256key != "" {
		hash := signBody(body, s.opts.sha256key)
		req.Header.Add(middleware.SignHeader, hash)
	}

	return req, nil
}

func (s *SenderHTTP) newBody(values []metrics.Metric) (*bytes.Buffer, error) {
	b, err := json.Marshal(&values)
	if err != nil {
		return nil, fmt.Errorf("encoding metrics: %w", err)
	}

	if s.opts.key != nil {
		b2, err := rsautil.Encrypt(s.opts.key, b)
		if err != nil {
			return nil, fmt.Errorf("encrypting metrics: %w", err)
		}
		b = b2
	}

	buf := bytes.NewBuffer(nil)

	// NOTE: проверка на ошибку не требуется,
	// т.к. flate.BestCompression валидное значение.
	gw, _ := gzip.NewWriterLevel(buf, flate.BestCompression)

	_, err = gw.Write(b)
	if err != nil {
		return nil, fmt.Errorf("compressing metrics: %w", err)
	}

	err = gw.Close()
	if err != nil {
		return nil, fmt.Errorf("closing gzip writer: %w", err)
	}

	return buf, nil
}

func (s *SenderHTTP) sendRequest(req *http.Request) error {
	ctx := req.Context()

	// NOTE: Если не удалось отправить запрос за установленное время,
	// то будет выполнено до 3-х попыток (с интервалами 1s, 3s и 5s),
	// прежде чем функция вернёт ошибку.
	for i := 1; i < 4; i++ {
		res, err := s.client.Do(req)
		if err == nil {
			if res.StatusCode != http.StatusOK {
				s.opts.logger.Log(logging.LevelDebug, "", "status_code", res.StatusCode)
			}
			gracefulClose(res)
			return nil
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
				continue
			}
		}

		return fmt.Errorf("sending a request: %w", err)
	}

	return errors.New("exceeded the number of attempts to send a request")
}

func gracefulClose(res *http.Response) {
	_, _ = io.Copy(io.Discard, res.Body)
	_ = res.Body.Close()
}

func signBody(body *bytes.Buffer, key string) string {
	s := sign.Signer(key)
	signed := s.Sign(body.Bytes())
	return base64.RawURLEncoding.EncodeToString(signed)
}
