package sources

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os"
	"time"

	"github.com/viniciushammett/go-log-aggregator/internal/buffer"
	"github.com/viniciushammett/go-log-aggregator/internal/metrics"
)

type FileTail struct {
	path        string
	name        string
	poll        time.Duration
}

func NewFileTail(path, name string, poll time.Duration) *FileTail {
	if name == "" { name = path }
	if poll <= 0 { poll = 500 * time.Millisecond }
	return &FileTail{path: path, name: name, poll: poll}
}

func (t *FileTail) run(ctx context.Context, out chan<- buffer.Event) {
	var f *os.File
	var r *bufio.Reader
	var err error
	var offset int64

	open := func() error {
		_ = closeFile(f)
		f, err = os.Open(t.path)
		if err != nil { return err }
		// seek end for tail semantics
		offset, err = f.Seek(0, io.SeekEnd)
		if err != nil { return err }
		r = bufio.NewReader(f)
		return nil
	}
	_ = open()

	ticker := time.NewTicker(t.poll)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			_ = closeFile(f)
			return
		case <-ticker.C:
			if f == nil {
				// try reopen
				if err := open(); err != nil {
					metrics.IngestErrors.WithLabelValues(t.name).Inc()
					continue
				}
			}
			for {
				line, e := r.ReadString('\n')
				if errors.Is(e, io.EOF) {
					// file rotated? verify size smaller than offset -> reopen
					if st, statErr := f.Stat(); statErr == nil && st.Size() < offset {
						_ = open()
					}
					break
				}
				offset += int64(len(line))
				if e != nil && !errors.Is(e, io.EOF) {
					metrics.IngestErrors.WithLabelValues(t.name).Inc()
					break
				}
				out <- buffer.Event{When: time.Now(), Source: t.name, Line: trimNL(line)}
			}
		}
	}
}

func closeFile(f *os.File) error {
	if f != nil { return f.Close() }
	return nil
}

func trimNL(s string) string {
	if len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		return s[:len(s)-1]
	}
	return s
}