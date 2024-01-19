package main

import (
	"github.com/sergeizaitcev/metrics/internal/agent"
	"github.com/sergeizaitcev/metrics/pkg/commands"
	"github.com/sergeizaitcev/metrics/version"
)

func main() {
	version.Print()
	commands.Execute("agent", agent.Run)
}
