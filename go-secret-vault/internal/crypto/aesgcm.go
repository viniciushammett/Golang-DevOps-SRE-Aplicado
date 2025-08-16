package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
)

type AESEncryptor struct { key []byte }

func NewAESEncryptor(masterKeyB64 string) (*AESEncryptor, error) {
	k, err := base64.StdEncoding.DecodeString(masterKeyB64)
	if err != nil { return nil, err }
	if len(k) < 32 { return nil, errors.New("master key must be 32 bytes (base64)") }
	k = k[:32]
	return &AESEncryptor{key: k}, nil
}

func (e *AESEncryptor) deriveKey(salt []byte) ([]byte, error) {
	h := hkdf.New(sha256.New, e.key, salt, []byte("gsv-aesgcm"))
	out := make([]byte, 32)
	if _, err := io.ReadFull(h, out); err != nil { return nil, err }
	return out, nil
}

// Encrypt returns base64(nonce|ciphertext)
func (e *AESEncryptor) Encrypt(plaintext []byte) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil { return "", err }
	k, err := e.deriveKey(salt)
	if err != nil { return "", err }
	block, err := aes.NewCipher(k)
	if err != nil { return "", err }
	gcm, err := cipher.NewGCM(block)
	if err != nil { return "", err }
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil { return "", err }
	ct := gcm.Seal(nil, nonce, plaintext, nil)
	buf := append(salt, append(nonce, ct...)...)
	return base64.StdEncoding.EncodeToString(buf), nil
}

func (e *AESEncryptor) Decrypt(b64 string) ([]byte, error) {
	buf, err := base64.StdEncoding.DecodeString(b64)
	if err != nil { return nil, err }
	if len(buf) < 16 { return nil, errors.New("cipher too short") }
	salt := buf[:16]
	rest := buf[16:]
	k, err := e.deriveKey(salt)
	if err != nil { return nil, err }
	block, err := aes.NewCipher(k)
	if err != nil { return nil, err }
	gcm, err := cipher.NewGCM(block)
	if err != nil { return nil, err }
	if len(rest) < gcm.NonceSize() { return nil, errors.New("nonce too short") }
	nonce := rest[:gcm.NonceSize()]
	ct := rest[gcm.NonceSize():]
	pt, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil { return nil, err }
	return pt, nil
}