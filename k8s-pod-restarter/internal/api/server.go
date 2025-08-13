package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/viniciushammett/k8s-pod-restarter/internal/logger"
	"github.com/viniciushammett/k8s-pod-restarter/internal/restarter"
	"k8s.io/client-go/kubernetes"
)

type ServerDeps struct {
	Log       *logger.Logger
	Clientset *kubernetes.Clientset
}

type Config struct {
	Addr      string
	AuthToken string
}

type Server struct {
	cfg  Config
	deps ServerDeps
}

func NewServer(deps ServerDeps, cfg Config) *Server {
	return &Server{cfg: cfg, deps: deps}
}

func (s *Server) Run(ctx context.Context) error {
	r := chi.NewRouter()
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) })
	r.Post("/restart", s.auth(s.handleRestart))

	srv := &http.Server{
		Addr:    s.cfg.Addr,
		Handler: s.deps.Log.HTTPLogger(r),
	}
	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.Background())
	}()
	return srv.ListenAndServe()
}

func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	if s.cfg.AuthToken == "" {
		return next
	}
	return func(w http.ResponseWriter, r *http.Request) {
		got := r.Header.Get("Authorization")
		if !strings.HasPrefix(got, "Bearer ") || strings.TrimPrefix(got, "Bearer ") != s.cfg.AuthToken {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("unauthorized"))
			return
		}
		next(w, r)
	}
}

type restartReq struct {
	Namespace   string        `json:"namespace"`
	Selector    string        `json:"selector"` // e.g. app=myapp
	DryRun      bool          `json:"dryRun"`
	Force       bool          `json:"force"`
	MaxAge      time.Duration `json:"maxAge"`
	GracePeriod time.Duration `json:"gracePeriod"`
}

func (s *Server) handleRestart(w http.ResponseWriter, r *http.Request) {
	var req restartReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Selector == "" {
		http.Error(w, "invalid payload (selector is required)", http.StatusBadRequest)
		return
	}
	rest := restarter.New(s.deps.Log, s.deps.Clientset)
	err := rest.RestartPods(r.Context(), restarter.Options{
		Namespace:   req.Namespace,
		Selector:    req.Selector,
		DryRun:      req.DryRun,
		Force:       req.Force,
		MaxAge:      req.MaxAge,
		GracePeriod: req.GracePeriod,
	})
	if err != nil {
		http.Error(w, "restart failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
}