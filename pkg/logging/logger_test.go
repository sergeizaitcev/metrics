package logging_test

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sergeizaitcev/metrics/pkg/logging"
)

func TestLogger(t *testing.T) {
	const pattern = `{
		"level":".+",
		"ts":"\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z",
		"msg":".+"
	}`

	r := strings.NewReplacer("\n", "", "\t", "")
	re := regexp.MustCompile(r.Replace(pattern))

	t.Run("log", func(t *testing.T) {
		levels := []logging.Level{
			logging.LevelDebug,
			logging.LevelInfo,
			logging.LevelError,
		}

		for _, level := range levels {
			t.Run(level.String(), func(t *testing.T) {
				var buf bytes.Buffer
				logger := logging.New(&buf, level)

				logger.Log(level, "test")
				if !assert.True(t, re.Match(buf.Bytes())) {
					t.Log(buf.String())
				}

				buf.Reset()

				logger.Log(level-1, "")
				if !assert.Empty(t, buf.Bytes()) {
					t.Log(buf.String())
				}
			})
		}
	})

	t.Run("custom", func(t *testing.T) {
		var buf bytes.Buffer

		logger := logging.New(&buf, logging.LevelInfo)
		logger.Log(logging.LevelInfo+2, "test")

		if !assert.True(t, re.Match(buf.Bytes())) {
			t.Log(buf.String())
		}
	})
}
