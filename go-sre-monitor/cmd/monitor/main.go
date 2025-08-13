package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/viniciushammett/go-sre-monitor/internal/config"
	"github.com/viniciushammett/go-sre-monitor/internal/httpserver"
	"github.com/viniciushammett/go-sre-monitor/internal/logger"
	"github.com/viniciushammett/go-sre-monitor/internal/metrics"
	"github.com/viniciushammett/go-sre-monitor/internal/monitor"
)

func main() {
	// ENV padrão
	cfgPath := getEnv("CONFIG_PATH", "configs/services.yaml")
	addr := getEnv("HTTP_ADDR", ":8080")
	logLevel := getEnv("LOG_LEVEL", "info")
	probeInterval := getEnvDuration("PROBE_INTERVAL", 15*time.Second)

	logg := logger.New(logLevel)
	logg.Info().Str("config", cfgPath).Str("addr", addr).Dur("interval", probeInterval).Msg("starting go-sre-monitor")

	// Carrega config
	cfg, err := config.Load(cfgPath)
	if err != nil {
		logg.Fatal().Err(err).Msg("failed to load config")
	}

	// Registra métricas
	metrics.MustRegister()

	// Inicia monitor
	mon := monitor.New(logg, cfg, probeInterval)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// HTTP server com /metrics e /healthz
	srv := httpserver.New(addr, logg)

	// Sinais p/ shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		logg.Warn().Msg("signal received, shutting down...")
		cancel()
		shutdown(ctx, mon, srv, logg)
	}()

	// Run background
	go mon.Run(ctx)

	// Start HTTP (bloqueante)
	if err := srv.Start(); err != nil {
		logg.Fatal().Err(err).Msg("http server failed")
	}
}

func shutdown(ctx context.Context, mon *monitor.Monitor, srv *httpserver.Server, logg *logger.Logger) {
	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := srv.Stop(ctxTimeout); err != nil {
		logg.Error().Err(err).Msg("http server graceful stop failed")
	}
	logg.Info().Msg("shutdown complete")
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		d, err := time.ParseDuration(v)
		if err == nil {
			return d
		}
		log.Printf("invalid duration for %s=%s, using default %s", key, v, def)
	}
	return def
}
