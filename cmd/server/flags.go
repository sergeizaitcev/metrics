package main

import "flag"

type flags struct {
	addr string
}

func parseFlags() *flags {
	var fs flags

	flag.StringVar(&fs.addr, "a", "localhost:8080", "server address")
	flag.Parse()

	return &fs
}
