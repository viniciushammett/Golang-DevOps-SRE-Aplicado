package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/viniciushammett/go-alert-router/internal/config"
	"github.com/viniciushammett/go-alert-router/internal/logger"
	"github.com/viniciushammett/go-alert-router/internal/metrics"
	"github.com/viniciushammett/go-alert-router/internal/model"
	"github.com/viniciushammett/go-alert-router/internal/router"
	"github.com/viniciushammett/go-alert-router/internal/store"
)

type Deps struct {
	Log    *logger.Logger
	Router *router.Router
	Store  *store.Store
	Config *config.Config
}
type Config struct {
	Addr string
}
type Server struct {
	deps Deps
	cfg  Config
}

func NewServer(d Deps, c Config) *Server { return &Server{deps: d, cfg: c} }

func (s *Server) Run(ctx context.Context) error {
	r := chi.NewRouter()
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) })
	r.Get("/metrics", func(w http.ResponseWriter, r *http.Request) { metrics.Handler().ServeHTTP(w, r) })
	r.Post("/webhook/alertmanager", s.handleAlertmanager)
	r.Post("/admin/silences", s.auth(s.handlePutSilence))
	r.Get("/admin/silences", s.auth(s.handleListSilences))

	srv := &http.Server{Addr: s.cfg.Addr, Handler: s.deps.Log.HTTP(r)}
	go func(){ <-ctx.Done(); _ = srv.Shutdown(context.Background()) }()
	s.deps.Log.Info().Str("addr", s.cfg.Addr).Msg("http listening")
	return srv.ListenAndServe()
}

func (s *Server) handleAlertmanager(w http.ResponseWriter, r *http.Request) {
	var p model.WebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest); return
	}
	for i := range p.Alerts { p.Alerts[i].EnsureFingerprint() }
	s.deps.Router.Ingest(p.Alerts, "alertmanager")
	_, _ = w.Write([]byte("ok"))
}

type silenceReq struct {
	Label string `json:"label"`
	Regex string `json:"regex"`
	Until time.Time `json:"until"`
	ID    string `json:"id"`
}
func (s *Server) handlePutSilence(w http.ResponseWriter, r *http.Request) {
	var req silenceReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Label=="" || req.Regex=="" || req.Until.IsZero() {
		http.Error(w, "bad silence", http.StatusBadRequest); return
	}
	if req.ID == "" { req.ID = req.Label + ":" + req.Regex }
	_ = s.deps.Store.PutSilence(store.Silence{
		ID: req.ID, Label: req.Label, Regex: req.Regex, Until: req.Until,
	})
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write([]byte("ok"))
}
func (s *Server) handleListSilences(w http.ResponseWriter, r *http.Request) {
	sils, _ := s.deps.Store.ListSilences()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(sils)
}

func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	if s.deps.Config.HTTPAuthToken == "" { return next }
	return func(w http.ResponseWriter, r *http.Request) {
		got := r.Header.Get("Authorization")
		if !strings.HasPrefix(got, "Bearer ") || strings.TrimPrefix(got, "Bearer ") != s.deps.Config.HTTPAuthToken {
			http.Error(w, "unauthorized", http.StatusUnauthorized); return
		}
		next(w, r)
	}
}