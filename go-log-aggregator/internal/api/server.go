package api

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/viniciushammett/go-log-aggregator/internal/buffer"
	"github.com/viniciushammett/go-log-aggregator/internal/filter"
	"github.com/viniciushammett/go-log-aggregator/internal/logger"
	"github.com/viniciushammett/go-log-aggregator/internal/metrics"
)

type Config struct {
	Addr string
}
type Deps struct {
	Log  *logger.Logger
	Ring *buffer.Ring
}

type Server struct {
	cfg  Config
	deps Deps
}

func NewServer(d Deps, c Config) *Server { return &Server{cfg: c, deps: d} }

func (s *Server) Run(ctx context.Context) error {
	r := chi.NewRouter()
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) })
	r.Get("/metrics", func(w http.ResponseWriter, r *http.Request) { metrics.Handler().ServeHTTP(w, r) })
	r.Get("/logs", s.handleLogs)

	srv := &http.Server{ Addr: s.cfg.Addr, Handler: s.deps.Log.HTTP(r) }
	go func(){ <-ctx.Done(); _ = srv.Shutdown(context.Background()) }()
	return srv.ListenAndServe()
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	// time range (RFC3339 or seconds offset)
	sinceStr := q.Get("since")
	untilStr := q.Get("until")
	var since, until time.Time
	if sinceStr != "" {
		if secs, err := strconv.Atoi(sinceStr); err == nil {
			since = time.Now().Add(-time.Duration(secs)*time.Second)
		} else if t, err := time.Parse(time.RFC3339, sinceStr); err == nil {
			since = t
		}
	}
	if untilStr != "" {
		if t, err := time.Parse(time.RFC3339, untilStr); err == nil {
			until = t
		}
	}

	// filters
	var inc, exc *regexp.Regexp
	var err error
	if inc, err = filter.CompileOrNil(q.Get("include")); err != nil {
		http.Error(w, "bad include regex", http.StatusBadRequest); return
	}
	if exc, err = filter.CompileOrNil(q.Get("exclude")); err != nil {
		http.Error(w, "bad exclude regex", http.StatusBadRequest); return
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	src := q.Get("source")

	events := s.deps.Ring.Snapshot(buffer.Query{
		Since: since, Until: until, Include: inc, Exclude: exc,
		SourceEqual: src, Limit: limit, Offset: offset,
	})
	metrics.APIQueries.WithLabelValues("/logs").Inc()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(events)
}