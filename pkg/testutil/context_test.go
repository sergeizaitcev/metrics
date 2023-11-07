package testutil_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/sergeizaitcev/metrics/pkg/testutil"
)

type mockTestingT struct {
	mock.Mock
}

func (m *mockTestingT) Deadline() (time.Time, bool) {
	args := m.Called()
	arg1 := args.Get(0).(time.Time)
	arg2 := args.Bool(1)
	return arg1, arg2
}

func (m *mockTestingT) Cleanup(f func()) {
	m.Called(f)
}

func TestContext(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		m := new(mockTestingT)
		m.On("Deadline").Return(time.Time{}, false)
		m.On("Cleanup", mock.AnythingOfType("func()"))

		ctx := testutil.Context(m)

		_, ok := ctx.Deadline()
		require.True(t, ok)
	})

	t.Run("with deadline", func(t *testing.T) {
		want := time.Now().Add(30 * time.Second)

		m := new(mockTestingT)
		m.On("Deadline").Return(want, true)
		m.On("Cleanup", mock.AnythingOfType("func()"))

		ctx := testutil.Context(m)

		got, ok := ctx.Deadline()
		require.True(t, ok)
		require.True(t, want.Equal(got))
	})
}
