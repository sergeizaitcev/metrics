package main

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/sergeizaitcev/metrics/internal/flagutil"
)

var (
	flagAddress         string
	flagStoreInterval   = flagutil.Second(300)
	flagFileStoragePath string
	flagRestore         bool
)

func init() {
	flag.StringVar(&flagAddress, "a", "localhost:8080", "server address")
	flag.Var(&flagStoreInterval, "i", "store interval in seconds (0 - sync write)")
	flag.StringVar(&flagFileStoragePath, "f", "/tmp/metrics-db.json", "file path storage")
	flag.BoolVar(&flagRestore, "r", true, "restore")
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

	storeInterval := os.Getenv("STORE_INTERVAL")
	if storeInterval != "" {
		err := flagStoreInterval.Set(storeInterval)
		if err != nil {
			return err
		}
	}
	if flagStoreInterval < 0 {
		return errors.New("store internval must be is greater or equal than zero")
	}

	fileStoragePath := os.Getenv("FILE_STORAGE_PATH")
	if fileStoragePath != "" {
		flagFileStoragePath = fileStoragePath
	}

	restore := os.Getenv("RESTORE")
	if restore != "" {
		b, err := strconv.ParseBool(restore)
		if err != nil {
			return err
		}
		flagRestore = b
	}

	return nil
}
