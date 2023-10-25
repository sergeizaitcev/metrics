package testutil

import (
	"context"
	"time"
)

type TestingT interface {
	Deadline() (time.Time, bool)
	Cleanup(func())
}

func Context(t TestingT) context.Context {
	deadline, ok := t.Deadline()
	if !ok {
		deadline = time.Now().Add(15 * time.Second)
	}

	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	t.Cleanup(cancel)

	return ctx
}
