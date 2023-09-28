package main

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"
)

var (
	flagAddress        string
	flagReportInterval = duration(10 * time.Second)
	flagPollInterval   = duration(2 * time.Second)
)

func init() {
	flag.StringVar(&flagAddress, "a", "localhost:8080", "server address")
	flag.Var(&flagReportInterval, "r", "report interval in seconds")
	flag.Var(&flagPollInterval, "p", "poll interval in seconds")
}

var _ flag.Value = (*duration)(nil)

type duration time.Duration

func (d duration) String() string {
	return time.Duration(d).String()
}

func (d *duration) Set(value string) error {
	v, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return err
	}
	if v <= 0 {
		return errors.New("value must be is greater than zero")
	}
	*d = duration(time.Duration(v) * time.Second)
	return nil
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

	report := os.Getenv("REPORT_INTERVAL")
	if report != "" {
		err = flagReportInterval.Set(report)
		if err != nil {
			return err
		}
	}

	return nil
}
