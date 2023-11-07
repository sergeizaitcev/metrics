package sign

import (
	"crypto/hmac"
	"crypto/sha256"
)

// Signer определяет ключ для подписи данных по алгоритму sha256.
type Signer []byte

// Sign вычисляет хеш данных и возвращает 32-битную подпись.
func (s Signer) Sign(b []byte) []byte {
	hash := hmac.New(sha256.New, s)
	hash.Write(b)
	return hash.Sum(nil)
}
