package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/viniciushammett/go-deploy-orchestrator/internal/api"
	"github.com/viniciushammett/go-deploy-orchestrator/internal/config"
	"github.com/viniciushammett/go-deploy-orchestrator/internal/logger"
	"github.com/viniciushammett/go-deploy-orchestrator/internal/metrics"
	"github.com/viniciushammett/go-deploy-orchestrator/internal/orchestrator"
	"github.com/viniciushammett/go-deploy-orchestrator/internal/prometheus"
	"github.com/viniciushammett/go-deploy-orchestrator/internal/store"
)

func main() {
	cfgPath := getenv("CONFIG_PATH", "configs/config.yaml")
	httpAddr := getenv("HTTP_ADDR", ":8080")
	logLevel := getenv("LOG_LEVEL", "info")

	log := logger.New(logLevel)
	log.Info().Str("cfg", cfgPath).Str("http", httpAddr).Msg("starting deploy orchestrator")

	cfg, err := config.Load(cfgPath)
	if err != nil { log.Fatal().Err(err).Msg("load config") }

	metrics.MustRegister()

	db, err := store.Open(cfg.Storage.Path)
	if err != nil { log.Fatal().Err(err).Msg("open store") }
	defer db.Close()

	prom := prometheus.NewEvaluator(cfg.Prometheus.URL, cfg.Prometheus.Timeout)

	orc, err := orchestrator.New(log, cfg, db, prom)
	if err != nil { log.Fatal().Err(err).Msg("init orchestrator") }

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		<-c
		cancel()
	}()

	srv := api.NewServer(api.Deps{
		Log:   log,
		Orc:   orc,
		Store: db,
		Cfg:   cfg,
	}, api.Config{Addr: httpAddr})

	if err := srv.Run(ctx); err != nil {
		log.Error().Err(err).Msg("http server stopped")
	}
}

func getenv(k, d string) string { if v := os.Getenv(k); v != "" { return v }; return d }