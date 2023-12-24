package middleware_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/require"

	"github.com/sergeizaitcev/metrics/pkg/middleware"
)

func TestTrace(t *testing.T) {
	testCases := []struct {
		name       string
		method     string
		path       string
		statusCode int
		body       []byte
	}{
		{
			name:       "get",
			method:     http.MethodGet,
			path:       "/example/1",
			statusCode: http.StatusOK,
			body:       []byte("test"),
		},
		{
			name:       "post",
			method:     http.MethodPost,
			path:       "/example/2",
			statusCode: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		handler := func(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
			middleware.WriteError(w, errors.New("error"))
			w.WriteHeader(tc.statusCode)
			_, _ = w.Write(tc.body)
		}

		paramsChan := make(chan middleware.Params, 1)
		paramsFunc := func(params *middleware.Params) { paramsChan <- *params }
		trace := middleware.Use(handler, middleware.Trace(paramsFunc))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(tc.method, tc.path, nil)

		trace(rec, req, httprouter.Params{})
		params := <-paramsChan

		require.Equal(t, tc.path, params.Path)
		require.Equal(t, tc.method, params.Method)
		require.NotEmpty(t, params.Elapsed)
		require.Equal(t, tc.statusCode, params.StatusCode)
		require.Equal(t, tc.body, params.Body)
		require.Error(t, params.Error)
	}
}
