package middleware

import (
	"compress/flate"
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"github.com/julienschmidt/httprouter"
)

// Gzip сжимает исходящий и входящий контент с уровнем сжатия level.
func Gzip(level int, supportCTypes ...string) Middleware {
	if level < flate.NoCompression && level > flate.BestCompression {
		level = flate.NoCompression
	}
	if len(supportCTypes) == 0 {
		supportCTypes = append(supportCTypes, "application/json", "text/html", "text/plain")
	}
	for i := range supportCTypes {
		supportCTypes[i] = strings.ToLower(supportCTypes[i])
	}

	return func(next httprouter.Handle) httprouter.Handle {
		return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
			decoding := r.Header.Get("Content-Encoding")

			if strings.Contains(decoding, "gzip") {
				gr, err := gzip.NewReader(r.Body)
				if err != nil {
					w.WriteHeader(http.StatusUnsupportedMediaType)
					return
				}

				r.Body = &gzipReader{
					body: r.Body,
					gr:   gr,
				}
			} else if decoding != "" {
				w.WriteHeader(http.StatusUnsupportedMediaType)
				return
			}

			encoding := r.Header.Get("Accept-Encoding")

			if (strings.Contains(encoding, "gzip") || strings.Contains(encoding, "*")) &&
				level > flate.NoCompression {
				gw, _ := gzip.NewWriterLevel(w, level)
				rw := &gzipResponseWriter{
					ResponseWriter: w,
					gw:             gw,
					ctypes:         supportCTypes,
				}

				next(rw, r, p)
				if rw.checkCType() {
					rw.gw.Close()
				}

				return
			}

			next(w, r, p)
		}
	}
}

type gzipReader struct {
	body io.ReadCloser
	gr   *gzip.Reader
}

func (r *gzipReader) Read(p []byte) (n int, err error) {
	return r.gr.Read(p)
}

func (r *gzipReader) Close() error {
	var firstErr error
	err := r.body.Close()
	if err != nil {
		firstErr = err
	}
	err = r.gr.Close()
	if err != nil && firstErr == nil {
		firstErr = err
	}
	return err
}

type gzipResponseWriter struct {
	http.ResponseWriter
	gw     *gzip.Writer
	ctypes []string
	cnt    int
}

func (w *gzipResponseWriter) WriteHeader(statusCode int) {
	if w.checkCType() {
		w.ResponseWriter.Header().Set("Content-Encoding", "gzip")
	}
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *gzipResponseWriter) Write(p []byte) (int, error) {
	if w.checkCType() {
		return w.gw.Write(p)
	}
	return w.ResponseWriter.Write(p)
}

func (w *gzipResponseWriter) checkCType() bool {
	ctype := strings.ToLower(w.ResponseWriter.Header().Get("Content-Type"))
	for _, target := range w.ctypes {
		if strings.Contains(ctype, target) {
			return true
		}
	}
	return false
}
