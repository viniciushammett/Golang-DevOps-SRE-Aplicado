package store

import (
	"encoding/json"
	"time"

	bolt "go.etcd.io/bbolt"
)

var (
	bAnoms = []byte("anomalies")
)

type Store struct{ db *bolt.DB }

func Open(path string) (*Store, error) {
	db, err := bolt.Open(path, 0o600, &bolt.Options{Timeout: 2 * time.Second})
	if err != nil { return nil, err }
	err = db.Update(func(tx *bolt.Tx) error { _, e := tx.CreateBucketIfNotExists(bAnoms); return e })
	if err != nil { db.Close(); return nil, err }
	return &Store{db: db}, nil
}
func (s *Store) Close() error { return s.db.Close() }

type Anomaly struct {
	ID       string            `json:"id"`
	When     time.Time         `json:"when"`
	Rule     string            `json:"rule"`
	Kind     string            `json:"kind"` // threshold|spike
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