package middleware_test

import (
	"bytes"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/sergeizaitcev/metrics/pkg/middleware"
	"github.com/sergeizaitcev/metrics/pkg/randutil"
)

type mockSigner struct {
	mock.Mock
}

func (m *mockSigner) Sign(b []byte) []byte {
	args := m.Called(b)
	return args.Get(0).([]byte)
}

func TestSign(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		w.WriteHeader(http.StatusOK)
		io.Copy(w, r.Body)
	}

	t.Run("no sign", func(t *testing.T) {
		m := new(mockSigner)
		sign := middleware.Use(handler, middleware.Sign(m))

		wantBody := randutil.String(64)

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(wantBody))

		sign(rec, req, httprouter.Params{})

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, wantBody, rec.Body.String())

		m.AssertNotCalled(t, "Sign")
	})

	t.Run("success", func(t *testing.T) {
		hash := randutil.String(32)
		bhash, err := base64.RawURLEncoding.DecodeString(hash)
		require.NoError(t, err)

		wantBody := randutil.Bytes(10)

		m := new(mockSigner)
		m.On("Sign", wantBody).Return(bhash)

		sign := middleware.Use(handler, middleware.Sign(m))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(wantBody))
		req.Header.Add(middleware.SignHeader, hash)

		sign(rec, req, httprouter.Params{})

		require.Equal(t, rec.Code, http.StatusOK)
		require.Equal(t, wantBody, rec.Body.Bytes())
	})

	t.Run("invalid", func(t *testing.T) {
		data := randutil.Bytes(10)

		m := new(mockSigner)
		m.On("Sign", data).Return(randutil.Bytes(32))

		sign := middleware.Use(handler, middleware.Sign(m))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(data))
		req.Header.Add(middleware.SignHeader, randutil.String(32))

		sign(rec, req, httprouter.Params{})

		require.Equal(t, rec.Code, http.StatusBadRequest)
	})
}
