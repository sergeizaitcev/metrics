package rsautil_test

import (
	"crypto/rsa"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/sergeizaitcev/metrics/pkg/rsautil"
	"github.com/sergeizaitcev/metrics/testdata"
)

type CryptoSuite struct {
	suite.Suite

	private *rsa.PrivateKey
	public  *rsa.PublicKey

	privateKey []byte
	publicKey  []byte
}

func TestRSA(t *testing.T) {
	suite.Run(t, new(CryptoSuite))
}

func (suite *CryptoSuite) SetupSuite() {
	suite.privateKey = testdata.Private
	suite.publicKey = testdata.Public
}

func (suite *CryptoSuite) TestA_Decode() {
	var err error

	suite.Run("private", func() {
		suite.private, err = rsautil.PrivateKey(suite.privateKey)
		suite.NoError(err)
		suite.NotNil(suite.private)
	})

	suite.Run("public", func() {
		suite.public, err = rsautil.PublicKey(suite.publicKey)
		suite.NoError(err)
		suite.NotNil(suite.public)
	})
}

func (suite *CryptoSuite) TestB_EncryptingMessage() {
	var cipher []byte
	var err error

	message := []byte("message")

	suite.Run("encrypt", func() {
		cipher, err = rsautil.Encrypt(suite.public, message)
		suite.NoError(err)
		suite.NotEqual(message, cipher)
	})

	suite.Run("decrypt", func() {
		got, err := rsautil.Decrypt(suite.private, cipher)
		suite.NoError(err)
		suite.Equal(message, got)
	})
}
