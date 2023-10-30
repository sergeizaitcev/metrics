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
	flagAddress         string
	flagDatabaseDSN     string
	flagFileStoragePath string
	flagSHA256Key       string
	flagStoreInterval   int64
	flagRestore         bool
)

func parseFlags() error {
	flags := flag.NewFlagSet("server", flag.ExitOnError)

	flags.StringVar(&flagAddress, "a", "localhost:8080", "server address")
	flags.StringVar(&flagDatabaseDSN, "d", "", "database dsn")
	flags.StringVar(&flagFileStoragePath, "f", "/tmp/metrics-db.wal", "file path storage")
	flags.StringVar(&flagSHA256Key, "k", "", "sha256 key")
	flags.Int64Var(&flagStoreInterval, "i", 300, "store interval in seconds")
	flags.BoolVar(&flagRestore, "r", true, "restore")

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

	databaseDSN := os.Getenv("DATABASE_DSN")
	if databaseDSN != "" {
		flagDatabaseDSN = databaseDSN
	}

	fileStoragePath := os.Getenv("FILE_STORAGE_PATH")
	if fileStoragePath != "" {
		flagFileStoragePath = fileStoragePath
	}

	key := os.Getenv("KEY")
	if key != "" {
		flagSHA256Key = key
	}

	storeInterval := os.Getenv("STORE_INTERVAL")
	if storeInterval != "" {
		v, err := strconv.ParseInt(storeInterval, 10, 64)
		if err != nil {
			return err
		}
		flagStoreInterval = v
	}
	if flagStoreInterval < 0 {
		return errors.New("store internval must be is greater or equal than zero")
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
