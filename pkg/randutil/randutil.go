package randutil

import (
	"crypto/rand"
	"encoding/binary"
	"io"
	mathrand "math/rand"
)

var Rand *mathrand.Rand

func init() {
	buf := make([]byte, 8)
	_, err := io.ReadFull(rand.Reader, buf)
	if err != nil {
		panic(err)
	}
	src := mathrand.NewSource(int64(binary.LittleEndian.Uint64(buf)))
	Rand = mathrand.New(src)
}
