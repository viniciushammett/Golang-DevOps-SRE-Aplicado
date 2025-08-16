package vault

import (
	"encoding/json"
	"errors"
	"time"

	bolt "go.etcd.io/bbolt"
	"github.com/google/uuid"
)

type Secret struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	CipherB64 string            `json:"cipher_b64"`
	Meta      map[string]string `json:"meta,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	ExpiresAt *time.Time        `json:"expires_at,omitempty"`
}

type BoltStore struct { db *bolt.DB }

var bucket = []byte("secrets")

func NewBolt(path string) (*BoltStore, error) {
	db, err := bolt.Open(path, 0o600, &bolt.Options{Timeout: time.Second})
	if err != nil { return nil, err }
	if err := db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(bucket)
		return err
	}); err != nil { return nil, err }
	return &BoltStore{db: db}, nil
}

func (s *BoltStore) Close() error { return s.db.Close() }

func (s *BoltStore) Create(name, cipher string, ttl time.Duration, meta map[string]string) (Secret, error) {
	id := uuid.NewString()
	now := time.Now().UTC()
	var exp *time.Time
	if ttl > 0 { t := now.Add(ttl); exp = &t }
	sec := Secret{ID: id, Name: name, CipherB64: cipher, Meta: meta, CreatedAt: now, UpdatedAt: now, ExpiresAt: exp}
	err := s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucket)
		_, err := b.CreateBucketIfNotExists([]byte("byname"))
		if err != nil { return err }
		if b.Bucket([]byte("byname")).Get([]byte(name)) != nil { return errors.New("name exists") }
		js, _ := json.Marshal(sec)
		if err := b.Put([]byte(id), js); err != nil { return err }
		return b.Bucket([]byte("byname")).Put([]byte(name), []byte(id))
	})
	return sec, err
}

func (s *BoltStore) Get(id string) (Secret, error) {
	var sec Secret
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucket)
		v := b.Get([]byte(id))
		if v == nil { return errors.New("not found") }
		return json.Unmarshal(v, &sec)
	})
	return sec, err
}

func (s *BoltStore) GetByName(name string) (Secret, error) {
	var id string
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucket)
		v := b.Bucket([]byte("byname")).Get([]byte(name))
		if v == nil { return errors.New("not found") }
		id = string(v)
		return nil
	})
	if err != nil { return Secret{}, err }
	return s.Get(id)
}

func (s *BoltStore) Update(id string, cipher string, ttl *time.Duration, meta map[string]string) (Secret, error) {
	var out Secret
	err := s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucket)
		v := b.Get([]byte(id))
		if v == nil { return errors.New("not found") }
		var sec Secret
		if err := json.Unmarshal(v, &sec); err != nil { return err }
		if cipher != "" { sec.CipherB64 = cipher }
		if ttl != nil {
			if *ttl > 0 { t := time.Now().UTC().Add(*ttl); sec.ExpiresAt = &t } else { sec.ExpiresAt = nil }
		}
		if meta != nil { sec.Meta = meta }
		sec.UpdatedAt = time.Now().UTC()
		js, _ := json.Marshal(sec)
		if err := b.Put([]byte(id), js); err != nil { return err }
		out = sec
		return nil
	})
	return out, err
}

func (s *BoltStore) Delete(id string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucket)
		v := b.Get([]byte(id))
		if v == nil { return errors.New("not found") }
		var sec Secret
		_ = json.Unmarshal(v, &sec)
		if err := b.Delete([]byte(id)); err != nil { return err }
		_ = b.Bucket([]byte("byname")).Delete([]byte(sec.Name))
		return nil
	})
}

func (s *BoltStore) List(includeExpired bool) ([]Secret, error) {
	var out []Secret
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucket)
		return b.ForEach(func(k, v []byte) error {
			if string(k) == "byname" { return nil }
			var sec Secret
			if err := json.Unmarshal(v, &sec); err != nil { return err }
			if !includeExpired && sec.ExpiresAt != nil && time.Now().UTC().After(*sec.ExpiresAt) {
				return nil
			}
			out = append(out, sec)
			return nil
		})
	})
	return out, err
}

func (s *BoltStore) ReapExpired() (int, error) {
	count := 0
	err := s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucket)
		var ids []string
		err := b.ForEach(func(k, v []byte) error {
			if string(k) == "byname" { return nil }
			var sec Secret
			if err := json.Unmarshal(v, &sec); err != nil { return err }
			if sec.ExpiresAt != nil && time.Now().UTC().After(*sec.ExpiresAt) {
				ids = append(ids, string(k))
			}
			return nil
		})
		if err != nil { return err }
		for _, id := range ids {
			v := b.Get([]byte(id))
			if v != nil { var s Secret; _ = json.Unmarshal(v, &s); _ = b.Bucket([]byte("byname")).Delete([]byte(s.Name)) }
			if err := b.Delete([]byte(id)); err != nil { return err }
			count++
		}
		return nil
	})
	return count, err
}