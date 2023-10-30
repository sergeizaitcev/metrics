package middleware

import (
	"bytes"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
)

// Params определяет параметры запроса.
type Params struct {
	URI        string
	Method     string
	Duration   time.Duration
	StatusCode int
	Body       []byte
	Error      error
}

// Trace передает параметры запроса в paramsFunc.
func Trace(paramsFunc func(*Params)) Middleware {
	return func(next httprouter.Handle) httprouter.Handle {
		return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
			rw := &traceResponseWriter{
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

			paramsFunc(&Params{
				URI:        reqURI,
				Method:     r.Method,
				Duration:   elapsed,
				StatusCode: rw.statusCode,
				Body:       rw.body.Bytes(),
				Error:      rw.err,
			})
		}
	}
}

type traceResponseWriter struct {
	http.ResponseWriter
	body       bytes.Buffer
	statusCode int
	err        error
}

// WriteError записывает ошибку в w.
func WriteError(w http.ResponseWriter, err error) {
	rw, ok := w.(*traceResponseWriter)
	if ok {
		rw.err = err
	}
}

func (w *traceResponseWriter) Write(p []byte) (int, error) {
	w.body.Write(p)
	return w.ResponseWriter.Write(p)
}

func (w *traceResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}
