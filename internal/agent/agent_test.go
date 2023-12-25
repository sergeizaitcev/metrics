package agent_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/sergeizaitcev/metrics/internal/agent"
	"github.com/sergeizaitcev/metrics/internal/configs"
	"github.com/sergeizaitcev/metrics/pkg/logging"
	"github.com/sergeizaitcev/metrics/pkg/testutil"
)

var _ http.RoundTripper = (*transportMock)(nil)

type transportMock struct {
	mock.Mock
}

func (m *transportMock) RoundTrip(req *http.Request) (*http.Response, error) {
	args := m.Called(req.Method, req.URL.EscapedPath())

	err := args.Error(1)
	if err != nil {
		return nil, err
	}

	statusCode := args.Int(0)

	res := &http.Response{
		StatusCode: statusCode,
		Status:     http.StatusText(statusCode),
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(nil)),
		Request:    req,
	}

	return res, nil
}

func TestAgent(t *testing.T) {
	ctx := testutil.Context(t)

	t.Run("run", func(t *testing.T) {
		config := &configs.Agent{
			Address:        "localhost",
			ReportInterval: 100 * time.Millisecond,
			PollInterval:   60 * time.Millisecond,
			RateLimit:      1,
		}
		m := new(transportMock)
		opts := &agent.AgentOpts{
			Logger:    logging.New(os.Stdout, logging.LevelDebug),
			Transport: m,
		}
		a := agent.New(config, opts)

		cancelCtx, cancel := context.WithTimeout(ctx, 110*time.Millisecond)
		defer cancel()

		m.On("RoundTrip", http.MethodPost, "/updates/").Return(http.StatusOK, nil)
		require.NoError(t, a.Run(cancelCtx))
	})
}
