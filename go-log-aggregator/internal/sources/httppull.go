package sources

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/viniciushammett/go-log-aggregator/internal/buffer"
	"github.com/viniciushammett/go-log-aggregator/internal/metrics"
)

type HTTPPull struct {
	url   string
	name  string
	intvl time.Duration
}

func NewHTTPPull(url, name string, interval time.Duration) *HTTPPull {
	if name == "" { name = url }
	if interval <= 0 { interval = 3 * time.Second }
	return &HTTPPull{url: url, name: name, intvl: interval}
}

func (p *HTTPPull) run(ctx context.Context, out chan<- buffer.Event) {
	client := &http.Client{ Timeout: 10 * time.Second }
	t := time.NewTicker(p.intvl)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			resp, err := client.Get(p.url)
			if err != nil {
				metrics.IngestErrors.WithLabelValues(p.name).Inc()
				continue
			}
			body, err := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if err != nil { metrics.IngestErrors.WithLabelValues(p.name).Inc(); continue }
			out <- buffer.Event{When: time.Now(), Source: p.name, Line: string(body)}
		}
	}
}