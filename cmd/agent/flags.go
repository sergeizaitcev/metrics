package main

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/sergeizaitcev/metrics/internal/flagutil"
)

var (
	flagAddress        string
	flagReportInterval = flagutil.Second(10)
	flagPollInterval   = flagutil.Second(2)
)

func init() {
	flag.StringVar(&flagAddress, "a", "localhost:8080", "server address")
	flag.Var(&flagReportInterval, "r", "report interval in seconds")
	flag.Var(&flagPollInterval, "p", "poll interval in seconds")
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
		err = flagPollInterval.Set(poll)
		if err != nil {
			return err
		}
	}
	if flagPollInterval <= 0 {
		return errors.New("poll internval must be is greater than zero")
	}

	report := os.Getenv("REPORT_INTERVAL")
	if report != "" {
		err = flagReportInterval.Set(report)
		if err != nil {
			return err
		}
	}
	if flagReportInterval <= 0 {
		return errors.New("report internval must be is greater than zero")
	}

	return nil
}
