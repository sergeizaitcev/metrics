package agent

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/sergeizaitcev/metrics/internal/configs"
	"github.com/sergeizaitcev/metrics/internal/metrics"
	"github.com/sergeizaitcev/metrics/pkg/iputil"
	"github.com/sergeizaitcev/metrics/pkg/logging"
	"github.com/sergeizaitcev/metrics/pkg/middleware"
	"github.com/sergeizaitcev/metrics/pkg/rsautil"
	"github.com/sergeizaitcev/metrics/pkg/sign"
)

var defaultOptions = &AgentOpts{
	Logger:    logging.Discard(),
	Transport: http.DefaultTransport.(*http.Transport).Clone(),
}

// AgentOpts определяет не обязательные параметры для Agent.
type AgentOpts struct {
	// Логирование ошибок.
	Logger *logging.Logger

	// Время ожидания ответа от сервера.
	Timeout time.Duration

	// Пользовательский транспорт.
	Transport http.RoundTripper

	// Публичный ключ для асиметричного шифрования.
	Key *rsa.PublicKey
}

// Agent определяет агент для сбора и отправки метрик на сервер.
type Agent struct {
	config *configs.Agent
	client *http.Client
	logger *logging.Logger
	key    *rsa.PublicKey
	ip     string
}

// New возвращает новый экземпляр Agent.
func New(config *configs.Agent, opts *AgentOpts) *Agent {
	if opts == nil {
		opts = defaultOptions
	}

	if opts.Logger == nil {
		opts.Logger = defaultOptions.Logger
	}
	if opts.Transport == nil {
		opts.Transport = defaultOptions.Transport.(*http.Transport).Clone()
	}

	client := &http.Client{
		Timeout:   opts.Timeout,
		Transport: opts.Transport,
	}

	return &Agent{
		config: config,
		client: client,
		logger: opts.Logger,
		key:    opts.Key,
		ip:     iputil.Local().String(),
	}
}

// Run собирает метрики и отправляет их на сервер; блокируется до тех пор, пока
// не сработает контекст или функция не вернёт ошибку.
func (a *Agent) Run(ctx context.Context) error {
	var wg sync.WaitGroup

	collectChan := make(chan []metrics.Metric, a.config.RateLimit)
	a.Collect(ctx, collectChan)

	for i := 0; i < a.config.RateLimit; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				case snapshot := <-collectChan:
					a.logger.Log(logging.LevelDebug, "sending a new metrics batch",
						"batch_size", len(snapshot),
					)

					start := time.Now()
					err := a.Send(ctx, snapshot)
					elapsed := time.Since(start)

					if err != nil {
						a.logger.Log(logging.LevelError, err.Error())
						continue
					}

					a.logger.Log(logging.LevelDebug, "the metrics batch was sent successfully",
						"batch_size", len(snapshot),
						"elapsed", elapsed.String(),
					)
				}
			}
		}()
	}

	wg.Wait()

	return nil
}

// Collect собирает метрики и передаёт их в канал collectChan.
func (a *Agent) Collect(ctx context.Context, collectChan chan<- []metrics.Metric) {
	go func() {
		pollChan := a.poll(ctx)

		ticker := time.NewTicker(a.config.ReportInterval)
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
			case collectChan <- <-pollChan:
			}
		}
	}()
}

// poll возвращает канал, в который отправляются снимки метрик с интервалом
// PollInterval.
func (a *Agent) poll(ctx context.Context) <-chan []metrics.Metric {
	pollChan := make(chan []metrics.Metric)

	go func() {
		ticker := time.NewTicker(a.config.PollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}

			snapshot := metrics.Snapshot()

			select {
			case pollChan <- snapshot:
			default:
			}
		}
	}()

	return pollChan
}

// Send отправляет метрики на сервер.
func (a *Agent) Send(ctx context.Context, values []metrics.Metric) error {
	if len(values) == 0 {
		return errors.New("agent: metrics is empty")
	}

	req, err := a.prepareRequest(ctx, values)
	if err != nil {
		return fmt.Errorf("agent: request preparation: %w", err)
	}

	err = a.sendRequest(req)
	if err != nil {
		return fmt.Errorf("agent: sending a request: %w", err)
	}

	return nil
}

func (a *Agent) prepareRequest(
	ctx context.Context,
	values []metrics.Metric,
) (*http.Request, error) {
	u := url.URL{
		Scheme: "http",
		Host:   a.config.Address,
		Path:   "/updates/",
	}

	body, err := a.newBody(values)
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
	req.Header.Add(middleware.IPHeader, a.ip)

	if a.config.SHA256Key != "" {
		hash := signBody(body, a.config.SHA256Key)
		req.Header.Add(middleware.SignHeader, hash)
	}

	return req, nil
}

func (a *Agent) newBody(values []metrics.Metric) (*bytes.Buffer, error) {
	b, err := json.Marshal(&values)
	if err != nil {
		return nil, fmt.Errorf("encoding metrics: %w", err)
	}

	if a.key != nil {
		b2, err := rsautil.Encrypt(a.key, b)
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

func signBody(body *bytes.Buffer, key string) string {
	s := sign.Signer(key)
	signed := s.Sign(body.Bytes())
	return base64.RawURLEncoding.EncodeToString(signed)
}

func (a *Agent) sendRequest(req *http.Request) error {
	ctx := req.Context()

	// NOTE: Если не удалось отправить запрос за установленное время,
	// то будет выполнено до 3-х попыток (с интервалами 1s, 3s и 5s),
	// прежде чем функция вернёт ошибку.
	for i := 1; i < 4; i++ {
		res, err := a.client.Do(req)
		if err == nil {
			if res.StatusCode != http.StatusOK {
				a.logger.Log(logging.LevelDebug, "", "status_code", res.StatusCode)
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
