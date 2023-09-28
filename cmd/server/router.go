package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/julienschmidt/httprouter"

	"github.com/sergeizaitcev/metrics/internal/metrics"
)

type handlers struct {
	metrics *metrics.Metrics
}

func newRouter(m *metrics.Metrics) http.Handler {
	handlers := handlers{metrics: m}

	router := httprouter.New()
	router.GET("/", handlers.all)
	router.GET("/value/:metric/:name", handlers.get)
	router.POST("/update/:metric/:name/:value", handlers.update)

	return router
}

func (h *handlers) all(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	ctx := r.Context()

	values, err := h.metrics.All(ctx)
	if err != nil {
		statusInternalServerError(w)
		return
	}

	statusOK(w)

	for _, value := range values {
		fmt.Fprintf(w, "%s=%s\n", value.Name(), value.String())
	}
}

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

	err := h.metrics.Save(ctx, metric)
	if err != nil {
		statusInternalServerError(w)
		return
	}

	statusOK(w)
}

func statusOK(w http.ResponseWriter) {
	status(w, http.StatusOK)
}

func statusNotFound(w http.ResponseWriter) {
	status(w, http.StatusNotFound)
}

func statusBadRequest(w http.ResponseWriter) {
	status(w, http.StatusBadRequest)
}

func statusInternalServerError(w http.ResponseWriter) {
	status(w, http.StatusInternalServerError)
}

func status(w http.ResponseWriter, code int) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
}
