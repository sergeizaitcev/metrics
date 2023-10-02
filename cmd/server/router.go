package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/julienschmidt/httprouter"

	"github.com/sergeizaitcev/metrics/internal/metrics"
	"github.com/sergeizaitcev/metrics/pkg/middleware"
)

type handlers struct {
	metrics *metrics.Metrics
}

func newRouter(m *metrics.Metrics, middlewares ...middleware.Middleware) http.Handler {
	handlers := handlers{metrics: m}

	router := httprouter.New()
	router.GET("/", middleware.Use(handlers.all, middlewares...))

	router.GET("/value/:metric/:name", middleware.Use(handlers.get, middlewares...))
	router.POST("/value", middleware.Use(handlers.getV2, middlewares...))

	router.POST("/update/:metric/:name/:value", middleware.Use(handlers.update, middlewares...))
	router.POST("/update", middleware.Use(handlers.updateV2, middlewares...))

	return router
}

func (h *handlers) all(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	ctx := r.Context()

	values, err := h.metrics.All(ctx)
	if err != nil {
		statusHTML(w, http.StatusInternalServerError)
		return
	}

	statusHTML(w, http.StatusOK)

	for _, value := range values {
		fmt.Fprintf(w, "%s=%s\n", value.Name(), value.String())
	}
}

// Deprecated: используется для обратной совместимости.
func (h *handlers) get(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	kind := metrics.ParseKind(p.ByName("metric"))
	if kind == metrics.KindUnknown {
		statusBadRequest(w)
		return
	}

	ctx := r.Context()

	metric, err := h.metrics.Lookup(ctx, p.ByName("name"))
	if errors.Is(err, metrics.ErrNotFound) {
		statusNotFound(w)
		return
	}
	if err != nil {
		statusInternalServerError(w)
		return
	}

	statusOK(w)
	fmt.Fprintln(w, metric.String())
}

func (h *handlers) getV2(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	ctype := r.Header.Get("Content-Type")
	if !strings.Contains(ctype, "application/json") {
		statusUnprocessableEntity(w)
		return
	}

	var metric metrics.Metric

	err := json.NewDecoder(r.Body).Decode(&metric)
	if err != nil || metric.IsEmpty() {
		statusBadRequest(w)
		return
	}

	ctx := r.Context()

	actual, err := h.metrics.Lookup(ctx, metric.Name())
	if errors.Is(err, metrics.ErrNotFound) {
		statusNotFound(w)
		return
	}
	if err != nil {
		statusInternalServerError(w)
		return
	}
	if metric.Kind() != actual.Kind() {
		statusNotFound(w)
		return
	}

	statusJSON(w, http.StatusOK)
	json.NewEncoder(w).Encode(&actual)
}

// Deprecated: используется для обратной совместимости.
func (h *handlers) update(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	kind := metrics.ParseKind(p.ByName("metric"))
	if kind == metrics.KindUnknown {
		statusBadRequest(w)
		return
	}

	name := p.ByName("name")
	value := p.ByName("value")

	var metric metrics.Metric

	switch kind {
	case metrics.KindCounter:
		v, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			statusBadRequest(w)
			return
		}
		metric = metrics.Counter(name, v)
	case metrics.KindGauge:
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			statusBadRequest(w)
			return
		}
		metric = metrics.Gauge(name, v)
	}

	ctx := r.Context()

	_, err := h.metrics.Save(ctx, metric)
	if err != nil {
		statusInternalServerError(w)
		return
	}

	statusOK(w)
}

func (h *handlers) updateV2(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	ctype := r.Header.Get("Content-Type")
	if !strings.Contains(ctype, "application/json") {
		statusUnprocessableEntity(w)
		return
	}

	var metric metrics.Metric

	err := json.NewDecoder(r.Body).Decode(&metric)
	if err != nil || metric.IsEmpty() {
		statusBadRequest(w)
		return
	}

	ctx := r.Context()

	actual, err := h.metrics.Save(ctx, metric)
	if err != nil {
		statusInternalServerError(w)
		return
	}

	statusJSON(w, http.StatusOK)
	json.NewEncoder(w).Encode(&actual)
}

func statusOK(w http.ResponseWriter) {
	statusText(w, http.StatusOK)
}

func statusNotFound(w http.ResponseWriter) {
	statusText(w, http.StatusNotFound)
}

func statusBadRequest(w http.ResponseWriter) {
	statusText(w, http.StatusBadRequest)
}

func statusUnprocessableEntity(w http.ResponseWriter) {
	statusText(w, http.StatusUnprocessableEntity)
}

func statusInternalServerError(w http.ResponseWriter) {
	statusText(w, http.StatusInternalServerError)
}

func statusText(w http.ResponseWriter, code int) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
}

func statusHTML(w http.ResponseWriter, code int) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
}

func statusJSON(w http.ResponseWriter, code int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
}
