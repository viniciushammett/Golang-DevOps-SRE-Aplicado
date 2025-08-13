package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/viniciushammett/go-alert-router/internal/api"
	"github.com/viniciushammett/go-alert-router/internal/config"
	"github.com/viniciushammett/go-alert-router/internal/logger"
	"github.com/viniciushammett/go-alert-router/internal/metrics"
	"github.com/viniciushammett/go-alert-router/internal/notify"
	"github.com/viniciushammett/go-alert-router/internal/router"
	"github.com/viniciushammett/go-alert-router/internal/scheduler"
	"github.com/viniciushammett/go-alert-router/internal/store"
)

func main() {
	cfgPath := getenv("CONFIG_PATH", "configs/config.yaml")
	httpAddr := getenv("HTTP_ADDR", ":8080")
	logLevel := getenv("LOG_LEVEL", "info")

	log := logger.New(logLevel)
	log.Info().Str("cfg", cfgPath).Str("http", httpAddr).Msg("starting go-alert-router")

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatal().Err(err).Msg("load config")
	}

	metrics.MustRegister()

	// Store (bbolt)
	db, err := store.Open(cfg.Storage.Path)
	if err != nil {
		log.Fatal().Err(err).Msg("open store")
	}
	defer db.Close()

	// Notifiers
	slack := notify.NewSlack(log)
	email := notify.NewEmail(log, cfg.Email)

	// Router
	rt := router.New(log, db, cfg, slack, email)

	// Context + signals
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		<-c
		cancel()
	}()

	// Background: workers + GC dos índices
	rt.Start(ctx)

	// Scheduler: expiração de silences e rotação de janelas
	scheduler.Start(ctx, log, db, time.Minute)

	// API Server
	srv := api.NewServer(api.Deps{
		Log:     log,
		Router:  rt,
		Store:   db,
		Config:  cfg,
	}, api.Config{ Addr: httpAddr })

	if err := srv.Run(ctx); err != nil {
		log.Error().Err(err).Msg("http server stopped")
	}
}

func getenv(k, d string) string { if v := os.Getenv(k); v != "" { return v }; return d }