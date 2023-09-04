package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/sergeizaitcev/metrics/internal/metrics"
)

// Storage представляет интерфейс хранилища метрик.
type Storage interface {
	// Set устанавливает новое значение метрики и возвращает предыдущее.
	Set(context.Context, metrics.Metric) (metrics.Metric, error)

	// Add увеличивает значение метрики и возвращает итоговый результат.
	Add(context.Context, metrics.Metric) (metrics.Metric, error)
}

type metricsHandler struct {
	storage Storage
}

func NewMetrics(s Storage) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/update/", &metricsHandler{storage: s})
	return mux
}

func (m *metricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		statusMethodNotAllowed(w)
		return
	}

	urlpath := strings.TrimSuffix(r.URL.Path, "/")
	path := strings.Split(urlpath[1:], "/")

	if len(path) < 4 {
		http.NotFound(w, r)
		return
	}
	if len(path) > 4 || path[1] == "" || path[2] == "" || path[3] == "" {
		statusBadRequest(w)
		return
	}

	switch path[1] {
	case "counter":
		m.counterHandle(w, path)
	case "gauge":
		m.gaugeHandle(w, path)
	default:
		statusBadRequest(w)
	}
}

func (m *metricsHandler) counterHandle(w http.ResponseWriter, path []string) {
	value, err := strconv.ParseInt(path[3], 10, 64)
	if err != nil {
		statusBadRequest(w)
		return
	}

	metric := metrics.Counter(path[2], value)

	_, err = m.storage.Add(context.Background(), metric)
	if err != nil {
		statusInternalServerError(w)
		return
	}

	statusOK(w)
}

func (m *metricsHandler) gaugeHandle(w http.ResponseWriter, path []string) {
	value, err := strconv.ParseFloat(path[3], 64)
	if err != nil {
		statusBadRequest(w)
		return
	}

	metric := metrics.Gauge(path[2], value)

	_, err = m.storage.Set(context.Background(), metric)
	if err != nil {
		statusInternalServerError(w)
		return
	}

	statusOK(w)
}

func statusOK(w http.ResponseWriter) {
	status(w, "200 OK", http.StatusOK)
}

func statusBadRequest(w http.ResponseWriter) {
	status(w, "400 bad request", http.StatusBadRequest)
}

func statusMethodNotAllowed(w http.ResponseWriter) {
	status(w, "405 method not allowed", http.StatusMethodNotAllowed)
}

func statusInternalServerError(w http.ResponseWriter) {
	status(w, "500 internal server error", http.StatusInternalServerError)
}

func status(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
	fmt.Fprintln(w, msg)
}
