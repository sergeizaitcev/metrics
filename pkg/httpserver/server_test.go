package httpserver_test

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/sergeizaitcev/metrics/pkg/httpserver"
	"github.com/sergeizaitcev/metrics/pkg/tcputil"
	"github.com/sergeizaitcev/metrics/pkg/testutil"
)

func httpServer(t *testing.T) (srv *httpserver.Server, host string) {
	port, err := tcputil.FreePort()
	require.NoError(t, err)

	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	host = net.JoinHostPort("localhost", port)

	srv = httpserver.New(&http.Server{
		Addr:    host,
		Handler: mux,
	})

	return srv, host
}

func TestServer(t *testing.T) {
	srv, host := httpServer(t)
	errc := make(chan error)

	ctx, cancel := context.WithTimeout(testutil.Context(t), 3*time.Second)
	t.Cleanup(cancel)

	go func() { errc <- srv.ListenAndServe(ctx) }()
	go func() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
			u := &url.URL{
				Scheme: "http",
				Host:   host,
				Path:   "/test",
			}

			res, err := http.Get(u.String())
			require.NoError(t, err)

			_, _ = io.Copy(io.Discard, res.Body)
			_ = res.Body.Close()

			require.Equal(t, http.StatusOK, res.StatusCode)
		}
	}()

	select {
	case <-ctx.Done():
	case err := <-errc:
		require.NoError(t, err)
	}

	cancel()
	require.NoError(t, <-errc)
}
