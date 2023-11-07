package middleware

import (
	"bytes"
	"encoding/base64"
	"io"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

// SignHeader определяет заголовок с подписью.
const SignHeader = "HashSHA256"

// Signer представляет интерфейс подписи данных.
type Signer interface {
	Sign([]byte) []byte
}

// Sign проверяет подпись тела запроса.
func Sign(s Signer) Middleware {
	return func(h httprouter.Handle) httprouter.Handle {
		return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
			wantHash := r.Header.Get(SignHeader)
			if wantHash == "" {
				h(w, r, p)
				return
			}

			var (
				body []byte
				err  error
			)

			body, r.Body, err = readBody(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			hash := s.Sign(body)
			gotHash := base64.RawURLEncoding.EncodeToString(hash)

			if wantHash != gotHash {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			h(w, r, p)
		}
	}
}

func readBody(b io.ReadCloser) ([]byte, io.ReadCloser, error) {
	if b == nil || b == http.NoBody {
		return nil, http.NoBody, nil
	}

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(b); err != nil {
		return nil, b, err
	}
	if err := b.Close(); err != nil {
		return nil, b, err
	}

	return buf.Bytes(), io.NopCloser(bytes.NewReader(buf.Bytes())), nil
}
