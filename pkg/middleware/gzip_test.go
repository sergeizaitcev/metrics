package middleware_test

import (
	"bytes"
	"compress/flate"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/require"

	"github.com/sergeizaitcev/metrics/pkg/middleware"
)

func TestGzip(t *testing.T) {
	testCases := []struct {
		name     string
		ctype    string
		noHeader bool
		want     int
	}{
		{
			name:  "encoding: json",
			ctype: "application/json",
			want:  50,
		},
		{
			name:  "encoding: html",
			ctype: "text/html",
			want:  50,
		},
		{
			name:  "encoding: text",
			ctype: "text/plain",
			want:  50,
		},
		{
			name:  "encoding: unsupport type",
			ctype: "application/octet-stream",
			want:  8000,
		},
		{
			name:     "no encoding: json",
			ctype:    "application/json",
			noHeader: true,
			want:     8000,
		},
		{
			name:     "no encoding: html",
			ctype:    "test/html",
			noHeader: true,
			want:     8000,
		},
		{
			name:     "no encoding: text",
			ctype:    "text/plain",
			noHeader: true,
			want:     8000,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler := func(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
				w.Header().Set("Content-Type", tc.ctype)
				w.WriteHeader(http.StatusOK)
				w.Write(bytes.Repeat([]byte("test"), 2000))
			}

			gzip := middleware.Use(handler, middleware.Gzip(flate.BestCompression))

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if !tc.noHeader {
				req.Header.Set("Accept-Encoding", "gzip")
			}

			gzip(rec, req, httprouter.Params{})
			require.Equal(t, tc.want, rec.Body.Len())
		})
	}
}
