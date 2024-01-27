package metrics

import (
	"bytes"
	"encoding/base64"

	"github.com/sergeizaitcev/metrics/pkg/sign"
)

// Sign вычисляет хеш метрик и возвращает 256-битную подпись.
func Sign(key string, values []Metric) string {
	var buf bytes.Buffer

	for _, value := range values {
		b, _ := value.MarshalBinary()
		_, _ = buf.Write(b)
	}

	s := sign.Signer(key)
	signed := s.Sign(buf.Bytes())

	return base64.RawURLEncoding.EncodeToString(signed)
}
