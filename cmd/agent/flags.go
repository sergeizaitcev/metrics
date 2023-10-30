package main

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
)

var (
	flagAddress        string
	flagSHA256Key      string
	flagReportInterval int64
	flagPollInterval   int64
	flagRateLimit      int
)

func parseFlags() error {
	flags := flag.NewFlagSet("agent", flag.ExitOnError)

	flags.StringVar(&flagAddress, "a", "localhost:8080", "server address")
	flags.StringVar(&flagSHA256Key, "k", "", "sha256 key")
	flags.Int64Var(&flagReportInterval, "r", 10, "report interval in seconds")
	flags.Int64Var(&flagPollInterval, "p", 2, "poll interval in seconds")
	flags.IntVar(&flagRateLimit, "l", 1, "rate limit")

	err := flags.Parse(os.Args[1:])
	if err != nil {
		flags.Usage()
	}

	addr := os.Getenv("ADDRESS")
	if addr != "" {
		flagAddress = addr
	}
	_, _, err = net.SplitHostPort(flagAddress)
	if err != nil {
		return fmt.Errorf("invalid address: %w", err)
	}

	key := os.Getenv("KEY")
	if key != "" {
		flagSHA256Key = key
	}

	poll := os.Getenv("POLL_INTERVAL")
	if poll != "" {
		v, err := strconv.ParseInt(poll, 10, 64)
		if err != nil {
			return err
		}
		flagPollInterval = v
	}
	if flagPollInterval <= 0 {
		return errors.New("poll interval must be is greater than zero")
	}

	report := os.Getenv("REPORT_INTERVAL")
	if report != "" {
		v, err := strconv.ParseInt(report, 10, 64)
		if err != nil {
			return err
		}
		flagReportInterval = v
	}
	if flagReportInterval <= 0 {
		return errors.New("report interval must be is greater than zero")
	}

	limit := os.Getenv("RATE_LIMIT")
	if limit != "" {
		v, err := strconv.ParseInt(limit, 10, 64)
		if err != nil {
			return err
		}
		flagRateLimit = int(v)
	}
	if flagRateLimit <= 0 {
		return errors.New("rate limit must be is greater than zero")
	}

	return nil
}
