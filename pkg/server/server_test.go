package server_test

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/nettest"

	"github.com/sergeizaitcev/metrics/pkg/server"
	"github.com/sergeizaitcev/metrics/pkg/testutil"
)

func TestServer(t *testing.T) {
	lis, err := nettest.NewLocalListener("tcp")
	require.NoError(t, err)
	t.Cleanup(func() { lis.Close() })

	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	opts := &server.ServerOpts{
		Listener: lis,
	}

	s := server.New(mux, opts)
	errc := make(chan error)

	ctx, cancel := context.WithTimeout(testutil.Context(t), 3*time.Second)
	t.Cleanup(cancel)

	go func() { errc <- s.ListenAndServe(ctx) }()
	go func() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
			u := &url.URL{
				Scheme: "http",
				Host:   lis.Addr().String(),
				Path:   "/test",
			}

			res, err := http.Get(u.String())
			require.NoError(t, err)

			io.Copy(io.Discard, res.Body)
			res.Body.Close()

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
