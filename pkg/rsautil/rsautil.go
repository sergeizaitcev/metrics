package rsautil

import (
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"strings"

	"github.com/sergeizaitcev/metrics/pkg/randutil"
)

// Generate генерирует и возвращает приватный RSA ключ.
func Generate(bits int) (*rsa.PrivateKey, error) {
	key, err := rsa.GenerateKey(randutil.Rand, bits)
	if err != nil {
		return nil, fmt.Errorf("generate rsa key: %w", err)
	}
	return key, nil
}

// Save сохраняет приватный RSA ключ в PEM формат.
func Save(key *rsa.PrivateKey, prefix string) error {
	pub := key.Public()

	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})

	pubPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: x509.MarshalPKCS1PublicKey(pub.(*rsa.PublicKey)),
	})

	prefix = strings.TrimSpace(prefix)

	err := os.WriteFile(prefix+".rsa", keyPEM, 0o600)
	if err != nil {
		return fmt.Errorf("write a private key to file: %w", err)
	}

	err = os.WriteFile(prefix+".rsa.pub", pubPEM, 0o644)
	if err != nil {
		return fmt.Errorf("write a public key to file: %w", err)
	}

	return nil
}

// Private возвращает приватный RSA ключ из filename.
func Private(filename string) (*rsa.PrivateKey, error) {
	block, err := open(filename)
	if err != nil {
		return nil, err
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

// Public возвращает публичный RSA ключ из filename.
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
