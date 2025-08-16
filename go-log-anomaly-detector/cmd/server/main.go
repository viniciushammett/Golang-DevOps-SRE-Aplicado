package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/viniciushammett/go-log-anomaly-detector/internal/api"
	"github.com/viniciushammett/go-log-anomaly-detector/internal/detector"
	"github.com/viniciushammett/go-log-anomaly-detector/internal/ingest"
	"github.com/viniciushammett/go-log-anomaly-detector/internal/logger"
	"github.com/viniciushammett/go-log-anomaly-detector/internal/metrics"
	"github.com/viniciushammett/go-log-anomaly-detector/internal/notify"
	"github.com/viniciushammett/go-log-anomaly-detector/internal/rules"
	"github.com/viniciushammett/go-log-anomaly-detector/internal/store"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Addr string `yaml:"addr"`
	} `yaml:"server"`
	Storage struct {
		Path string `yaml:"path"`
	} `yaml:"storage"`
	AuthToken string `yaml:"authToken"`
	RulesFile string `yaml:"rulesFile"`
	Slack struct {
		Enabled bool   `yaml:"enabled"`
		Webhook string `yaml:"webhook"`
	} `yaml:"slack"`
}

func main() {
	log := logger.New(env("LOG_LEVEL","info"))
	cfgPath := env("CONFIG_PATH","configs/config.yaml")

	var cfg Config
	if b, err := os.ReadFile(cfgPath); err == nil { _ = yaml.Unmarshal(b, &cfg) }
	if cfg.Server.Addr == "" { cfg.Server.Addr = ":8080" }
	if cfg.Storage.Path == "" { cfg.Storage.Path = "data/log-anomaly.db" }

	metrics.MustRegister()

	db, err := store.Open(cfg.Storage.Path); if err != nil { log.Fatal().Err(err).Msg("open store") }
	defer db.Close()

	// load rules
	ruleSet, err := rules.LoadFromFile(cfg.RulesFile)
	if err != nil { log.Fatal().Err(err).Str("file", cfg.RulesFile).Msg("load rules") }

	// notifier
	notifier := notify.NewSlack(cfg.Slack.Enabled, cfg.Slack.Webhook)

	// detector
	det := detector.New(log, db, ruleSet, notifier)

	// ingestor (HTTP + stubs Kafka/File)
	ing := ingest.New(det)

	ctx, cancel := context.WithCancel(context.Background()); defer cancel()
	go func(){ c:=make(chan os.Signal,1); signal.Notify(c, syscall.SIGINT, syscall.SIGTERM); <-c; cancel() }()

	srv := api.NewServer(api.Deps{
		Log: log, Store: db, Ingest: ing, Detector: det,
		AuthToken: cfg.AuthToken,
	}, api.Config{Addr: cfg.Server.Addr})
	if err := srv.Run(ctx); err != nil { log.Error().Err(err).Msg("server stopped") }
}

func env(k,d string) string { if v:=os.Getenv(k); v!=""){return v}; return d }