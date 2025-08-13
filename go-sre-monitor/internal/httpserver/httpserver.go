package httpserver

import (
	"context"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/viniciushammett/go-sre-monitor/internal/logger"
)

type Server struct {
	srv *http.Server
	log *logger.Logger
}

func New(addr string, log *logger.Logger) *Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("go-sre-monitor up"))
	})

	return &Server{
		srv: &http.Server{
			Addr:              addr,
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
		},
		log: log,
	}
}

func (s *Server) Start() error {
	s.log.Info().Str("addr", s.srv.Addr).Msg("http server listening")
	return s.srv.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) error {
	s.log.Info().Msg("stopping http server")
	return s.srv.Shutdown(ctx)
}