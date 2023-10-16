package middleware

import (
	"compress/flate"
	"compress/gzip"
	"net/http"
	"strings"

	"github.com/julienschmidt/httprouter"
)

// Gzip сжимает исходящий контент для типов supportCTypes с уровнем сжатия level.
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
			} else {
				next(w, r, p)
			}
		}
	}
}

type gzipResponseWriter struct {
	http.ResponseWriter
	gw     *gzip.Writer
	ctypes []string
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
