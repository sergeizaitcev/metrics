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
	flagReportInterval int64
	flagPollInterval   int64
)

func init() {
	flag.StringVar(&flagAddress, "a", "localhost:8080", "server address")
	flag.Int64Var(&flagReportInterval, "r", 10, "report interval in seconds")
	flag.Int64Var(&flagPollInterval, "p", 2, "poll interval in seconds")
}

func parseFlags() (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("%s", e)
		}
	}()

	err = flag.CommandLine.Parse(os.Args[1:])
	if err != nil {
		return err
	}

	addr := os.Getenv("ADDRESS")
	if addr != "" {
		flagAddress = addr
	}

	_, _, err = net.SplitHostPort(flagAddress)
	if err != nil {
		return err
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
