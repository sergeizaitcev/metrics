package main

import (
	"bytes"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
)

type middleware func(httprouter.Handle) httprouter.Handle

func use(handler httprouter.Handle, middlewares ...middleware) httprouter.Handle {
	for _, middleware := range middlewares {
		handler = middleware(handler)
	}
	return handler
}

type params struct {
	uri        string
	method     string
	duration   time.Duration
	statusCode int
	body       []byte
}

func trace(paramsFunc func(*params)) middleware {
	return func(next httprouter.Handle) httprouter.Handle {
		return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
			rw := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			start := time.Now()
			next(rw, r, p)
			elapsed := time.Since(start)

			reqURI := r.RequestURI
			if reqURI == "" {
				reqURI = r.URL.RequestURI()
			}

			paramsFunc(&params{
				uri:        reqURI,
				method:     r.Method,
				duration:   elapsed,
				statusCode: rw.statusCode,
				body:       rw.body.Bytes(),
			})
		}
	}
}

type responseWriter struct {
	http.ResponseWriter
	body       bytes.Buffer
	statusCode int
}

func (w *responseWriter) Write(p []byte) (int, error) {
	w.body.Write(p)
	return w.ResponseWriter.Write(p)
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}
