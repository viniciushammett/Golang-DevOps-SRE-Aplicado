package audit

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

type Logger struct {
	mu   sync.Mutex
	file *os.File
}

type Entry struct {
	Time    time.Time         `json:"time"`
	Actor   string            `json:"actor"`
	Action  string            `json:"action"`
	Target  string            `json:"target"`
	Outcome string            `json:"outcome"`
	Meta    map[string]string `json:"meta,omitempty"`
}

func New(path string) (*Logger, error) {
	if err := os.MkdirAll("./data", 0o755); err != nil { /* best-effort */ }
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil { return nil, err }
	return &Logger{file: f}, nil
}

func (l *Logger) Close() error { return l.file.Close() }

func (l *Logger) Log(e Entry) {
	l.mu.Lock()
	defer l.mu.Unlock()
	e.Time = time.Now().UTC()
	_ = json.NewEncoder(l.file).Encode(e)
}