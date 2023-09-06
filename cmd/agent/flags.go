package main

import (
	"flag"
	"fmt"
	"net"
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
	var reportInterval, pollInterval int64

	flag.StringVar(&fs.addr, "a", "localhost:8080", "server address")
	flag.Int64Var(&reportInterval, "r", 10, "report interval in seconds")
	flag.Int64Var(&pollInterval, "p", 2, "poll interval in seconds")

	flag.Parse()

	addr := os.Getenv("ADDRESS")
	if addr != "" {
		if _, _, err := net.SplitHostPort(addr); err != nil {
			return nil, fmt.Errorf("main: ADDRESS is invalid: %s", err)
		}
		fs.addr = addr
	}

	poll := os.Getenv("POLL_INTERVAL")
	if poll != "" {
		v, err := strconv.ParseInt(poll, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("main: POLL_INTERVAL is invalid: %s", err)
		}
		pollInterval = v
	}

	report := os.Getenv("REPORT_INTERVAL")
	if report != "" {
		v, err := strconv.ParseInt(report, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("main: REPORT_INTERVAL is invalid: %s", err)
		}
		reportInterval = v
	}

	fs.reportInterval = time.Duration(reportInterval) * time.Second
	fs.pollInterval = time.Duration(pollInterval) * time.Second

	return &fs, nil
}
