package rsautil_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sergeizaitcev/metrics/pkg/rsautil"
)

var (
	privateKey = filepath.Join("testdata", "test.rsa")
	publicKey  = filepath.Join("testdata", "test.rsa.pub")
)

func TestRSA(t *testing.T) {
	key, err := rsautil.Private(privateKey)
	require.NoError(t, err)
	require.NotNil(t, key)

	pub, err := rsautil.Public(publicKey)
	require.NoError(t, err)
	require.NotNil(t, pub)

	want := []byte("testtesttest")

	cipherText, err := rsautil.Encrypt(pub, want)
	require.NoError(t, err)
	require.NotEqual(t, want, cipherText)

	got, err := rsautil.Decrypt(key, cipherText)
	require.NoError(t, err)
	require.Equal(t, want, got)
}
