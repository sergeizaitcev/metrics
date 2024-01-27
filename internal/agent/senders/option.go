package senders

import (
	"crypto/rsa"

	"github.com/sergeizaitcev/metrics/pkg/logging"
)

type Option func(*commonOptions)

type commonOptions struct {
	key       *rsa.PublicKey
	logger    *logging.Logger
	ip        string
	sha256key string
}

func WithEncrypt(key *rsa.PublicKey) Option {
	return func(opt *commonOptions) {
		opt.key = key
	}
}

func WithLogger(logger *logging.Logger) Option {
	return func(opt *commonOptions) {
		opt.logger = logger
	}
}

func WithIP(ip string) Option {
	return func(opt *commonOptions) {
		opt.ip = ip
	}
}

func WithSHA256Key(key string) Option {
	return func(opt *commonOptions) {
		opt.sha256key = key
	}
}
