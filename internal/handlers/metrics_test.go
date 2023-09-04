package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sergeizaitcev/metrics/internal/handlers"
	"github.com/sergeizaitcev/metrics/internal/metrics"
	"github.com/sergeizaitcev/metrics/internal/storage/local"
)

func TestMetrics_update(t *testing.T) {
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

func TestMetrics_get(t *testing.T) {
	testCases := []struct {
		method   string
		path     string
		wantCode int
		wantBody string
	}{
		{
			method:   http.MethodGet,
			path:     "/value/counter/testCounter",
			wantCode: http.StatusOK,
			wantBody: "1\n",
		},
		{
			method:   http.MethodGet,
			path:     "/value/gauge/testGauge",
			wantCode: http.StatusOK,
			wantBody: "1\n",
		},
		{
			method:   http.MethodGet,
			path:     "/value/unknown/unknown",
			wantCode: http.StatusBadRequest,
		},
		{
			method:   http.MethodGet,
			path:     "/value/counter/unknown",
			wantCode: http.StatusNotFound,
		},
		{
			method:   http.MethodPost,
			path:     "/value/counter/testCounter",
			wantCode: http.StatusMethodNotAllowed,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.method+"_"+tc.path, func(t *testing.T) {
			storage := local.NewStorage()
			storage.Set(context.Background(), metrics.Counter("testCounter", 1))
			storage.Set(context.Background(), metrics.Gauge("testGauge", 1))

			handler := handlers.NewMetrics(storage)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, tc.path, nil)

			handler.ServeHTTP(rec, req)

			require.Equal(t, tc.wantCode, rec.Code)

			if rec.Code == http.StatusOK {
				require.Equal(t, tc.wantBody, rec.Body.String())
			}
		})
	}
}

func TestMetrics_all(t *testing.T) {
	ctx := context.Background()

	storage := local.NewStorage()
	storage.Set(ctx, metrics.Counter("testCounter", 1))
	storage.Set(ctx, metrics.Counter("testCounter2", 2))
	storage.Set(ctx, metrics.Gauge("testGauge", 1))
	storage.Set(ctx, metrics.Gauge("testGauge2", 2))
	storage.Set(ctx, metrics.Gauge("testGauge3", 3))

	handler := handlers.NewMetrics(storage)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	t.Logf("\n%s", rec.Body.String())
}
