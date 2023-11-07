package closer

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"
)

const cancelTimeout = 5 * time.Second

type unsupportedError struct {
	a any
}

func (err *unsupportedError) Error() string {
	return fmt.Sprintf("close func is unsupported: %T", err.a)
}

// Closer определяет фоновый финишер процессов.
type Closer struct {
	wg       sync.WaitGroup
	termChan chan struct{}

	firstErrOnce sync.Once
	firstErr     error
}

// New возвращает новый экземпляр Closer.
func New() *Closer {
	return &Closer{
		termChan: make(chan struct{}),
	}
}

// Closer дожидается завершения всех добавленных фоновых процессов
// и возвращает первую ошибку.
func (c *Closer) Close() error {
	close(c.termChan)
	c.wg.Wait()
	c.firstErrOnce.Do(func() {
		c.firstErr = nil
	})
	return c.firstErr
}

// Add добавляет новый финишер.
func (c *Closer) Add(ctx context.Context, closeFunc any) {
	fn := convertCloseFunc(closeFunc)

	c.wg.Add(1)

	go func() {
		defer c.wg.Done()

		select {
		case <-ctx.Done():
		case <-c.termChan:
		}

		closeCtx, closeCancel := context.WithTimeout(context.Background(), cancelTimeout)
		defer closeCancel()

		err := fn(closeCtx)
		if err != nil {
			c.firstErrOnce.Do(func() {
				c.firstErr = err
			})
		}
	}()
}

func convertCloseFunc(closeFunc any) func(context.Context) error {
	if closeFunc == nil {
		panic(&unsupportedError{closeFunc})
	}

	rv := reflect.ValueOf(closeFunc)
	if rv.IsNil() || rv.Type().Kind() != reflect.Func {
		panic(&unsupportedError{closeFunc})
	}

	switch impl := rv.Interface().(type) {
	case func():
		return func(context.Context) error { impl(); return nil }
	case func() error:
		return func(context.Context) error { return impl() }
	case func(context.Context) error:
		return impl
	default:
		panic(&unsupportedError{closeFunc})
	}
}
