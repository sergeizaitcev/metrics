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
}

type metricsHandler struct {
	storage Storage
}

func NewMetrics(s Storage) http.Handler {
	m := &metricsHandler{storage: s}
	router := httprouter.New()

	router.POST("/update/:type/:name/:value", m.updateHandle)

	return router
}

func (m *metricsHandler) updateHandle(
	w http.ResponseWriter,
	r *http.Request,
	params httprouter.Params,
) {
	name := params.ByName("name")
	value := params.ByName("value")

	if name == "" || value == "" {
		statusBadRequest(w)
		return
	}

	switch params.ByName("type") {
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
