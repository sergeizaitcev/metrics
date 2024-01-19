package rsautil

import (
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"os"

	"github.com/sergeizaitcev/metrics/pkg/randutil"
)

// Private возвращает приватный rsa ключ из filename.
func Private(filename string) (*rsa.PrivateKey, error) {
	block, err := open(filename)
	if err != nil {
		return nil, err
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

// Public возвращает публичный rsa ключ из filename.
func Public(filename string) (*rsa.PublicKey, error) {
	block, err := open(filename)
	if err != nil {
		return nil, err
	}
	return x509.ParsePKCS1PublicKey(block.Bytes)
}

// Encrypt шифрует сообщение при помощи публичного ключа.
func Encrypt(key *rsa.PublicKey, message []byte) ([]byte, error) {
	hash := sha256.New()
	msgLen := len(message)
	step := key.Size() - 2*hash.Size() - 2

	var encryptedBytes []byte
	label := []byte("")

	for start := 0; start < msgLen; start += step {
		finish := start + step
		if finish > msgLen {
			finish = msgLen
		}

		encryptedBlockBytes, err := rsa.EncryptOAEP(
			hash,
			randutil.Rand,
			key,
			message[start:finish],
			label,
		)
		if err != nil {
			return nil, err
		}

		encryptedBytes = append(encryptedBytes, encryptedBlockBytes...)
	}

	return encryptedBytes, nil
}

// Decrypt расшифровывает сообщение при помощи приватного ключа.
func Decrypt(key *rsa.PrivateKey, message []byte) ([]byte, error) {
	msgLen := len(message)
	step := key.PublicKey.Size()

	var decryptedBytes []byte
	hash := sha256.New()
	label := []byte("")

	for start := 0; start < msgLen; start += step {
		finish := start + step
		if finish > msgLen {
			finish = msgLen
		}

		decryptedBlockBytes, err := rsa.DecryptOAEP(
			hash,
			randutil.Rand,
			key,
			message[start:finish],
			label,
		)
		if err != nil {
			return nil, err
		}

		decryptedBytes = append(decryptedBytes, decryptedBlockBytes...)
	}

	return decryptedBytes, nil
}

func open(filename string) (*pem.Block, error) {
	b, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(b)
	return block, nil
}
