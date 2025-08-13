package logger

import (
	"net/http"
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

func (l *Logger) HTTPLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		l.Info().Str("method", r.Method).Str("path", r.URL.Path).Dur("dur", time.Since(start)).Msg("http")
	})
}