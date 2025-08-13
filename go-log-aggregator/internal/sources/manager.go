package sources

import (
	"context"

	"github.com/viniciushammett/go-log-aggregator/internal/buffer"
	"github.com/viniciushammett/go-log-aggregator/internal/logger"
)

type runner interface {
	run(context.Context, chan<- buffer.Event)
}

type Manager struct {
	log   *logger.Logger
	items []runner
}

func NewManager(log *logger.Logger) *Manager { return &Manager{log: log} }
func (m *Manager) Add(r runner) { m.items = append(m.items, r) }

func (m *Manager) Run(ctx context.Context) <-chan buffer.Event {
	out := make(chan buffer.Event, 1024)
	for _, r := range m.items {
		go r.run(ctx, out)
	}
	go func() {
		<-ctx.Done()
		close(out)
	}()
	return out
}