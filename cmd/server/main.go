package main

import (
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/sergeizaitcev/metrics/internal/handlers"
	"github.com/sergeizaitcev/metrics/internal/storage/local"
)

const (
	host = "localhost"
	port = "8080"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	storage := local.NewStorage()
	handler := handlers.NewMetrics(storage)
	addr := net.JoinHostPort(host, port)
	return http.ListenAndServe(addr, handler)
}
