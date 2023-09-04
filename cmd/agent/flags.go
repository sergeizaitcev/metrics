package main

import (
	"flag"
	"os"
	"strconv"
	"time"
)

type flags struct {
	addr           string
	reportInterval time.Duration
	pollInterval   time.Duration
}

func parseFlags() (*flags, error) {
	var fs flags
	var reportInterval, pollInterval int

	flag.StringVar(&fs.addr, "a", "localhost:8080", "server address")
	flag.IntVar(&reportInterval, "r", 10, "report interval in seconds")
	flag.IntVar(&pollInterval, "p", 2, "poll interval in seconds")

	flag.Parse()

	fs.reportInterval = time.Duration(reportInterval) * time.Second
	fs.pollInterval = time.Duration(pollInterval) * time.Second

	addr := os.Getenv("ADDRESS")
	if addr != "" {
		fs.addr = addr
	}

	poll := os.Getenv("POLL_INTERVAL")
	if poll != "" {
		v, err := strconv.ParseInt(poll, 10, 64)
		if err != nil {
			return nil, err
		}
		fs.pollInterval = time.Duration(v) * time.Second
	}

	report := os.Getenv("REPORT_INTERVAL")
	if report != "" {
		v, err := strconv.ParseInt(report, 10, 64)
		if err != nil {
			return nil, err
		}
		fs.reportInterval = time.Duration(v) * time.Second
	}

	return &fs, nil
}
