package scheduler

import (
	"context"
	"time"

	"github.com/viniciushammett/go-alert-router/internal/logger"
	"github.com/viniciushammett/go-alert-router/internal/store"
)

func Start(ctx context.Context, log *logger.Logger, st *store.Store, interval time.Duration) {
	t := time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				_ = st.PurgeExpiredSilences()
				// Aqui poderia rolar limpeza de dedupe muito antigo, DLQ trim, etc.
			}
		}
	}()
}