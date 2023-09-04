package main

import (
	"flag"
	"time"
)

type flags struct {
	addr           string
	reportInterval time.Duration
	pollInterval   time.Duration
}

func parseFlags() *flags {
	var fs flags
	var reportInterval, pollInterval int

	flag.StringVar(&fs.addr, "a", "localhost:8080", "server address")
	flag.IntVar(&reportInterval, "r", 10, "report interval in seconds")
	flag.IntVar(&pollInterval, "p", 2, "poll interval in seconds")

	flag.Parse()

	fs.reportInterval = time.Duration(reportInterval) * time.Second
	fs.pollInterval = time.Duration(pollInterval) * time.Second

	return &fs
}
