package main

import (
	"github.com/sergeizaitcev/metrics/internal/server"
	"github.com/sergeizaitcev/metrics/pkg/commands"
	"github.com/sergeizaitcev/metrics/version"
)

func main() {
	version.Print()
	commands.Execute("server", server.Run)
}
