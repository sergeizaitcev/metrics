package main

import (
	"fmt"
	"os"

	"github.com/sergeizaitcev/metrics/internal/server"
)

func main() {
	if err := server.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
