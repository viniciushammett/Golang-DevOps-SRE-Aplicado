package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"

	"go-secret-vault/internal/audit"
	"go-secret-vault/internal/auth"
	"go-secret-vault/internal/config"
	"go-secret-vault/internal/crypto"
	"go-secret-vault/internal/vault"
)

type api struct {
	vault *vault.Service
	store *vault.BoltStore
	jwt   *auth.JWT
	users *auth.UserStore
	log   *audit.Logger
}

func main() {
	cfgPath := os.Getenv("GSV_CONFIG")
	if cfgPath == "" { cfgPath = "./configs/config.yaml" }
	cfg, err := config.Load(cfgPath)
	if err != nil { log.Fatal(err) }

	enc, err := crypto.NewAESEncryptor(cfg.Security.MasterKeyB64)
	if err != nil { log.Fatal(err) }
	st, err := vault.NewBolt(cfg.Storage.BoltPath)
	if err != nil { log.Fatal(err) }
	defer st.Close()
	v := vault.NewService(st, enc)
	jwt, err := auth.NewJWT(cfg.Security.JWTSecretB64)
	if err != nil { log.Fatal(err) }
	users := auth.NewUserStore(convertUsers(cfg))
	alog, err := audit.New(cfg.Audit.File)
	if err != nil { log.Fatal(err) }
	defer alog.Close()

	api := &api{vault: v, store: st, jwt: jwt, users: users, log: alog}

	go func() { // TTL reaper
		for {
			if n, err := st.ReapExpired(); err == nil && n > 0 {
				alog.Log(audit.Entry{Actor: "system", Action: "reap", Outcome: "ok", Meta: map[string]string{"count": string(rune(n))}})
			}
			time.Sleep(time.Minute)
		}
	}()

	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{AllowedOrigins: cfg.Server.CORSOrigins, AllowedMethods: []string{"GET","POST","PUT","DELETE","OPTIONS"}, AllowedHeaders: []string{"*"}, AllowCredentials: true}))
	r.Use(jwt.Middleware)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })
	r.Post("/login", api.login)

	r.Route("/secrets", func(sr chi.Router) {
		sr.Post("/", api.createSecret)
		sr.Get("/", api.listSecrets)
		sr.Get("/{id}", api.getSecret)
		sr.Put("/{id}", api.updateSecret)
		sr.Delete("/{id}", api.deleteSecret)
		sr.Post("/{id}/export/k8s", api.exportK8s)
	})

	log.Printf("listening on %s", cfg.Server.Addr)
	if err := http.ListenAndServe(cfg.Server.Addr, r); err != nil { log.Fatal(err) }
}

func convertUsers(cfg *config.Config) []struct{ Username, PasswordBcrypt string; Roles []string } {
	var out []struct{ Username, PasswordBcrypt string; Roles []string }
	for _, u := range cfg.Users {
		out = append(out, struct{ Username, PasswordBcrypt string; Roles []string }{u.Username, u.PasswordBcrypt, u.Roles})
	}
	return out
}

// Handlers

type loginReq struct { Username, Password string }

type loginResp struct { Token string `json:"token"` }

func (a *api) login(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { http.Error(w, "bad json", 400); return }
	u, err := a.users.Verify(req.Username, req.Password)
	if err != nil { a.log.Log(audit.Entry{Actor: req.Username, Action: "login", Outcome: "denied"}); http.Error(w, "unauthorized", 401); return }
	tok, _ := a.jwt.Issue(auth.User{Username: u.Username, Roles: u.Roles}, 8*time.Hour)
	a.log.Log(audit.Entry{Actor: u.Username, Action: "login", Outcome: "ok"})
	json.NewEncoder(w).Encode(loginResp{Token: tok})
}

type createReq struct { Name string `json:"name"`; Value string `json:"value"`; TTL string `json:"ttl,omitempty"`; Meta map[string]string `json:"meta"` }

type secretMeta struct { ID, Name string; CreatedAt, UpdatedAt time.Time; ExpiresAt *time.Time; Meta map[string]string }

func (a *api) createSecret(w http.ResponseWriter, r *http.Request) {
	var req createReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { http.Error(w, "bad json", 400); return }
	var ttl time.Duration
	if req.TTL != "" { d, err := time.ParseDuration(req.TTL); if err != nil { http.Error(w, "bad ttl", 400); return }; ttl = d }
	sec, err := a.vault.Create(req.Name, []byte(req.Value), ttl, req.Meta)
	if err != nil { http.Error(w, err.Error(), 400); return }
	a.log.Log(audit.Entry{Actor: who(r), Action: "create", Outcome: "ok", Target: sec.ID, Meta: map[string]string{"name": sec.Name}})
	json.NewEncoder(w).Encode(secretMeta{ID: sec.ID, Name: sec.Name, CreatedAt: sec.CreatedAt, UpdatedAt: sec.UpdatedAt, ExpiresAt: sec.ExpiresAt, Meta: sec.Meta})
}

func (a *api) listSecrets(w http.ResponseWriter, r *http.Request) {
	secs, err := a.store.List(false)
	if err != nil { http.Error(w, err.Error(), 500); return }
	out := make([]secretMeta, 0, len(secs))
	for _, s := range secs { out = append(out, secretMeta{ID: s.ID, Name: s.Name, CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt, ExpiresAt: s.ExpiresAt, Meta: s.Meta}) }
	json.NewEncoder(w).Encode(out)
}

func (a *api) getSecret(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sec, pt, err := a.vault.GetDecrypted(id)
	if err != nil { http.Error(w, err.Error(), 404); return }
	a.log.Log(audit.Entry{Actor: who(r), Action: "read", Outcome: "ok", Target: id})
	json.NewEncoder(w).Encode(struct { ID, Name, Value string }{ID: sec.ID, Name: sec.Name, Value: string(pt)})
}

type updateReq struct { Value *string `json:"value"`; TTL *string `json:"ttl"`; Meta map[string]string `json:"meta"` }

func (a *api) updateSecret(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req updateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { http.Error(w, "bad json", 400); return }
	var ttl *time.Duration
	if req.TTL != nil && *req.TTL != "" { if d, err := time.ParseDuration(*req.TTL); err == nil { ttl = &d } else { http.Error(w, "bad ttl", 400); return } }
	var plain []byte
	if req.Value != nil { plain = []byte(*req.Value) }
	sec, err := a.vault.Update(id, plain, ttl, req.Meta)
	if err != nil { http.Error(w, err.Error(), 400); return }
	a.log.Log(audit.Entry{Actor: who(r), Action: "update", Outcome: "ok", Target: id})
	json.NewEncoder(w).Encode(secretMeta{ID: sec.ID, Name: sec.Name, CreatedAt: sec.CreatedAt, UpdatedAt: sec.UpdatedAt, ExpiresAt: sec.ExpiresAt, Meta: sec.Meta})
}

func (a *api) deleteSecret(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := a.store.Delete(id); err != nil { http.Error(w, err.Error(), 404); return }
	a.log.Log(audit.Entry{Actor: who(r), Action: "delete", Outcome: "ok", Target: id})
	w.WriteHeader(204)
}

func (a *api) exportK8s(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct { Namespace, Key string }
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil { body.Namespace = "default" }
	b, err := a.vault.ExportK8sYAML(id, body.Namespace, body.Key)
	if err != nil { http.Error(w, err.Error(), 400); return }
	w.Header().Set("Content-Type", "application/x-yaml")
	w.Write(b)
}

func who(r *http.Request) string {
	if u, ok := auth.FromCtx(r.Context()); ok { return u.Username }
	return "?"
}