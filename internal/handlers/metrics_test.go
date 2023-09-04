package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sergeizaitcev/metrics/internal/handlers"
	"github.com/sergeizaitcev/metrics/internal/storage/local"
)

func TestMetrics(t *testing.T) {
	testCases := []struct {
		method   string
		path     string
		wantCode int
	}{
		{
			method:   http.MethodPost,
			path:     "/update/counter/testCounter/100",
			wantCode: http.StatusOK,
		},
		{
			method:   http.MethodPost,
			path:     "/update/counter/testCounter/none",
			wantCode: http.StatusBadRequest,
		},
		{
			method:   http.MethodPost,
			path:     "/update/gauge/testGauge/100",
			wantCode: http.StatusOK,
		},
		{
			method:   http.MethodPost,
			path:     "/update/gauge/testGauge/none",
			wantCode: http.StatusBadRequest,
		},
		{
			method:   http.MethodGet,
			path:     "/update/counter/testCounter/100",
			wantCode: http.StatusMethodNotAllowed,
		},
		{
			method:   http.MethodPost,
			path:     "/",
			wantCode: http.StatusNotFound,
		},
		{
			method:   http.MethodPost,
			path:     "/update/unknown",
			wantCode: http.StatusNotFound,
		},
		{
			method:   http.MethodPost,
			path:     "/delete/counter/testCounter/100",
			wantCode: http.StatusNotFound,
		},
		{
			method:   http.MethodPost,
			path:     "/update/unknown/test/100",
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.method+"_"+tc.path, func(t *testing.T) {
			storage := local.NewStorage()
			handler := handlers.NewMetrics(storage)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, tc.path, nil)

			handler.ServeHTTP(rec, req)

			require.Equal(t, tc.wantCode, rec.Code, rec.Body.String())
		})
	}
}
