package handlers

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

// New возвращает новый обработчик HTTP-запросов.
func New(s storage.Storager, middlewares ...middleware.Middleware) http.Handler {
	router := httprouter.New()
	router.GET("/ping", ping(s))

	for _, h := range []struct {
		method string
		path   string
		handle func(storage.Storager) httprouter.Handle
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
			path:   "/updates/",
			handle: updateV3,
		},
	} {
		handle := middleware.Use(h.handle(s), middlewares...)
		router.Handle(h.method, h.path, handle)
	}

	return router
}

func ping(s storage.Storager) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx := r.Context()
		if err := s.Ping(ctx); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

func all(s storage.Storager) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx := r.Context()

		values, err := s.GetAll(ctx)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusOK)

		for _, value := range values {
			fmt.Fprintf(w, "%s=%s\n", value.Name(), value.Str())
		}
	}
}

// Deprecated: используется для обратной совместимости.
func get(s storage.Storager) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		kind := metrics.ParseKind(p.ByName("metric"))
		if kind == metrics.KindUnknown {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		ctx := r.Context()

		metric, err := s.Get(ctx, p.ByName("name"))
		if errors.Is(err, storage.ErrNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusOK)

		fmt.Fprintln(w, metric.Str())
	}
}

func getV2(s storage.Storager) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctype := r.Header.Get("Content-Type")
		if !strings.Contains(ctype, "application/json") {
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}

		var metric metrics.Metric

		err := json.NewDecoder(r.Body).Decode(&metric)
		if err != nil || metric.IsEmpty() {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		ctx := r.Context()

		actual, err := s.Get(ctx, metric.Name())
		if errors.Is(err, storage.ErrNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
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
func update(s storage.Storager) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		kind := metrics.ParseKind(p.ByName("metric"))
		if kind == metrics.KindUnknown {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		name := p.ByName("name")
		value := p.ByName("value")

		var metric metrics.Metric

		switch kind {
		case metrics.KindCounter:
			v, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			metric = metrics.Counter(name, v)
		case metrics.KindGauge:
			v, err := strconv.ParseFloat(value, 64)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			metric = metrics.Gauge(name, v)
		}

		ctx := r.Context()

		_, err := s.Save(ctx, metric)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

// Deprecated: используется для обратной совместимости.
func updateV2(s storage.Storager) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctype := r.Header.Get("Content-Type")
		if !strings.Contains(ctype, "application/json") {
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}

		var metric metrics.Metric

		err := json.NewDecoder(r.Body).Decode(&metric)
		if err != nil || metric.IsEmpty() {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		ctx := r.Context()

		actual, err := s.Save(ctx, metric)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
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

func updateV3(s storage.Storager) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctype := r.Header.Get("Content-Type")
		if !strings.Contains(ctype, "application/json") {
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}

		var values []metrics.Metric

		err := json.NewDecoder(r.Body).Decode(&values)
		if err != nil || len(values) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		ctx := r.Context()

		_, err = s.Save(ctx, values...)
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
		}

		w.WriteHeader(http.StatusOK)
	}
}
