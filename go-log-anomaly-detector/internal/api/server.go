package api

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/viniciushammett/go-log-anomaly-detector/internal/detector"
	"github.com/viniciushammett/go-log-anomaly-detector/internal/logger"
	"github.com/viniciushammett/go-log-anomaly-detector/internal/metrics"
	"github.com/viniciushammett/go-log-anomaly-detector/internal/store"
)

var tracer = otel.Tracer("api")

type Deps struct {
	Log       *logger.Logger
	Store     *store.Store
	Ingest    func(source, msg string, meta map[string]string, ts time.Time)
	Detector  *detector.Detector
	AuthToken string
}
type Config struct{ Addr string }
type Server struct{ d Deps; c Config }

func NewServer(d Deps, c Config) *Server { return &Server{d: d, c: c} }

func (s *Server) Run(ctx context.Context) error {
	r := chi.NewRouter()
	r.Get("/healthz", func(w http.ResponseWriter,_ *http.Request){ _,_ = w.Write([]byte("ok")) })
	r.Get("/metrics", func(w http.ResponseWriter, r *http.Request){ metrics.Handler().ServeHTTP(w,r) })
	r.Post("/v1/logs", s.handleLogs)
	r.Get("/v1/anomalies", s.handleList)

	// Servir frontend compilado se existir ./frontend/dist
	if stat, err := os.Stat("frontend/dist"); err == nil && stat.IsDir() {
		fs := http.FileServer(http.Dir("frontend/dist"))
		r.Handle("/ui/*", http.StripPrefix("/ui/", fs))
		r.Handle("/ui", http.StripPrefix("/ui", fs))
	}

	srv := api.NewServer(api.Deps{
  Log: log, Store: db, Ingest: ing.Submit, Detector: det, AuthToken: cfg.AuthToken,
}, api.Config{Addr: cfg.Server.Addr})

}

func (s *Server) auth(r *http.Request) bool {
	if s.d.AuthToken == "" { return true }
	got := r.Header.Get("Authorization")
	return strings.HasPrefix(got, "Bearer ") && strings.TrimPrefix(got, "Bearer ") == s.d.AuthToken
}

type logPayload struct {
	Source string            `json:"source"`
	Msg    string            `json:"msg"`
	Meta   map[string]string `json:"meta"`
	TS     *time.Time        `json:"ts,omitempty"`
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "POST /v1/logs")
	defer span.End()

	if !s.auth(r) { http.Error(w, "unauthorized", http.StatusUnauthorized); return }
	var p logPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil || p.Msg == "" {
		http.Error(w, "invalid payload", http.StatusBadRequest); return
	}
	ts := time.Now()
	if p.TS != nil { ts = *p.TS }

	span.SetAttributes(
		attribute.String("source", p.Source),
		attribute.Int("meta_len", len(p.Meta)),
	)

	s.d.Ingest(p.Source, p.Msg, p.Meta, ts)
	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) handleList(w http.ResponseWriter, r *http.Request) {
	_, span := tracer.Start(r.Context(), "GET /v1/anomalies")
	defer span.End()

	arr, _ := s.d.Store.List(200)
	w.Header().Set("Content-Type","application/json")
	_ = json.NewEncoder(w).Encode(arr)
}
