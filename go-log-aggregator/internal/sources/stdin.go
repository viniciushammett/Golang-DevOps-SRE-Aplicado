package sources

import (
	"bufio"
	"context"
	"os"
	"time"

	"github.com/viniciushammett/go-log-aggregator/internal/buffer"
)

type Stdin struct{ name string }
func NewStdin(name string) *Stdin { if name=="" { name="stdin" }; return &Stdin{name: name} }

func (s *Stdin) run(ctx context.Context, out chan<- buffer.Event) {
	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		out <- buffer.Event{When: time.Now(), Source: s.name, Line: sc.Text()}
	}
}