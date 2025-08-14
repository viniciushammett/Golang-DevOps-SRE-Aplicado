package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/viniciushammett/go-deploy-orchestrator/internal/config"
	"github.com/viniciushammett/go-deploy-orchestrator/internal/logger"
	"github.com/viniciushammett/go-deploy-orchestrator/internal/metrics"
	"github.com/viniciushammett/go-deploy-orchestrator/internal/orchestrator"
	"github.com/viniciushammett/go-deploy-orchestrator/internal/store"
)

type Deps struct {
	Log   *logger.Logger
	Orc   *orchestrator.Orchestrator
	Store *store.Store
	Cfg   *config.Config
}
type Config struct{ Addr string }

type Server struct {
	d Deps
	c Config
}

func NewServer(d Deps, c Config) *Server { return &Server{d: d, c: c} }

func (s *Server) Run(ctx context.Context) error {
	r := chi.NewRouter()
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) })
	r.Get("/metrics", func(w http.ResponseWriter, r *http.Request) { metrics.Handler().ServeHTTP(w, r) })
	r.Post("/deploys", s.handleStart)
	r.Get("/deploys", s.handleList)
	r.Get("/deploys/{id}", s.handleGet)
	r.Post("/deploys/{id}/approve", s.auth(s.handleApprove))

	srv := &http.Server{Addr: s.c.Addr, Handler: s.d.Log.HTTP(r)}
	go func(){ <-ctx.Done(); _ = srv.Shutdown(context.Background()) }()
	s.d.Log.Info().Str("addr", s.c.Addr).Msg("http listening")
	return srv.ListenAndServe()
}

type deployReq struct {
	App       string            `json:"app"`
	Namespace string            `json:"namespace"`
	Image     string            `json:"image"`
	Strategy  string            `json:"strategy"`
	Params    map[string]string `json:"params"`
	RequireApproval bool        `json:"requireApproval"`
}

func (s *Server) handleStart(w http.ResponseWriter, r *http.Request) {
	var req deployReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.App=="" || req.Image=="" {
		http.Error(w, "invalid deploy payload", http.StatusBadRequest); return
	}
	rec, err := s.d.Orc.StartDeploy(r.Context(), orchestrator.DeployInput{
		App: req.App, Namespace: req.Namespace, Image: req.Image, Strategy: req.Strategy, Params: req.Params, RequireApproval: req.RequireApproval,
	})
	if err != nil { http.Error(w, err.Error(), http.StatusInternalServerError); return }
	w.Header().Set("Content-Type","application/json")
	_ = json.NewEncoder(w).Encode(rec)
}

func (s *Server) handleList(w http.ResponseWriter, r *http.Request) {
	arr, _ := s.d.Store.List()
	w.Header().Set("Content-Type","application/json")
	_ = json.NewEncoder(w).Encode(arr)
}
func (s *Server) handleGet(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	rec, _ := s.d.Store.Get(id)
	if rec == nil { http.Error(w, "not found", http.StatusNotFound); return }
	w.Header().Set("Content-Type","application/json")
	_ = json.NewEncoder(w).Encode(rec)
}

func (s *Server) handleApprove(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	rec, _ := s.d.Store.Get(id)
	if rec == nil { http.Error(w, "not found", http.StatusNotFound); return }
	if rec.Status != "waiting_approval" { http.Error(w, "not waiting approval", http.StatusBadRequest); return }
	rec.Status = "running"
	_ = s.d.Store.Put(*rec)
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	if s.d.Cfg.AuthToken == "" { return next }
	return func(w http.ResponseWriter, r *http.Request) {
		got := r.Header.Get("Authorization")
		if !strings.HasPrefix(got, "Bearer ") || strings.TrimPrefix(got, "Bearer ") != s.d.Cfg.AuthToken {
			http.Error(w, "unauthorized", http.StatusUnauthorized); return
		}
		next(w, r)
	}
}