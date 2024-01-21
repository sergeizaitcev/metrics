package main

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/sergeizaitcev/metrics/pkg/commands"
	"github.com/sergeizaitcev/metrics/pkg/randutil"
)

var _ commands.Config = (*Config)(nil)

type Config struct {
	commands.UnimplementedConfig

	Bits     int
	Filename string
}

func (c *Config) SetFlags(fs *flag.FlagSet) {
	fs.IntVar(&c.Bits, "b", 3072, "number of bits")
	fs.StringVar(&c.Filename, "f", "key", "key filename")
}

func (c *Config) Validate() error {
	if !contains(c.Bits, []int{1024, 2048, 3072, 4096}) {
		return fmt.Errorf("number of bits is invalid: %d", c.Bits)
	}
	if c.Filename == "" {
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
	key, err := rsa.GenerateKey(randutil.Rand, c.Bits)
	if err != nil {
		return fmt.Errorf("generate rsa key: %w", err)
	}

	pub := key.Public()

	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})

	pubPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: x509.MarshalPKCS1PublicKey(pub.(*rsa.PublicKey)),
	})

	filename := strings.TrimSpace(c.Filename)

	err = os.WriteFile(filename+".rsa", keyPEM, 0o600)
	if err != nil {
		return fmt.Errorf("write a private key to file: %w", err)
	}

	err = os.WriteFile(filename+".rsa.pub", pubPEM, 0o644)
	if err != nil {
		return fmt.Errorf("write a public key to file: %w", err)
	}

	return nil
}
