package rsautil

import (
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"

	"github.com/sergeizaitcev/metrics/pkg/randutil"
)

// PrivateKey возвращает приватный RSA ключ из data.
func PrivateKey(data []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(data)
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parsing private key: %w", err)
	}
	rsakey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("unsupported private key type: %T", key)
	}
	return rsakey, nil
}

// PrivateKeyFrom возвращает приватный RSA ключ из файла name.
func PrivateKeyFrom(name string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(name)
	if err != nil {
		return nil, err
	}
	return PrivateKey(data)
}

// PublicKey возвращает публичный RSA ключ из data.
func PublicKey(data []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(data)
	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parsing public key: %w", err)
	}
	rsakey, ok := key.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("unsuppored public key type: %T", key)
	}
	return rsakey, nil
}

// PublicKeyFrom возвращает публичный RSA ключ из файла name.
func PublicKeyFrom(name string) (*rsa.PublicKey, error) {
	data, err := os.ReadFile(name)
	if err != nil {
		return nil, err
	}
	return PublicKey(data)
}

// Encrypt шифрует сообщение при помощи публичного RSA ключа.
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

// Decrypt расшифровывает сообщение при помощи приватного RSA ключа.
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
