package server_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/stretchr/testify/mock"

	"github.com/sergeizaitcev/metrics/internal/metrics"
	"github.com/sergeizaitcev/metrics/internal/server"
	"github.com/sergeizaitcev/metrics/internal/storage/mocks"
)

func ExampleHandler_ping() {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", http.NoBody)

	storage := mocks.NewMockStorage()
	storage.On("Ping", mock.Anything).Return(nil)

	h := server.NewHandler(storage)
	h.ServeHTTP(rec, req)

	fmt.Println(rec.Code, http.StatusText(rec.Code))

	// Output:
	// 200 OK
}

func ExampleHandler_all() {
	values := []metrics.Metric{
		metrics.Counter("counter", 1),
		metrics.Gauge("gauge", 1),
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)

	storage := mocks.NewMockStorage()
	storage.On("GetAll", mock.Anything).Return(values, nil)

	h := server.NewHandler(storage)
	h.ServeHTTP(rec, req)

	fmt.Println("Content-Type:", rec.Header().Get("Content-Type"))
	fmt.Println("X-Content-Type-Options:", rec.Header().Get("X-Content-Type-Options"))
	fmt.Println(rec.Code, http.StatusText(rec.Code))
	fmt.Printf("\n%s", rec.Body.String())

	// Output:
	// Content-Type: text/html; charset=utf-8
	// X-Content-Type-Options: nosniff
	// 200 OK
	//
	// counter=1
	// gauge=1
}

func ExampleHandler_get() {
	value := metrics.Counter("my_counter", 1)
	body, _ := json.Marshal(&value)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/value", bytes.NewReader(body))
	req.Header.Add("Content-Type", "application/json")

	storage := mocks.NewMockStorage()
	storage.On("Get", mock.Anything, value.Name()).Return(value, nil)

	h := server.NewHandler(storage)
	h.ServeHTTP(rec, req)

	fmt.Println("Content-Type:", rec.Header().Get("Content-Type"))
	fmt.Println("X-Content-Type-Options:", rec.Header().Get("X-Content-Type-Options"))
	fmt.Println(rec.Code, http.StatusText(rec.Code))
	fmt.Printf("\n%s", rec.Body.String())

	// Output:
	// Content-Type: application/json; charset=utf-8
	// X-Content-Type-Options: nosniff
	// 200 OK
	//
	// {"type":"counter","id":"my_counter","delta":1}
}

func ExampleHandler_update() {
	values := []metrics.Metric{
		metrics.Counter("my_counter", 1),
		metrics.Gauge("my_gauge", 1),
	}
	body, _ := json.Marshal(values)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(body))
	req.Header.Add("Content-Type", "application/json")

	storage := mocks.NewMockStorage()
	storage.On("Save", mock.Anything, values).Return(([]metrics.Metric)(nil), nil)

	h := server.NewHandler(storage)
	h.ServeHTTP(rec, req)

	fmt.Println(rec.Code, http.StatusText(rec.Code))

	// Output:
	// 200 OK
}
