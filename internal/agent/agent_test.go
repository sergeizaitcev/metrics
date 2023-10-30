package agent_test

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/sergeizaitcev/metrics/internal/agent"
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

var _ net.Error = (*timeoutError)(nil)

type timeoutError string

func newTimeoutError(s string) error {
	err := timeoutError(s)
	return &err
}

func (err *timeoutError) Error() string {
	return string(*err)
}

func (err *timeoutError) Timeout() bool {
	return true
}

func (err *timeoutError) Temporary() bool {
	return false
}

func TestAgent(t *testing.T) {
	ctx := testutil.Context(t)

	t.Run("run", func(t *testing.T) {
		m := new(transportMock)
		opts := &agent.AgentOpts{
			ReportInterval: 100 * time.Millisecond,
			PollInterval:   60 * time.Millisecond,
			Transport:      m,
		}
		a := agent.New("localhost", opts)

		cancelCtx, cancel := context.WithTimeout(ctx, 110*time.Millisecond)
		defer cancel()

		m.On("RoundTrip", http.MethodPost, "/updates/").Return(http.StatusOK, nil)
		require.NoError(t, a.Run(cancelCtx))
	})

	t.Run("retry", func(t *testing.T) {
		m := new(transportMock)
		opts := &agent.AgentOpts{
			ReportInterval: 100 * time.Millisecond,
			PollInterval:   60 * time.Millisecond,
			Transport:      m,
		}
		a := agent.New("localhost", opts)

		cancelCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		errTimeout := newTimeoutError("error")

		m.On("RoundTrip", http.MethodPost, "/updates/").Return(0, errTimeout)
		require.Error(t, a.Run(cancelCtx))
	})
}
