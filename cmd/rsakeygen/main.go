package main

import (
	"context"
	"errors"
	"flag"
	"fmt"

	"github.com/sergeizaitcev/metrics/pkg/commands"
	"github.com/sergeizaitcev/metrics/pkg/rsautil"
)

var _ commands.Config = (*Config)(nil)

type Config struct {
	commands.UnimplementedConfig

	Bits   int
	Prefix string
}

func (c *Config) SetFlags(fs *flag.FlagSet) {
	fs.IntVar(&c.Bits, "b", 3072, "number of bits")
	fs.StringVar(&c.Prefix, "p", "key", "prefix filename")
}

func (c *Config) Validate() error {
	if !contains(c.Bits, []int{1024, 2048, 3072, 4096}) {
		return fmt.Errorf("number of bits is invalid: %d", c.Bits)
	}
	if c.Prefix == "" {
		return errors.New("filename must be non empty")
	}
	return nil
}

func contains(target int, src []int) bool {
	for _, want := range src {
		if want == target {
			return true
		}
	}
	return false
}

func main() {
	commands.Execute("rsakeygen", run)
}

func run(_ context.Context, c *Config) error {
	key, err := rsautil.Generate(c.Bits)
	if err != nil {
		return fmt.Errorf("generate rsa key: %w", err)
	}

	err = rsautil.Save(key, c.Prefix)
	if err != nil {
		return fmt.Errorf("writing a key to file: %w", err)
	}

	return nil
}
