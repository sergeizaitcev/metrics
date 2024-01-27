package rsautil_test

import (
	"crypto/rsa"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/sergeizaitcev/metrics/pkg/rsautil"
)

type RsaSuite struct {
	suite.Suite

	private *rsa.PrivateKey
	public  *rsa.PublicKey

	prefix         string
	privateKeyPath string
	publicKeyPath  string
}

func TestRSA(t *testing.T) {
	suite.Run(t, new(RsaSuite))
}

func (suite *RsaSuite) SetupSuite() {
	dir := suite.T().TempDir()
	suite.prefix = filepath.Join(dir, "test")
	suite.privateKeyPath = suite.prefix + ".rsa"
	suite.publicKeyPath = suite.prefix + ".rsa.pub"
}

func (suite *RsaSuite) TestA_SaveToFile() {
	var key *rsa.PrivateKey
	var err error

	suite.Run("generate", func() {
		key, err = rsautil.Generate(2048)
		suite.NoError(err)
		suite.NotNil(key)
	})

	suite.Run("save", func() {
		err = rsautil.Save(key, suite.prefix)
		suite.NoError(err)
	})
}

func (suite *RsaSuite) TestB_ReadFromFile() {
	var err error

	suite.Run("private", func() {
		suite.private, err = rsautil.Private(suite.privateKeyPath)
		suite.NoError(err)
		suite.NotNil(suite.private)
	})

	suite.Run("public", func() {
		suite.public, err = rsautil.Public(suite.publicKeyPath)
		suite.NoError(err)
		suite.NotNil(suite.public)
	})
}

func (suite *RsaSuite) TestC_EncryptingMessage() {
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
