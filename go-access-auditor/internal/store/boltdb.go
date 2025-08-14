package store

import (
	"encoding/json"
	"time"

	bolt "go.etcd.io/bbolt"
)

var (
	bEvents = []byte("events") // key=id, val=json
)

type Store struct{ db *bolt.DB }

func Open(path string) (*Store, error) {
	db, err := bolt.Open(path, 0o600, &bolt.Options{Timeout: 2 * time.Second})
	if err != nil { return nil, err }
	err = db.Update(func(tx *bolt.Tx) error {
		_, e := tx.CreateBucketIfNotExists(bEvents); return e
	})
	if err != nil { _ = db.Close(); return nil, err }
	return &Store{db: db}, nil
}
func (s *Store) Close() error { return s.db.Close() }

type Query struct {
	User, Host, Source string
	Since, Until       time.Time
	Text               string // contains
	Limit, Offset      int
	SensitiveOnly      bool
}

type Event struct {
	ID       string            `json:"id"`
	When     time.Time         `json:"when"`
	User     string            `json:"user"`
	Host     string            `json:"host"`
	Source   string            `json:"source"`
	Command  string            `json:"command"`
	Meta     map[string]string `json:"meta"`
	FlagSensitive bool         `json:"flagSensitive"`
	RuleMatched   string       `json:"ruleMatched,omitempty"`
}

func (s *Store) Put(ev Event) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b, _ := json.Marshal(ev)
		return tx.Bucket(bEvents).Put([]byte(ev.ID), b)
	})
}

func (s *Store) List(q Query) ([]Event, error) {
	var out []Event
	err := s.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket(bEvents).Cursor()
		for k, v := c.Last(); k != nil; k, v = c.Prev() {
			var e Event
			if json.Unmarshal(v, &e) != nil { continue }
			if !q.Since.IsZero() && e.When.Before(q.Since) { continue }
			if !q.Until.IsZero() && e.When.After(q.Until) { continue }
			if q.User != "" && e.User != q.User { continue }
			if q.Host != "" && e.Host != q.Host { continue }
			if q.Source != "" && e.Source != q.Source { continue }
			if q.SensitiveOnly && !e.FlagSensitive { continue }
			if q.Text != "" && !containsFold(e.Command, q.Text) { continue }
			out = append(out, e)
			if q.Limit > 0 && len(out) >= q.Limit+q.Offset { break }
		}
		return nil
	})
	if err != nil { return nil, err }
	if q.Offset > 0 && q.Offset < len(out) { out = out[q.Offset:] }
	if q.Limit > 0 && q.Limit < len(out) { out = out[:q.Limit] }
	// newest-first já garantido
	return out, nil
}

func containsFold(s, sub string) bool {
	sLower, subLower := []rune(s), []rune(sub)
	_ = sLower
	_ = subLower
	// cheap case-insensitive check
	return len(sub) == 0 || (len(sub) > 0 && (indexFold(s, sub) >= 0))
}

func indexFold(s, substr string) int {
	return indexFoldASCII(s, substr)
}
// ascii-only for simplicity
func indexFoldASCII(s, sub string) int {
	toLow := func(b byte) byte {
		if 'A'<=b && b<='Z' { return b+('a'-'A') }; return b
	}
	for i:=0; i+len(sub)<=len(s); i++ {
		ok := true
		for j:=0; j<len(sub); j++ {
			if toLow(s[i+j]) != toLow(sub[j]) { ok=false; break }
		}
		if ok { return i }
	}
	return -1
}