package agent

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
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/sergeizaitcev/metrics/internal/metrics"
	"github.com/sergeizaitcev/metrics/pkg/middleware"
	"github.com/sergeizaitcev/metrics/pkg/sign"
)

var defaultOptions = &AgentOpts{
	ReportInterval: 10 * time.Second,
	PollInterval:   2 * time.Second,
	RateLimit:      1,
	Transport:      http.DefaultTransport.(*http.Transport).Clone(),
}

// AgentOpts определяет не обязательные параметры для агента.
type AgentOpts struct {
	// Ключ подписи данных. Если ключ пуст, то данные не подписываются.
	SHA256Key string

	// Интервал отправки метрик на сервер.
	//
	// По умолчанию 10s.
	ReportInterval time.Duration

	// Интервал сбора метрик.
	//
	// По умолчанию 2s.
	PollInterval time.Duration

	// Количество одновременных запросов на сервер.
	//
	// По умолчанию 1.
	RateLimit int

	// Время ожидания ответа от сервера.
	Timeout time.Duration

	// Пользовательский транспорт.
	Transport http.RoundTripper
}

func (o *AgentOpts) clone() *AgentOpts {
	o2 := *o
	return &o2
}

// Agent определяет агент для сбора и отправки метрик на сервер.
type Agent struct {
	addr   string
	client http.Client
	opts   *AgentOpts
}

// New возвращает новый экземпляр Agent.
func New(addr string, opts *AgentOpts) *Agent {
	if opts == nil {
		opts = defaultOptions
	}

	o2 := opts.clone()

	if o2.PollInterval <= 0 {
		o2.PollInterval = defaultOptions.PollInterval
	}
	if o2.ReportInterval <= 0 {
		o2.ReportInterval = defaultOptions.ReportInterval
	}
	if o2.RateLimit < 1 {
		o2.RateLimit = 1
	}
	if o2.Transport == nil {
		o2.Transport = defaultOptions.Transport.(*http.Transport).Clone()
	}

	client := http.Client{
		Timeout:   o2.Timeout,
		Transport: o2.Transport,
	}

	return &Agent{addr: addr, client: client, opts: o2}
}

// Run собирает метрики и отправляет их на сервер; блокируется до тех пор, пока
// не сработает контекст или функция не вернёт ошибку.
func (a *Agent) Run(ctx context.Context) error {
	errg, errgCtx := errgroup.WithContext(ctx)
	collectChan := a.Collect(errgCtx)

	for i := 0; i < a.opts.RateLimit; i++ {
		errg.Go(func() error {
			for {
				select {
				case <-errgCtx.Done():
					return nil
				case snapshot := <-collectChan:
					err := a.Send(ctx, snapshot)
					if err != nil {
						return err
					}
				}
			}
		})
	}

	return errg.Wait()
}

// Collect собирает метрики и передаёт их в возвращаемый канал.
func (a *Agent) Collect(ctx context.Context) <-chan []metrics.Metric {
	snapshotChan := make(chan []metrics.Metric)

	go func() {
		pollChan := a.poll(ctx)

		ticker := time.NewTicker(a.opts.ReportInterval)
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
			case snapshotChan <- <-pollChan:
			}
		}
	}()

	return snapshotChan
}

// poll возвращает канал, в который отправляются снимки метрик с интервалом
// PollInterval.
func (a *Agent) poll(ctx context.Context) <-chan []metrics.Metric {
	pollChan := make(chan []metrics.Metric, 1)

	go func() {
		ticker := time.NewTicker(a.opts.PollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}

			// NOTE: Если предыдущий снимок не был прочитан адресатом,
			// то он будет обновлён.
			select {
			case <-pollChan:
			default:
			}

			pollChan <- metrics.Snapshot()
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
		Host:   a.addr,
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

	if a.opts.SHA256Key != "" {
		s := sign.Signer(a.opts.SHA256Key)
		signed := s.Sign(buf.Bytes())
		hash := base64.RawURLEncoding.EncodeToString(signed)
		req.Header.Add(middleware.SignHeader, hash)
	}

	return req, nil
}

func (a *Agent) sendRequest(req *http.Request) error {
	ctx := req.Context()

	// NOTE: Если не удалось отправить запрос за установленное время,
	// то будет выполнено до 3-х попыток (с интервалами 1s, 3s и 5s),
	// прежде чем функция вернёт ошибку.
	for i := 1; i < 4; i++ {
		res, err := a.client.Do(req)
		if err == nil {
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
	io.Copy(io.Discard, res.Body)
	res.Body.Close()
}
