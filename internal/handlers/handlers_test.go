package handlers_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/sergeizaitcev/metrics/internal/handlers"
	"github.com/sergeizaitcev/metrics/internal/metrics"
	"github.com/sergeizaitcev/metrics/internal/storage"
	"github.com/sergeizaitcev/metrics/internal/storage/mocks"
)

func TestHandlers_ping(t *testing.T) {
	testCases := []struct {
		name      string
		mockError error
		wantCode  int
	}{
		{
			name:      "ok",
			mockError: nil,
			wantCode:  http.StatusOK,
		},
		{
			name:      "error",
			mockError: errors.New("error"),
			wantCode:  http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			storage := mocks.NewMockStorage()
			storage.On("Ping", mock.Anything).Return(tc.mockError)

			handler := handlers.New(storage)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/ping", nil)

			handler.ServeHTTP(rec, req)

			require.Equal(t, tc.wantCode, rec.Code)
		})
	}
}

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

			handler := handlers.New(storage)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)

			handler.ServeHTTP(rec, req)

			require.Equal(t, tc.wantCode, rec.Code)
			require.Equal(t, tc.wantBody, rec.Body.String())
		})
	}
}

func TestHandlers_update(t *testing.T) {
	testCases := []struct {
		name      string
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
			metric:    metrics.Counter("counter", 1),
			mockError: errors.New("error"),
			path:      "/update/counter/counter/1",
			wantCode:  http.StatusInternalServerError,
		},
		{
			name:     "gauge",
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
			metric:    metrics.Gauge("gauge", 1),
			mockError: errors.New("error"),
			path:      "/update/gauge/gauge/1",
			wantCode:  http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			storage := mocks.NewMockStorage()
			if tc.metric.Kind() != metrics.KindUnknown {
				switch tc.metric.Kind() {
				case metrics.KindCounter:
					storage.On("Add", mock.Anything, tc.metric).
						Return(metrics.Metric{}, tc.mockError)

				case metrics.KindGauge:
					storage.On("Set", mock.Anything, tc.metric).
						Return(metrics.Metric{}, tc.mockError)
				}
			}

			handler := handlers.New(storage)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, tc.path, nil)

			handler.ServeHTTP(rec, req)

			require.Equal(t, tc.wantCode, rec.Code)
		})
	}
}

func TestHandlers_updateV2(t *testing.T) {
	testCases := []struct {
		name       string
		metric     metrics.Metric
		mockMetric metrics.Metric
		mockError  error
		noHeader   bool
		body       string
		wantCode   int
		wantBody   string
	}{
		{
			name:     "unknown content type",
			noHeader: true,
			wantCode: http.StatusUnprocessableEntity,
		},
		{
			name:     "empty body",
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "empty counter",
			body:     `{}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:       "counter",
			metric:     metrics.Counter("test", 1),
			mockMetric: metrics.Counter("test", 2),
			body:       `{"type":"counter","id":"test","delta":1}`,
			wantCode:   http.StatusOK,
			wantBody:   `{"type":"counter","id":"test","delta":2}`,
		},
		{
			name:      "counter don't save",
			metric:    metrics.Counter("test", 1),
			mockError: errors.New("error"),
			body:      `{"type":"counter","id":"test","delta":1}`,
			wantCode:  http.StatusInternalServerError,
		},
		{
			name:       "gauge",
			metric:     metrics.Gauge("test", 1),
			mockMetric: metrics.Gauge("test", 2),
			body:       `{"type":"gauge","id":"test","value":1}`,
			wantCode:   http.StatusOK,
			wantBody:   `{"type":"gauge","id":"test","value":2}`,
		},
		{
			name:      "gauge don't save",
			metric:    metrics.Gauge("test", 1),
			mockError: errors.New("error"),
			body:      `{"type":"gauge","id":"test","value":1}`,
			wantCode:  http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			storage := mocks.NewMockStorage()
			if tc.metric.Kind() != metrics.KindUnknown {
				switch tc.metric.Kind() {
				case metrics.KindCounter:
					storage.On("Add", mock.Anything, tc.metric).
						Return(tc.mockMetric, tc.mockError)
				case metrics.KindGauge:
					storage.On("Set", mock.Anything, tc.metric).
						Return(tc.mockMetric, tc.mockError)
				}
			}

			handler := handlers.New(storage)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(tc.body))
			if !tc.noHeader {
				req.Header.Add("Content-Type", "application/json")
			}

			handler.ServeHTTP(rec, req)

			wantBody := tc.wantBody
			if wantBody != "" {
				wantBody += "\n"
			}

			require.Equal(t, tc.wantCode, rec.Code)
			require.Equal(t, wantBody, rec.Body.String())
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
			mockError: storage.ErrNotFound,
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
			mockError: storage.ErrNotFound,
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

			handler := handlers.New(storage)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)

			handler.ServeHTTP(rec, req)

			require.Equal(t, tc.wantCode, rec.Code)
			require.Equal(t, tc.wantBody, rec.Body.String())
		})
	}
}

func TestHandlers_getV2(t *testing.T) {
	testCases := []struct {
		name       string
		metric     string
		mockMetric metrics.Metric
		mockError  error
		body       string
		noHeader   bool
		wantCode   int
		wantBody   string
	}{
		{
			name:     "unknown content type",
			noHeader: true,
			wantCode: http.StatusUnprocessableEntity,
		},
		{
			name:     "empty body",
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "empty counter",
			body:     `{}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:       "counter",
			metric:     "test",
			mockMetric: metrics.Counter("test", 1),
			body:       `{"type":"counter","id":"test"}`,
			wantCode:   http.StatusOK,
			wantBody:   `{"type":"counter","id":"test","delta":1}`,
		},
		{
			name:      "counter not found",
			metric:    "test",
			mockError: storage.ErrNotFound,
			body:      `{"type":"counter","id":"test"}`,
			wantCode:  http.StatusNotFound,
		},
		{
			name:      "counter internal error",
			metric:    "test",
			mockError: errors.New("error"),
			body:      `{"type":"counter","id":"test"}`,
			wantCode:  http.StatusInternalServerError,
		},
		{
			name:       "counter not equal",
			metric:     "test",
			mockMetric: metrics.Gauge("test", 1),
			body:       `{"type":"counter","id":"test"}`,
			wantCode:   http.StatusNotFound,
		},
		{
			name:       "gauge",
			metric:     "test",
			mockMetric: metrics.Gauge("test", 1),
			body:       `{"type":"gauge","id":"test"}`,
			wantCode:   http.StatusOK,
			wantBody:   `{"type":"gauge","id":"test","value":1}`,
		},
		{
			name:      "gauge not found",
			metric:    "test",
			mockError: storage.ErrNotFound,
			body:      `{"type":"gauge","id":"test"}`,
			wantCode:  http.StatusNotFound,
		},
		{
			name:      "gauge internal error",
			metric:    "test",
			mockError: errors.New("error"),
			body:      `{"type":"gauge","id":"test"}`,
			wantCode:  http.StatusInternalServerError,
		},
		{
			name:       "gauge not equal",
			metric:     "test",
			mockMetric: metrics.Counter("test", 1),
			body:       `{"type":"gauge","id":"test"}`,
			wantCode:   http.StatusNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			storage := mocks.NewMockStorage()
			if tc.metric != "" {
				storage.On("Get", mock.Anything, tc.metric).Return(tc.mockMetric, tc.mockError)
			}

			handler := handlers.New(storage)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/value", strings.NewReader(tc.body))
			if !tc.noHeader {
				req.Header.Set("Content-Type", "application/json")
			}

			handler.ServeHTTP(rec, req)

			wantBody := tc.wantBody
			if wantBody != "" {
				wantBody += "\n"
			}

			require.Equal(t, tc.wantCode, rec.Code)
			require.Equal(t, wantBody, rec.Body.String())
		})
	}
}
