package main

import (
	"errors"
	"flag"
	"net"
	"os"
	"strconv"
)

var (
	flagAddress        string
	flagSHA256Key      string
	flagReportInterval int64
	flagPollInterval   int64
)

func parseFlags() error {
	flags := flag.NewFlagSet("agent", flag.ExitOnError)

	flags.StringVar(&flagAddress, "a", "localhost:8080", "server address")
	flags.StringVar(&flagSHA256Key, "k", "", "sha256 key")
	flags.Int64Var(&flagReportInterval, "r", 10, "report interval in seconds")
	flags.Int64Var(&flagPollInterval, "p", 2, "poll interval in seconds")

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
		return err
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
		return errors.New("poll internval must be is greater than zero")
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
		return errors.New("report internval must be is greater than zero")
	}

	return nil
}
