package vault

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"time"

	"go-secret-vault/internal/crypto"
)

type Store interface {
	Create(name, cipher string, ttl time.Duration, meta map[string]string) (Secret, error)
	Get(id string) (Secret, error)
	GetByName(name string) (Secret, error)
	Update(id string, cipher string, ttl *time.Duration, meta map[string]string) (Secret, error)
	Delete(id string) error
	List(includeExpired bool) ([]Secret, error)
	ReapExpired() (int, error)
}

type Service struct {
	store Store
	enc   *crypto.AESEncryptor
}

func NewService(store Store, enc *crypto.AESEncryptor) *Service { return &Service{store: store, enc: enc} }

func (s *Service) Create(name string, plaintext []byte, ttl time.Duration, meta map[string]string) (Secret, error) {
	cipher, err := s.enc.Encrypt(plaintext)
	if err != nil { return Secret{}, err }
	return s.store.Create(name, cipher, ttl, meta)
}

func (s *Service) GetDecrypted(id string) (Secret, []byte, error) {
	sec, err := s.store.Get(id)
	if err != nil { return Secret{}, nil, err }
	pt, err := s.enc.Decrypt(sec.CipherB64)
	return sec, pt, err
}

func (s *Service) Update(id string, newPlain []byte, ttl *time.Duration, meta map[string]string) (Secret, error) {
	cipher := ""
	if newPlain != nil { c, err := s.enc.Encrypt(newPlain); if err != nil { return Secret{}, err }; cipher = c }
	return s.store.Update(id, cipher, ttl, meta)
}

func (s *Service) ExportK8sYAML(id, namespace, key string) ([]byte, error) {
	sec, pt, err := s.GetDecrypted(id)
	if err != nil { return nil, err }
	if key == "" { key = "VALUE" }
	b64 := base64.StdEncoding.EncodeToString(pt)
	yaml := fmt.Sprintf(`apiVersion: v1
kind: Secret
metadata:
  name: %s
  namespace: %s
type: Opaque
data:
  %s: %s
`, sanitizeName(sec.Name), namespace, key, b64)
	return []byte(yaml), nil
}

func sanitizeName(n string) string {
	var b bytes.Buffer
	for i := 0; i < len(n); i++ {
		c := n[i]
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' { b.WriteByte(c) } else if c >= 'A' && c <= 'Z' { b.WriteByte(c + 32) } else { b.WriteByte('-') }
	}
	if b.Len() == 0 { return "secret" }
	return b.String()
}