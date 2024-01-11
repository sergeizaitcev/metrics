package main

import (
	"fmt"
	"os"

	"github.com/sergeizaitcev/metrics/internal/server"
	"github.com/sergeizaitcev/metrics/version"
)

func main() {
	version.Print()
	if err := server.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
