package store

import (
	"encoding/json"
	"time"

	bolt "go.etcd.io/bbolt"
)

var (
	bDeploys = []byte("deploys") // id -> DeployRecord
)

type Store struct{ db *bolt.DB }

func Open(path string) (*Store, error) {
	db, err := bolt.Open(path, 0o600, &bolt.Options{Timeout: 2 * time.Second})
	if err != nil { return nil, err }
	err = db.Update(func(tx *bolt.Tx) error {
		_, e := tx.CreateBucketIfNotExists(bDeploys); return e
	})
	if err != nil { _ = db.Close(); return nil, err }
	return &Store{db: db}, nil
}
func (s *Store) Close() error { return s.db.Close() }

type DeployRecord struct {
	ID        string            `json:"id"`
	App       string            `json:"app"`
	Namespace string            `json:"namespace"`
	ImageNew  string            `json:"imageNew"`
	ImageOld  string            `json:"imageOld"`
	Strategy  string            `json:"strategy"`
	Status    string            `json:"status"` // started|waiting_approval|running|succeeded|failed|rolled_back
	Reason    string            `json:"reason,omitempty"`
	StartedAt time.Time         `json:"startedAt"`
	FinishedAt *time.Time       `json:"finishedAt,omitempty"`
	Params    map[string]string `json:"params"`
}

func (s *Store) Put(rec DeployRecord) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b, _ := json.Marshal(rec)
		return tx.Bucket(bDeploys).Put([]byte(rec.ID), b)
	})
}

func (s *Store) Get(id string) (*DeployRecord, error) {
	var out DeployRecord
	err := s.db.View(func(tx *bolt.Tx) error {
		v := tx.Bucket(bDeploys).Get([]byte(id))
		if v == nil { return nil }
		return json.Unmarshal(v, &out)
	})
	if err != nil { return nil, err }
	if out.ID == "" { return nil, nil }
	return &out, nil
}

func (s *Store) List() ([]DeployRecord, error) {
	var arr []DeployRecord
	err := s.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket(bDeploys).Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var r DeployRecord
			if json.Unmarshal(v, &r) == nil { arr = append(arr, r) }
		}
		return nil
	})
	return arr, err
}