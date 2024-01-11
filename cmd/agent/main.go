package main

import (
	"fmt"
	"os"

	"github.com/sergeizaitcev/metrics/internal/agent"
	"github.com/sergeizaitcev/metrics/version"
)

func main() {
	version.Print()
	if err := agent.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
