package store

import (
	"encoding/json"
	"time"

	bolt "go.etcd.io/bbolt"
)

var (
	bucketDedupe = []byte("dedupe") // key=fingerprint, val=unix_ts (last seen)
	bucketSilence = []byte("silence") // key=id, val=json
	bucketDLQ   = []byte("dlq")    // key=auto, val=json
	bucketRate  = []byte("rate")   // key=route:minute_unix, val=count
)

type Store struct{ db *bolt.DB }

func Open(path string) (*Store, error) {
	db, err := bolt.Open(path, 0o600, &bolt.Options{Timeout: 2 * time.Second})
	if err != nil { return nil, err }
	err = db.Update(func(tx *bolt.Tx) error {
		for _, b := range [][]byte{bucketDedupe, bucketSilence, bucketDLQ, bucketRate} {
			if _, e := tx.CreateBucketIfNotExists(b); e != nil { return e }
		}
		return nil
	})
	if err != nil { _ = db.Close(); return nil, err }
	return &Store{db: db}, nil
}
func (s *Store) Close() error { return s.db.Close() }

// Dedupe
func (s *Store) SeenRecently(fp string, window time.Duration) (bool, error) {
	var last time.Time
	err := s.db.View(func(tx *bolt.Tx) error {
		v := tx.Bucket(bucketDedupe).Get([]byte(fp))
		if v == nil { return nil }
		var ts int64
		if err := json.Unmarshal(v, &ts); err != nil { return err }
		last = time.Unix(ts,0)
		return nil
	})
	if err != nil { return false, err }
	if last.IsZero() { return false, nil }
	return time.Since(last) < window, nil
}
func (s *Store) MarkSeen(fp string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		ts, _ := json.Marshal(time.Now().Unix())
		return tx.Bucket(bucketDedupe).Put([]byte(fp), ts)
	})
}

// Rate limit (per-minute key)
func (s *Store) IncRate(route string, limitPerMin int) (allowed bool, err error) {
	nowMin := time.Now().Unix() / 60
	key := []byte(route + ":" + itoa(nowMin))
	err = s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketRate)
		v := b.Get(key)
		var c int
		if v != nil { _ = json.Unmarshal(v, &c) }
		if c >= limitPerMin { allowed = false; return nil }
		c++
		buf, _ := json.Marshal(c)
		return b.Put(key, buf)
	})
	if err != nil { return false, err }
	if limitPerMin <= 0 { return true, nil }
	if !allowed && limitPerMin > 0 { return false, nil }
	return true, nil
}

// DLQ
type DLQItem struct {
	When   time.Time `json:"when"`
	Route  string    `json:"route"`
	Dest   string    `json:"dest"`
	Alert  any       `json:"alert"`
	Error  string    `json:"error"`
}
func (s *Store) PutDLQ(item DLQItem) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		id := []byte(itob(time.Now().UnixNano()))
		b, _ := json.Marshal(item)
		return tx.Bucket(bucketDLQ).Put(id, b)
	})
}

// Silences (simplificado)
type Silence struct {
	ID     string    `json:"id"`
	Label  string    `json:"label"`
	Regex  string    `json:"regex"`
	Until  time.Time `json:"until"`
}
func (s *Store) PutSilence(sil Silence) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b, _ := json.Marshal(sil)
		return tx.Bucket(bucketSilence).Put([]byte(sil.ID), b)
	})
}
func (s *Store) ListSilences() ([]Silence, error) {
	var out []Silence
	err := s.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket(bucketSilence).Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var si Silence
			if err := json.Unmarshal(v, &si); err == nil { out = append(out, si) }
		}
		return nil
	})
	return out, err
}
func (s *Store) PurgeExpiredSilences() error {
	now := time.Now()
	return s.db.Update(func(tx *bolt.Tx) error {
		c := tx.Bucket(bucketSilence).Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var si Silence
			if err := json.Unmarshal(v, &si); err == nil {
				if now.After(si.Until) { _ = c.Delete() }
			}
		}
		return nil
	})
}

// utils
func itoa(v int64) string { return string([]byte(fmtInt(v))) }
func itob(v int64) []byte { return []byte(fmtInt(v)) }
func fmtInt(v int64) string { return fmt.Sprintf("%d", v) }

import "fmt"