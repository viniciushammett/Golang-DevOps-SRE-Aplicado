package store

import (
	"encoding/json"
	"time"

	bolt "go.etcd.io/bbolt"
)

var (
	bAnoms = []byte("anomalies")
	bLogs  = []byte("logs") // NOVO: logs brutos (key=tsNano:rand, val=json)
)

type Store struct{ db *bolt.DB }

func Open(path string) (*Store, error) {
	db, err := bolt.Open(path, 0o600, &bolt.Options{Timeout: 2 * time.Second})
	if err != nil { return nil, err }
	err = db.Update(func(tx *bolt.Tx) error {
		if _, e := tx.CreateBucketIfNotExists(bAnoms); e != nil { return e }
		if _, e := tx.CreateBucketIfNotExists(bLogs); e != nil { return e }
		return nil
	})
	if err != nil { db.Close(); return nil, err }
	return &Store{db: db}, nil
}
func (s *Store) Close() error { return s.db.Close() }

// -------- Anomalias (já existentes) --------

type Anomaly struct {
	ID       string            `json:"id"`
	When     time.Time         `json:"when"`
	Rule     string            `json:"rule"`
	Kind     string            `json:"kind"`
	Sample   string            `json:"sample"`
	Count    int               `json:"count"`
	Window   string            `json:"window"`
	Severity string            `json:"severity"`
	Meta     map[string]string `json:"meta,omitempty"`
}

func (s *Store) PutAnomaly(a Anomaly) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		j, _ := json.Marshal(a)
		return tx.Bucket(bAnoms).Put([]byte(a.ID), j)
	})
}

func (s *Store) List(limit int) ([]Anomaly, error) {
	out := []Anomaly{}
	_ = s.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket(bAnoms).Cursor()
		for k, v := c.Last(); k != nil; k, v = c.Prev() {
			var a Anomaly
			if json.Unmarshal(v, &a) == nil {
				out = append(out, a)
				if limit > 0 && len(out) >= limit { break }
			}
		}
		return nil
	})
	return out, nil
}

// -------- NOVO: Logs brutos --------

type LogRecord struct {
	TS     time.Time         `json:"ts"`
	Source string            `json:"source"`
	Msg    string            `json:"msg"`
	Meta   map[string]string `json:"meta,omitempty"`
}

func (s *Store) PutLog(lr LogRecord) error {
	key := []byte(lr.TS.UTC().Format(time.RFC3339Nano)) // ordenável por tempo
	return s.db.Update(func(tx *bolt.Tx) error {
		j, _ := json.Marshal(lr)
		// Para evitar colisão no mesmo nanossegundo, acrescente sufixo incremental se necessário
		b := tx.Bucket(bLogs)
		k := key
		for i := 0; ; i++ {
			if i > 0 {
				k = []byte(lr.TS.UTC().Format(time.RFC3339Nano) + sprintf(":%03d", i))
			}
			if b.Get(k) == nil { return b.Put(k, j) }
		}
	})
}

func (s *Store) IterateLogs(fn func(lr LogRecord) bool) error {
	return s.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket(bLogs).Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var lr LogRecord
			if json.Unmarshal(v, &lr) != nil { continue }
			if !fn(lr) { break }
		}
		return nil
	})
}

// helper para fmt sem importar globalmente
func sprintf(format string, a ...any) string {
	return fmtSprintf(format, a...)
}
func fmtSprintf(format string, a ...any) string {
	return fmt.Sprintf(format, a...)
}