package auth

import (
	"errors"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	Username string
	Hash     string
	Roles    []string
}

type UserStore struct { users map[string]User }

func NewUserStore(cfg []struct{ Username, PasswordBcrypt string; Roles []string }) *UserStore {
	m := make(map[string]User)
	for _, u := range cfg {
		m[u.Username] = User{Username: u.Username, Hash: u.PasswordBcrypt, Roles: u.Roles}
	}
	return &UserStore{users: m}
}

func (s *UserStore) Verify(username, password string) (User, error) {
	u, ok := s.users[username]
	if !ok { return User{}, errors.New("user not found") }
	if bcrypt.CompareHashAndPassword([]byte(u.Hash), []byte(password)) != nil { return User{}, errors.New("invalid password") }
	return u, nil
}