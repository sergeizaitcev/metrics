package middleware

import (
	"bytes"
	"crypto/rsa"
	"io"
	"net/http"

	"github.com/julienschmidt/httprouter"

	"github.com/sergeizaitcev/metrics/pkg/rsautil"
)

// RSA дешифрует входящий контент при помощи приватного RSA ключа.
func RSA(key *rsa.PrivateKey) Middleware {
	return func(h httprouter.Handle) httprouter.Handle {
		return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
			var cipherText []byte
			var err error

			cipherText, r.Body, err = readBody(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			body, err := rsautil.Decrypt(key, cipherText)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			r.Body = io.NopCloser(bytes.NewReader(body))
			h(w, r, p)
		}
	}
}
