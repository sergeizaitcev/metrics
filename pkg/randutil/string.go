package randutil

import (
	"unsafe"
)

const ascii = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFJHIJKLMNOPQRSTUVWXYZ"

// Bytes генерирует случайную байтовую последовательность длинной n, состоящую
// из ASCII-символов.
//
// Если n <= 0, то возвращает nil.
func Bytes(n int) []byte {
	if n <= 0 {
		return nil
	}

	buf := make([]byte, 0, n)
	i := 0

	for len(buf) < n {
		idx := rnd.Intn(len(ascii) - 1)
		char := ascii[idx]
		if i == 0 && '0' <= char && char <= '9' {
			continue
		}
		buf = append(buf, char)
		i++
	}

	return buf
}

// String генерирует случайную ASCII-последовательность длинной n.
//
// Если n <= 0, то возвращает пустую строку.
func String(n int) string {
	b := Bytes(n)	
	return unsafe.String(unsafe.SliceData(b), len(b))
}
