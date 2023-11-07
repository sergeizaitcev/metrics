package closer_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/sergeizaitcev/metrics/pkg/closer"
)

func TestCloser(t *testing.T) {
	var cnt atomic.Int32

	wantErr := errors.New("error")

	closeFunc1 := func() { cnt.Add(1) }
	closeFunc2 := func() error { cnt.Add(1); return wantErr }
	closeFunc3 := func(context.Context) error { cnt.Add(1); return nil }

	c := closer.New()
	defer func() {
		err := c.Close()
		require.ErrorIs(t, err, wantErr)
		require.EqualValues(t, 3, cnt.Load())
	}()

	ctx := context.Background()

	c.Add(ctx, closeFunc1)
	c.Add(ctx, closeFunc2)
	c.Add(ctx, closeFunc3)

	time.Sleep(100 * time.Millisecond)
}

func TestCloser_panic(t *testing.T) {
	ctx := context.Background()
	c := closer.New()
	require.Panics(t, func() { c.Add(ctx, nil) })
	require.Panics(t, func() { c.Add(ctx, (func())(nil)) })
	require.Panics(t, func() { c.Add(ctx, func() int { return 0 }) })
}
