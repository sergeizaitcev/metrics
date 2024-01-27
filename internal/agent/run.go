package agent

import (
	"context"
	"crypto/rsa"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/sergeizaitcev/metrics/internal/agent/senders"
	"github.com/sergeizaitcev/metrics/internal/configs"
	"github.com/sergeizaitcev/metrics/pkg/logging"
	"github.com/sergeizaitcev/metrics/pkg/rsautil"
	"github.com/sergeizaitcev/metrics/pkg/tcputil"
)

// Run инициализирует агент сбора метрик и запускает его.
func Run(ctx context.Context, c *configs.Agent) (err error) {
	var key *rsa.PublicKey
	if c.PublicKeyPath != "" {
		key, err = rsautil.PublicKeyFrom(c.PublicKeyPath)
		if err != nil {
			return err
		}
	}

	logger := logging.New(os.Stdout, c.Level)
	ip := tcputil.Local()
	opts := []senders.Option{
		senders.WithEncrypt(key),
		senders.WithLogger(logger),
		senders.WithIP(ip.String()),
		senders.WithSHA256Key(c.SHA256Key),
	}

	var sender senders.Sender
	if c.GRPCEnabled {
		creds := insecure.NewCredentials()
		conn, err := grpc.Dial(c.Address, grpc.WithTransportCredentials(creds))
		if err != nil {
			return err
		}
		defer conn.Close()

		sender = senders.GRPC(conn, opts...)
	} else {
		sender = senders.HTTP(c.Address, opts...)
	}

	agent := NewAgent(sender, c)
	agent.SetLogger(logger)

	return agent.Run(ctx)
}
