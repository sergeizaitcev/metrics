package middleware_test

import (
	"bytes"
	"crypto/rsa"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sergeizaitcev/metrics/pkg/middleware"
	"github.com/sergeizaitcev/metrics/pkg/rsautil"
)

func TestCrypto(t *testing.T) {
	key, err := rsautil.Generate(2048)
	require.NoError(t, err)

	encrypt := func(t *testing.T, message []byte) []byte {
		pub := key.Public().(*rsa.PublicKey)
		cypher, err := rsautil.Encrypt(pub, message)
		require.NoError(t, err)
		return cypher
	}

	testCases := []struct {
		name       string
		message    []byte
		cypherText []byte
		want       int
	}{
		{
			name:       "success",
			message:    []byte("success"),
			cypherText: encrypt(t, []byte("success")),
			want:       http.StatusOK,
		},
		{
			name:       "empty",
			cypherText: encrypt(t, nil),
			want:       http.StatusOK,
		},
		{
			name:       "invalid",
			cypherText: []byte("invalid"),
			want:       http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			next := func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
				w.WriteHeader(http.StatusOK)
				_, _ = io.Copy(w, r.Body)
			}

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", bytes.NewReader(tc.cypherText))

			crypto := middleware.Use(next, middleware.RSA(key))
			crypto(rec, req, httprouter.Params{})

			if assert.Equal(t, tc.want, rec.Code) {
				assert.Equal(t, tc.message, rec.Body.Bytes())
			}
		})
	}
}
