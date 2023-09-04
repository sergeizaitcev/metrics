package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/julienschmidt/httprouter"

	"github.com/sergeizaitcev/metrics/internal/metrics"
)

// Storage представляет интерфейс хранилища метрик.
type Storage interface {
	// Set устанавливает новое значение метрики и возвращает предыдущее.
	Set(context.Context, metrics.Metric) (metrics.Metric, error)

	// Add увеличивает значение метрики и возвращает итоговый результат.
	Add(context.Context, metrics.Metric) (metrics.Metric, error)

	// Get возвращает метрику.
	Get(context.Context, string) (metrics.Metric, error)
}

type metricsHandler struct {
	storage Storage
}

func NewMetrics(s Storage) http.Handler {
	m := &metricsHandler{storage: s}
	router := httprouter.New()

	router.POST("/update/:metric/:name/:value", m.updateHandle)
	router.GET("/value/:metric/:name", m.getHandle)

	return router
}

func (m *metricsHandler) updateHandle(
	w http.ResponseWriter,
	r *http.Request,
	params httprouter.Params,
) {
	name := params.ByName("name")
	value := params.ByName("value")

	switch params.ByName("metric") {
	case "counter":
		m.addCounter(w, name, value)
	case "gauge":
		m.setGauge(w, name, value)
	default:
		statusBadRequest(w)
	}
}

func (m *metricsHandler) addCounter(w http.ResponseWriter, name, value string) {
	v, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		statusBadRequest(w)
		return
	}

	metric := metrics.Counter(name, v)

	_, err = m.storage.Add(context.Background(), metric)
	if err != nil {
		statusInternalServerError(w)
		return
	}

	statusOK(w)
}

func (m *metricsHandler) setGauge(w http.ResponseWriter, name, value string) {
	v, err := strconv.ParseFloat(value, 64)
	if err != nil {
		statusBadRequest(w)
		return
	}

	metric := metrics.Gauge(name, v)

	_, err = m.storage.Set(context.Background(), metric)
	if err != nil {
		statusInternalServerError(w)
		return
	}

	statusOK(w)
}

func (m *metricsHandler) getHandle(
	w http.ResponseWriter,
	r *http.Request,
	params httprouter.Params,
) {
	switch params.ByName("metric") {
	case "counter", "gauge":
		m.getMetric(w, params.ByName("name"))
	default:
		statusBadRequest(w)
	}
}

func (m *metricsHandler) getMetric(w http.ResponseWriter, name string) {
	metric, err := m.storage.Get(context.Background(), name)
	if err != nil {
		statusInternalServerError(w)
		return
	}

	if metric.Kind() == metrics.KindUnknown {
		statusNotFound(w)
		return
	}

	status(w, http.StatusOK, metric.String())
}

func statusOK(w http.ResponseWriter) {
	status(w, http.StatusOK, "200 OK")
}

func statusNotFound(w http.ResponseWriter) {
	status(w, http.StatusNotFound, "404 not found")
}

func statusBadRequest(w http.ResponseWriter) {
	status(w, http.StatusBadRequest, "400 bad request")
}

func statusMethodNotAllowed(w http.ResponseWriter) {
	status(w, http.StatusMethodNotAllowed, "405 method not allowed")
}

func statusInternalServerError(w http.ResponseWriter) {
	status(w, http.StatusInternalServerError, "500 internal server error")
}

func status(w http.ResponseWriter, code int, a ...any) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
	fmt.Fprintln(w, a...)
}
