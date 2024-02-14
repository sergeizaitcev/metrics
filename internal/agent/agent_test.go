package agent_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/sergeizaitcev/metrics/internal/agent"
	"github.com/sergeizaitcev/metrics/internal/configs"
	"github.com/sergeizaitcev/metrics/internal/metrics"
	"github.com/sergeizaitcev/metrics/pkg/testutil"
)

var _ agent.Sender = (*senderMock)(nil)

type senderMock struct {
	mock.Mock
}

func (m *senderMock) Send(ctx context.Context, values []metrics.Metric) error {
	args := m.Called(ctx, values)
	return args.Error(0)
}

func TestAgent(t *testing.T) {
	ctx := testutil.Context(t)

	t.Run("run", func(t *testing.T) {
		config := &configs.Agent{
			Address:        "localhost",
			ReportInterval: 150 * time.Millisecond,
			PollInterval:   100 * time.Millisecond,
			RateLimit:      1,
		}

		values := []metrics.Metric{
			metrics.Counter("counter", 1),
			metrics.Gauge("gauge", 1.1),
		}

		sender := new(senderMock)
		sender.On("Send", mock.Anything, values).Return(nil)

		a := agent.NewAgent(sender, config)

		cancelCtx, cancel := context.WithTimeout(ctx, 180*time.Millisecond)
		defer cancel()

		require.NoError(t, a.Run(cancelCtx))
	})
}
