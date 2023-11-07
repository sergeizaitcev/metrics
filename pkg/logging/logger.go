package logging

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

var defaultLogger = New(os.Stdout, LevelDebug)

func init() {
	zerolog.TimestampFunc = func() time.Time { return time.Now().UTC() }
	zerolog.TimeFieldFormat = time.RFC3339
	zerolog.TimestampFieldName = "ts"
	zerolog.MessageFieldName = "msg"
}

// Logger определяет регистратор логов.
type Logger struct {
	level Level
	log   zerolog.Logger
}

// New возвращает новый экземпляр Logger.
func New(w io.Writer, level Level) *Logger {
	log := zerolog.New(w)
	return &Logger{level: level, log: log}
}

// Discard возвращает пустой Logger.
func Discard() *Logger {
	return New(io.Discard, LevelError+1)
}

// allowed возвращает true, если уровень логирования разрешён.
func (l *Logger) allowed(level Level) bool {
	return level >= l.level
}

// Log записывает форматирование сообщение.
func (l *Logger) Log(level Level, msg string, a ...any) {
	if !l.allowed(level) {
		return
	}

	ev := l.log.Log()
	ev = ev.Str("level", level.String())
	ev = ev.Timestamp()

	if len(a) > 0 && len(a)%2 == 0 {
		for i := 0; i < len(a)-1; i = i + 2 {
			key, ok := a[i].(string)
			if !ok {
				key = fmt.Sprintf("%s", a[i])
			}
			value := a[i+1]
			ev = ev.Any(key, value)
		}
	}

	ev.Msg(msg)
}

// Debug записывает форматирование сообщение с уровнем debug.
func Debug(msg string, a ...any) {
	defaultLogger.Log(LevelDebug, msg, a...)
}

// Info записывает форматирование сообщение с уровнем info.
func Info(msg string, a ...any) {
	defaultLogger.Log(LevelInfo, msg, a...)
}

// Error записывает форматирование сообщение с уровнем error.
func Error(msg string, a ...any) {
	defaultLogger.Log(LevelError, msg, a...)
}

// Log записывает форматирование сообщение.
func Log(level Level, msg string, a ...any) {
	defaultLogger.Log(level, msg, a...)
}
