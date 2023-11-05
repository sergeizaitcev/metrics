package main

import (
	"fmt"
	"os"

	"github.com/sergeizaitcev/metrics/internal/agent"
)

func main() {
	if err := agent.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
