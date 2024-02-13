package agent

import (
	"context"
	"sync"
	"time"

	"github.com/sergeizaitcev/metrics/internal/configs"
	"github.com/sergeizaitcev/metrics/internal/metrics"
	"github.com/sergeizaitcev/metrics/pkg/logging"
)

// Sender описывает интерфейс отправителя метрик на сервер.
type Sender interface {
	// Send отправляет метрики на сервер.
	Send(ctx context.Context, values []metrics.Metric) error
}

// Agent определяет агент для сбора и отправки метрик на сервер.
type Agent struct {
	sender         Sender
	logger         *logging.Logger
	pollInterval   time.Duration
	reportInterval time.Duration
	rateLimit      int
}

// NewAgent инициализирует и возвращает новый экземпляр Agent.
func NewAgent(sender Sender, config *configs.Agent) *Agent {
	agent := &Agent{
		sender:         sender,
		logger:         logging.Discard(),
		pollInterval:   config.PollInterval,
		reportInterval: config.ReportInterval,
		rateLimit:      config.RateLimit,
	}
	return agent
}

// SetLogger устанавливает логгер для агента.
func (a *Agent) SetLogger(logger *logging.Logger) {
	a.logger = logger
}

// Run собирает метрики и отправляет их на сервер; блокируется до тех пор, пока
// не сработает контекст или функция не вернёт ошибку.
func (a *Agent) Run(ctx context.Context) error {
	var wg sync.WaitGroup

	collectChan := make(chan []metrics.Metric, a.rateLimit)
	a.collect(ctx, collectChan)

	for i := 0; i < a.rateLimit; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				case snapshot := <-collectChan:
					if len(snapshot) == 0 {
						a.logger.Log(logging.LevelError, "metrics is empty")
						continue
					}

					a.logger.Log(logging.LevelDebug, "sending a new metrics batch",
						"batch_size", len(snapshot),
					)

					start := time.Now()
					err := a.sender.Send(ctx, snapshot)
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

// collect собирает метрики и передаёт их в канал collectChan.
func (a *Agent) collect(ctx context.Context, collectChan chan<- []metrics.Metric) {
	go func() {
		pollChan := a.poll(ctx)

		ticker := time.NewTicker(a.reportInterval)
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
		ticker := time.NewTicker(a.pollInterval)
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
