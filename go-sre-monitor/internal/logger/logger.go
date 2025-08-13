package logger

import (
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

type Logger struct{ zerolog.Logger }

func New(level string) *Logger {
	lvl, err := zerolog.ParseLevel(strings.ToLower(level))
	if err != nil {
		lvl = zerolog.InfoLevel
	}
	z := zerolog.New(os.Stdout).With().Timestamp().Logger().Level(lvl)
	zerolog.TimeFieldFormat = time.RFC3339
	return &Logger{z}
}