package main

import (
	"context"
	"os"
	"os/signal"
	"time"
	"syscall"

	"github.com/viniciushammett/go-log-aggregator/internal/api"
	"github.com/viniciushammett/go-log-aggregator/internal/buffer"
	"github.com/viniciushammett/go-log-aggregator/internal/config"
	"github.com/viniciushammett/go-log-aggregator/internal/filter"
	"github.com/viniciushammett/go-log-aggregator/internal/logger"
	"github.com/viniciushammett/go-log-aggregator/internal/metrics"
	"github.com/viniciushammett/go-log-aggregator/internal/sources"
)

func main() {
	// ENV defaults
	httpAddr := getenv("HTTP_ADDR", ":8080")
	cfgPath  := getenv("CONFIG_PATH", "configs/config.yaml")
	logLevel := getenv("LOG_LEVEL", "info")

	log := logger.New(logLevel)
	log.Info().Str("cfg", cfgPath).Str("http", httpAddr).Msg("starting go-log-aggregator")

	// config
	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatal().Err(err).Msg("load config")
	}

	// metrics
	metrics.MustRegister()

	// ring buffer
	ring := buffer.NewRing(cfg.BufferSize)

	// context + signals
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	signals(cancel)

	// start sources
	srcMgr := sources.NewManager(log)
	// file tails
	for _, f := range cfg.Sources.Files {
		srcMgr.Add(sources.NewFileTail(f.Path, f.Name, time.Duration(f.PollInterval)))
	}
	// http pull
	for _, h := range cfg.Sources.HTTP {
		srcMgr.Add(sources.NewHTTPPull(h.URL, h.Name, time.Duration(h.Interval)))
	}
	// stdin
	if cfg.Sources.Stdin.Enabled {
		srcMgr.Add(sources.NewStdin("stdin"))
	}

	// fan-in loop
	go func() {
		ch := srcMgr.Run(ctx)
		for ev := range ch {
			start := time.Now()
			ring.Push(ev)
			metrics.LogsIngested.WithLabelValues(ev.Source).Inc()
			metrics.IngestLatency.WithLabelValues(ev.Source).Observe(time.Since(start).Seconds())
		}
	}()

	// API server
	srv := api.NewServer(api.Deps{
		Log:   log,
		Ring:  ring,
	}, api.Config{
		Addr: httpAddr,
	})
	if err := srv.Run(ctx); err != nil {
		log.Error().Err(err).Msg("api server stopped")
	}
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" { return v }
	return d
}
func signals(cancel func()) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	go func() { <-c; cancel() }()
}