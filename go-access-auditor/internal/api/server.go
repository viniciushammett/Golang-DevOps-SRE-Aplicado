package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/viniciushammett/go-access-auditor/internal/export"
	"github.com/viniciushammett/go-access-auditor/internal/ingest"
	"github.com/viniciushammett/go-access-auditor/internal/logger"
	"github.com/viniciushammett/go-access-auditor/internal/metrics"
	"github.com/viniciushammett/go-access-auditor/internal/store"
)

type Deps struct {
	Log       *logger.Logger
	Store     *store.Store
	Proc      *ingest.Processor
	AuthToken string
}
type Config struct{ Addr string }

type Server struct{ d Deps; c Config }

func NewServer(d Deps, c Config) *Server { return &Server{d: d, c: c} }

func (s *Server) Run(ctx context.Context) error {
	r := chi.NewRouter()
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) })
	r.Get("/metrics", func(w http.ResponseWriter, r *http.Request) { metrics.Handler().ServeHTTP(w, r) })
	r.Post("/v1/events", s.handleEvent)
	r.Get("/v1/events", s.handleList)
	r.Get("/v1/export.csv", s.handleCSV)
	r.Get("/", s.serveDashboard)

	srv := &http.Server{Addr: s.c.Addr, Handler: s.d.Log.HTTP(r)}
	go func(){ <-ctx.Done(); _ = srv.Shutdown(context.Background()) }()
	s.d.Log.Info().Str("addr", s.c.Addr).Msg("http listening")
	return srv.ListenAndServe()
}

func (s *Server) auth(w http.ResponseWriter, r *http.Request) bool {
	if s.d.AuthToken == "" { return true }
	got := r.Header.Get("Authorization")
	return strings.HasPrefix(got, "Bearer ") && strings.TrimPrefix(got, "Bearer ") == s.d.AuthToken
}

func (s *Server) handleEvent(w http.ResponseWriter, r *http.Request) {
	if !s.auth(w, r) { http.Error(w, "unauthorized", http.StatusUnauthorized); return }
	var in ingest.Incoming
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "bad payload", http.StatusBadRequest); return
	}
	rec, err := s.d.Proc.Handle(in)
	if err != nil { http.Error(w, err.Error(), http.StatusInternalServerError); return }
	w.Header().Set("Content-Type","application/json")
	_ = json.NewEncoder(w).Encode(rec)
}

func (s *Server) handleList(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	parseTime := func(k string) time.Time {
		if v := q.Get(k); v != "" {
			if t, err := time.Parse(time.RFC3339, v); err == nil { return t }
		}
		return time.Time{}
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	sensOnly := q.Get("sensitive") == "true"
	evs, _ := s.d.Store.List(store.Query{
		User: q.Get("user"), Host: q.Get("host"), Source: q.Get("source"),
		Since: parseTime("since"), Until: parseTime("until"),
		Text: q.Get("q"), Limit: limit, Offset: offset,
		SensitiveOnly: sensOnly,
	})
	w.Header().Set("Content-Type","application/json")
	_ = json.NewEncoder(w).Encode(evs)
}

func (s *Server) handleCSV(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	evs, _ := s.d.Store.List(store.Query{
		User: q.Get("user"), Host: q.Get("host"), Source: q.Get("source"),
		Text: q.Get("q"), SensitiveOnly: q.Get("sensitive")=="true",
	})
	w.Header().Set("Content-Type","text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition","attachment; filename=events.csv")
	_ = export.WriteCSV(w, evs)
}

func (s *Server) serveDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type","text/html; charset=utf-8")
	_, _ = w.Write([]byte(indexHTML))
}

const indexHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8" />
<title>Go Access Auditor</title>
<meta name="viewport" content="width=device-width, initial-scale=1" />
<script src="https://unpkg.com/htmx.org@1.9.12"></script>
<link rel="stylesheet" href="https://unpkg.com/@sakun/system.css" />
<style> table{width:100%} code{white-space:pre-wrap} .pill{padding:.15rem .5rem;border-radius:999px;background:#eee} </style>
</head>
<body>
<header><h2>Go Access Auditor</h2></header>
<section>
  <form id="f" hx-get="/v1/events" hx-target="#rows" hx-trigger="submit, load">
    <input type="text" name="q" placeholder="buscar comando...">
    <input type="text" name="user" placeholder="user">
    <input type="text" name="host" placeholder="host">
    <input type="text" name="source" placeholder="source (bash|ssh|kubectl)">
    <label><input type="checkbox" name="sensitive" value="true"> apenas sensíveis</label>
    <input type="hidden" name="limit" value="200">
    <button>Buscar</button>
    <a href="/v1/export.csv" target="_blank">Export CSV</a>
  </form>
</section>
<section>
  <table>
    <thead><tr><th>Quando</th><th>user@host</th><th>source</th><th>Cmd</th><th></th></tr></thead>
    <tbody id="rows" hx-get="/v1/events" hx-trigger="load delay:500ms">
      <tr><td colspan="5">Carregando...</td></tr>
    </tbody>
  </table>
</section>
<script>
document.body.addEventListener('htmx:afterOnLoad', function(evt){
  if(evt.detail.target.id==='rows'){
    try{
      const data = JSON.parse(evt.detail.xhr.responseText);
      evt.detail.target.innerHTML = data.map(e=>`
      <tr>
        <td>${new Date(e.when).toLocaleString()}</td>
        <td>${e.user}@${e.host}</td>
        <td><span class="pill">${e.source}</span></td>
        <td><code>${(e.command||'').replace(/</g,'&lt;')}</code></td>
        <td>${e.flagSensitive ? '🚨' : ''}</td>
      </tr>`).join('');
    }catch(e){}
  }
});
</script>
</body></html>`