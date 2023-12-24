package sign_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sergeizaitcev/metrics/pkg/randutil"
	"github.com/sergeizaitcev/metrics/pkg/sign"
)

func TestSigner(t *testing.T) {
	key := randutil.Bytes(64)
	s := sign.Signer(key)
	signed := s.Sign(randutil.Bytes(1024))
	require.Len(t, signed, sign.SignLen)
}
