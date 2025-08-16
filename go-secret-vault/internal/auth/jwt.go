package auth

import (
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWT struct { key []byte }

type Claims struct {
	Username string   `json:"sub"`
	Roles    []string `json:"roles"`
	jwt.RegisteredClaims
}

func NewJWT(secretB64 string) (*JWT, error) {
	k, err := base64.StdEncoding.DecodeString(secretB64)
	if err != nil { return nil, err }
	if len(k) < 32 { return nil, errors.New("jwt secret must be >=32 bytes") }
	return &JWT{key: k}, nil
}

func (j *JWT) Issue(u User, ttl time.Duration) (string, error) {
	claims := &Claims{
		Username: u.Username,
		Roles:    u.Roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(j.key)
}

func (j *JWT) Parse(token string) (*Claims, error) {
	tok, err := jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (interface{}, error) { return j.key, nil })
	if err != nil { return nil, err }
	if c, ok := tok.Claims.(*Claims); ok && tok.Valid { return c, nil }
	return nil, errors.New("invalid token")
}

func (j *JWT) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/healthz") || strings.HasPrefix(r.URL.Path, "/login") {
			next.ServeHTTP(w, r); return
		}
		h := r.Header.Get("Authorization")
		parts := strings.SplitN(h, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			http.Error(w, "missing bearer token", http.StatusUnauthorized); return
		}
		claims, err := j.Parse(parts[1])
		if err != nil { http.Error(w, "invalid token", http.StatusUnauthorized); return }
		// attach user to context (simplified)
		r = r.WithContext(WithUser(r.Context(), claims.Username, claims.Roles))
		next.ServeHTTP(w, r)
	})
}