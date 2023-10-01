package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/require"
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
			w.WriteHeader(tc.statusCode)
			w.Write(tc.body)
		}

		paramsChan := make(chan params, 1)
		paramsFunc := func(params *params) { paramsChan <- *params }
		trace := use(handler, trace(paramsFunc))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(tc.method, tc.path, nil)

		trace(rec, req, httprouter.Params{})
		params := <-paramsChan

		require.Equal(t, tc.path, params.uri)
		require.Equal(t, tc.method, params.method)
		require.NotEmpty(t, params.duration)
		require.Equal(t, tc.statusCode, params.statusCode)
		require.Equal(t, tc.body, params.body)
	}
}
