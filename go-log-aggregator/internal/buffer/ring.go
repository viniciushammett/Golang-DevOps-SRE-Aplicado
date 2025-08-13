package buffer

import (
	"regexp"
	"sync"
	"time"
)

type Event struct {
	When   time.Time `json:"when"`
	Source string    `json:"source"`
	Line   string    `json:"line"`
}

type Ring struct {
	mu   sync.RWMutex
	data []Event
	size int
	pos  int
	full bool
}

func NewRing(size int) *Ring {
	if size < 128 { size = 128 }
	return &Ring{data: make([]Event, size), size: size}
}

func (r *Ring) Push(ev Event) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[r.pos] = ev
	r.pos = (r.pos + 1) % r.size
	if r.pos == 0 { r.full = true }
}

type Query struct {
	Since       time.Time
	Until       time.Time
	Include     *regexp.Regexp
	Exclude     *regexp.Regexp
	SourceEqual string
	Limit       int
	Offset      int
}

func (r *Ring) Snapshot(q Query) []Event {
	r.mu.RLock(); defer r.mu.RUnlock()
	var out []Event
	N := r.size
	if !r.full { N = r.pos }
	// iterate from newest to oldest to honor offset/limit naturally
	for i := 0; i < N; i++ {
		idx := (r.pos - 1 - i + r.size) % r.size
		ev := r.data[idx]
		if ev.When.IsZero() { continue }
		if !q.Since.IsZero() && ev.When.Before(q.Since) { continue }
		if !q.Until.IsZero() && ev.When.After(q.Until) { continue }
		if q.SourceEqual != "" && ev.Source != q.SourceEqual { continue }
		if q.Include != nil && !q.Include.MatchString(ev.Line) { continue }
		if q.Exclude != nil && q.Exclude.MatchString(ev.Line) { continue }
		out = append(out, ev)
		if q.Limit > 0 && len(out) >= q.Limit+q.Offset { break }
	}
	// apply offset/limit (already newest-first)
	if q.Offset > 0 && q.Offset < len(out) {
		out = out[q.Offset:]
	}
	if q.Limit > 0 && q.Limit < len(out) {
		out = out[:q.Limit]
	}
	// reverse to oldest->newest for API friendliness
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}