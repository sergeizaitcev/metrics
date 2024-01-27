package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/julienschmidt/httprouter"

	"github.com/sergeizaitcev/metrics/internal/metrics"
	"github.com/sergeizaitcev/metrics/internal/storage"
	"github.com/sergeizaitcev/metrics/pkg/middleware"
)

var (
	errMetricEmpty            = errors.New("metric is empty")
	errMetricUnknown          = errors.New("metric kind is unknown")
	errContentTypeUnsupported = errors.New("content type is unsupported")
)

// New возвращает новый обработчик HTTP-запросов.
func NewHandler(s storage.Storage, middlewares ...middleware.Middleware) http.Handler {
	router := &httprouter.Router{
		HandleMethodNotAllowed: true,
		HandleOPTIONS:          true,
	}

	router.GET("/ping", ping(s))

	for _, h := range []struct {
		method string
		path   string
		handle func(storage.Storage) httprouter.Handle
	}{
		{
			method: http.MethodGet,
			path:   "/",
			handle: all,
		},
		{
			method: http.MethodGet,
			path:   "/value/:metric/:name",
			handle: get,
		},
		{
			method: http.MethodPost,
			path:   "/value",
			handle: getV2,
		},
		{
			method: http.MethodPost,
			path:   "/value/", // NOTE: в iter14 проверяется редирект.

			handle: getV2,
		},
		{
			method: http.MethodPost,
			path:   "/update/:metric/:name/:value",
			handle: update,
		},
		{
			method: http.MethodPost,
			path:   "/update",
			handle: updateV2,
		},
		{
			method: http.MethodPost,
			path:   "/update/", // NOTE: в iter14 проверяется редирект.
			handle: updateV2,
		},
		{
			method: http.MethodPost,
			path:   "/updates/",
			handle: updateV3,
		},
	} {
		handle := middleware.Use(h.handle(s), middlewares...)
		router.Handle(h.method, h.path, handle)
	}

	return router
}

func ping(s storage.Storage) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx := r.Context()
		if err := s.Ping(ctx); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

func all(s storage.Storage) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx := r.Context()

		values, err := s.GetAll(ctx)
		if err != nil {
			sendError(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusOK)

		for _, value := range values {
			fmt.Fprintf(w, "%s=%s\n", value.Name(), value.String())
		}
	}
}

// Deprecated: используется для обратной совместимости.
func get(s storage.Storage) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		kind := metrics.ParseKind(p.ByName("metric"))
		if kind == metrics.KindUnknown {
			sendError(w, http.StatusBadRequest, errMetricUnknown)
			return
		}

		ctx := r.Context()

		metric, err := s.Get(ctx, p.ByName("name"))
		if errors.Is(err, storage.ErrNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if err != nil {
			sendError(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusOK)

		fmt.Fprintln(w, metric.String())
	}
}

func getV2(s storage.Storage) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctype := r.Header.Get("Content-Type")
		if !strings.Contains(ctype, "application/json") {
			sendError(w, http.StatusUnprocessableEntity, errContentTypeUnsupported)
			return
		}

		var metric metrics.Metric

		err := json.NewDecoder(r.Body).Decode(&metric)
		if err != nil {
			sendError(w, http.StatusBadRequest, err)
			return
		}
		if metric.IsEmpty() {
			sendError(w, http.StatusBadRequest, errMetricEmpty)
			return
		}

		ctx := r.Context()

		actual, err := s.Get(ctx, metric.Name())
		if errors.Is(err, storage.ErrNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if err != nil {
			sendError(w, http.StatusInternalServerError, err)
			return
		}
		if metric.Kind() != actual.Kind() {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusOK)

		json.NewEncoder(w).Encode(&actual)
	}
}

// Deprecated: используется для обратной совместимости.
func update(s storage.Storage) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		kind := metrics.ParseKind(p.ByName("metric"))
		if kind == metrics.KindUnknown {
			sendError(w, http.StatusBadRequest, errMetricUnknown)
			return
		}

		name := p.ByName("name")
		value := p.ByName("value")

		var metric metrics.Metric

		switch kind {
		case metrics.KindCounter:
			v, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				sendError(w, http.StatusBadRequest, fmt.Errorf("parse int: %s", err))
				return
			}
			metric = metrics.Counter(name, v)
		case metrics.KindGauge:
			v, err := strconv.ParseFloat(value, 64)
			if err != nil {
				sendError(w, http.StatusBadRequest, fmt.Errorf("parse float: %s", err))
				return
			}
			metric = metrics.Gauge(name, v)
		}

		ctx := r.Context()

		_, err := s.Save(ctx, metric)
		if err != nil {
			sendError(w, http.StatusInternalServerError, err)
		}
	}
}

// Deprecated: используется для обратной совместимости.
func updateV2(s storage.Storage) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctype := r.Header.Get("Content-Type")
		if !strings.Contains(ctype, "application/json") {
			sendError(w, http.StatusUnprocessableEntity, errContentTypeUnsupported)
			return
		}

		var metric metrics.Metric

		err := json.NewDecoder(r.Body).Decode(&metric)
		if err != nil {
			sendError(w, http.StatusBadRequest, err)
			return
		}
		if metric.IsEmpty() {
			sendError(w, http.StatusBadRequest, errMetricEmpty)
			return
		}

		ctx := r.Context()

		actual, err := s.Save(ctx, metric)
		if err != nil {
			sendError(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusOK)

		if len(actual) > 0 {
			json.NewEncoder(w).Encode(&actual[0])
		}
	}
}

func updateV3(s storage.Storage) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctype := r.Header.Get("Content-Type")
		if !strings.Contains(ctype, "application/json") {
			sendError(w, http.StatusUnprocessableEntity, errContentTypeUnsupported)
			return
		}

		var values []metrics.Metric

		err := json.NewDecoder(r.Body).Decode(&values)
		if err != nil {
			sendError(w, http.StatusBadRequest, err)
			return
		}
		if len(values) == 0 {
			sendError(w, http.StatusBadRequest, errMetricEmpty)
			return
		}

		ctx := r.Context()

		_, err = s.Save(ctx, values...)
		if err != nil {
			sendError(w, http.StatusInternalServerError, err)
		}
	}
}

func sendError(w http.ResponseWriter, code int, err error) {
	middleware.WriteError(w, err)
	w.WriteHeader(code)
}
