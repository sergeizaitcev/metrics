package main

import (
	"flag"
	"os"
)

type flags struct {
	addr string
}

func parseFlags() (*flags, error) {
	var fs flags

	flag.StringVar(&fs.addr, "a", "localhost:8080", "server address")
	flag.Parse()

	addr := os.Getenv("ADDRESS")
	if addr != "" {
		fs.addr = addr
	}

	return &fs, nil
}
