package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/viniciushammett/go-log-anomaly-detector/internal/detector"
	"github.com/viniciushammett/go-log-anomaly-detector/internal/logger"
	"github.com/viniciushammett/go-log-anomaly-detector/internal/metrics"
	"github.com/viniciushammett/go-log-anomaly-detector/internal/store"
)

type Deps struct {
	Log       *logger.Logger
	Store     *store.Store
	Ingest    *ingestWrapper
	Detector  *detector.Detector
	AuthToken string
}
type Config struct{ Addr string }

type ingestWrapper struct{ s func(source, msg string, meta map[string]string, ts time.Time) }

func NewServer(d Deps, c Config) *Server {
	return &Server{
		d: Deps{
			Log: d.Log, Store: d.Store, Detector: d.Detector, AuthToken: d.AuthToken,
			Ingest: &ingestWrapper{s: d.Ingest},
		}, c: c,
	}
}

type Server struct{ d Deps; c Config }

func (s *Server) Run(ctx context.Context) error {
	r := chi.NewRouter()
	r.Get("/healthz", func(w http.ResponseWriter,_ *http.Request){ _,_ = w.Write([]byte("ok")) })
	r.Get("/metrics", func(w http.ResponseWriter, r *http.Request){ metrics.Handler().ServeHTTP(w,r) })
	r.Post("/v1/logs", s.handleLogs)
	r.Get("/v1/anomalies", s.handleList)
	srv := &http.Server{Addr: s.c.Addr, Handler: s.d.Log.HTTP(r)}
	go func(){ <-ctx.Done(); _ = srv.Shutdown(context.Background()) }()
	s.d.Log.Info().Str("addr", s.c.Addr).Msg("http listening")
	return srv.ListenAndServe()
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
	if !s.auth(r) { http.Error(w, "unauthorized", http.StatusUnauthorized); return }
	var p logPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil || p.Msg == "" {
		http.Error(w, "invalid payload", http.StatusBadRequest); return
	}
	ts := time.Now()
	if p.TS != nil { ts = *p.TS }
	s.d.Ingest.s(p.Source, p.Msg, p.Meta, ts)
	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) handleList(w http.ResponseWriter, r *http.Request) {
	arr, _ := s.d.Store.List(200)
	w.Header().Set("Content-Type","application/json")
	_ = json.NewEncoder(w).Encode(arr)
}