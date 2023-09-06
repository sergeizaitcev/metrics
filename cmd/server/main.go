package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/sergeizaitcev/metrics/internal/handlers"
	"github.com/sergeizaitcev/metrics/internal/storage/local"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	fs, err := parseFlags()
	if err != nil {
		return err
	}

	storage := local.NewStorage()
	handler := handlers.NewMetrics(storage)

	return http.ListenAndServe(fs.addr, handler)
}
