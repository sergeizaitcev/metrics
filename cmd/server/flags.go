package main

import (
	"flag"
	"fmt"
	"net"
	"os"
)

var flagAddress string

func init() {
	flag.StringVar(&flagAddress, "a", "localhost:8080", "server address")
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

	return nil
}
