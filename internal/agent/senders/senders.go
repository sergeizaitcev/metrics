package senders

import (
	"context"

	"github.com/sergeizaitcev/metrics/internal/metrics"
)

// Sender описывает интерфейс отправителя метрик на сервер.
type Sender interface {
	// Send отправляет метрики на сервер.
	Send(ctx context.Context, values []metrics.Metric) error
}
