package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/sergeizaitcev/metrics/internal/metrics"
	"github.com/sergeizaitcev/metrics/internal/metrics/mocks"
)

func TestHandlers_all(t *testing.T) {
	testCases := []struct {
		name        string
		mockMetrics []metrics.Metric
		mockError   error
		wantCode    int
		wantBody    string
	}{
		{
			name: "ok",
			mockMetrics: []metrics.Metric{
				metrics.Counter("counter", 1),
				metrics.Gauge("gauge", 1),
			},
			wantCode: http.StatusOK,
			wantBody: "counter=1\ngauge=1\n",
		},
		{
			name:      "internal error",
			mockError: errors.New("error"),
			wantCode:  http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			storage := mocks.NewMockStorage()
			storage.On("GetAll", mock.Anything).Return(tc.mockMetrics, tc.mockError)

			router := newRouter(metrics.NewMetrics(storage))

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)

			router.ServeHTTP(rec, req)

			require.Equal(t, tc.wantCode, rec.Code)
			require.Equal(t, tc.wantBody, rec.Body.String())
			require.Equal(t, "text/plain; charset=utf-8", rec.Header().Get("Content-Type"))
		})
	}
}

func TestHandlers_update(t *testing.T) {
	testCases := []struct {
		name      string
		method    string
		metric    metrics.Metric
		mockError error
		path      string
		wantCode  int
	}{
		{
			name:     "unknown kind",
			path:     "/update/unknown/counter/1",
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "counter",
			method:   "Add",
			metric:   metrics.Counter("counter", 1),
			path:     "/update/counter/counter/1",
			wantCode: http.StatusOK,
		},
		{
			name:     "counter not parse",
			path:     "/update/counter/counter/1.01",
			wantCode: http.StatusBadRequest,
		},
		{
			name:      "counter don't save",
			method:    "Add",
			metric:    metrics.Counter("counter", 1),
			mockError: errors.New("error"),
			path:      "/update/counter/counter/1",
			wantCode:  http.StatusInternalServerError,
		},
		{
			name:     "gauge",
			method:   "Set",
			metric:   metrics.Gauge("gauge", 1),
			path:     "/update/gauge/gauge/1",
			wantCode: http.StatusOK,
		},
		{
			name:     "gauge not parse",
			path:     "/update/gauge/gauge/none",
			wantCode: http.StatusBadRequest,
		},
		{
			name:      "gauge don't save",
			method:    "Set",
			metric:    metrics.Gauge("gauge", 1),
			mockError: errors.New("error"),
			path:      "/update/gauge/gauge/1",
			wantCode:  http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			storage := mocks.NewMockStorage()
			if tc.method != "" {
				storage.On(tc.method, mock.Anything, tc.metric).
					Return(metrics.Metric{}, tc.mockError)
			}

			router := newRouter(metrics.NewMetrics(storage))

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, tc.path, nil)

			router.ServeHTTP(rec, req)

			require.Equal(t, tc.wantCode, rec.Code)
		})
	}
}

func TestHandlers_get(t *testing.T) {
	testCases := []struct {
		name       string
		metric     string
		mockMetric metrics.Metric
		mockError  error
		path       string
		wantCode   int
		wantBody   string
	}{
		{
			name:     "unknown kind",
			path:     "/value/unknown/counter",
			wantCode: http.StatusBadRequest,
		},
		{
			name:       "counter",
			metric:     "counter",
			mockMetric: metrics.Counter("counter", 1),
			path:       "/value/counter/counter",
			wantCode:   http.StatusOK,
			wantBody:   "1\n",
		},
		{
			name:      "counter not found",
			metric:    "counter",
			mockError: metrics.ErrNotFound,
			path:      "/value/counter/counter",
			wantCode:  http.StatusNotFound,
		},
		{
			name:      "counter internal error",
			metric:    "counter",
			mockError: errors.New("error"),
			path:      "/value/counter/counter",
			wantCode:  http.StatusInternalServerError,
		},
		{
			name:       "gauge",
			metric:     "gauge",
			mockMetric: metrics.Gauge("gauge", 1),
			path:       "/value/gauge/gauge",
			wantCode:   http.StatusOK,
			wantBody:   "1\n",
		},
		{
			name:      "gauge not found",
			metric:    "gauge",
			mockError: metrics.ErrNotFound,
			path:      "/value/gauge/gauge",
			wantCode:  http.StatusNotFound,
		},
		{
			name:      "gauge internal error",
			metric:    "gauge",
			mockError: errors.New("error"),
			path:      "/value/gauge/gauge",
			wantCode:  http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			storage := mocks.NewMockStorage()
			if tc.metric != "" {
				storage.On("Get", mock.Anything, tc.metric).Return(tc.mockMetric, tc.mockError)
			}

			router := newRouter(metrics.NewMetrics(storage))

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)

			router.ServeHTTP(rec, req)

			require.Equal(t, tc.wantCode, rec.Code)
			require.Equal(t, tc.wantBody, rec.Body.String())
		})
	}
}
