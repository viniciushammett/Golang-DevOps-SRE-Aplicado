package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/viniciushammett/go-access-auditor/internal/api"
	"github.com/viniciushammett/go-access-auditor/internal/ingest"
	"github.com/viniciushammett/go-access-auditor/internal/logger"
	"github.com/viniciushammett/go-access-auditor/internal/metrics"
	"github.com/viniciushammett/go-access-auditor/internal/rules"
	"github.com/viniciushammett/go-access-auditor/internal/store"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Addr string `yaml:"addr"`
	} `yaml:"server"`
	AuthToken string `yaml:"authToken"`
	Storage   struct{ Path string `yaml:"path"` } `yaml:"storage"`
	Rules     []rules.Rule `yaml:"rules"`
	Slack     struct {
		Webhook string `yaml:"webhook"`
		Channel string `yaml:"channel"`
		Enabled bool   `yaml:"enabled"`
	} `yaml:"slack"`
}

func main() {
	log := logger.New(env("LOG_LEVEL", "info"))
	cfgPath := env("CONFIG_PATH", "configs/config.yaml")

	var cfg Config
	if b, err := os.ReadFile(cfgPath); err == nil {
		_ = yaml.Unmarshal(b, &cfg)
	}
	if cfg.Server.Addr == "" { cfg.Server.Addr = ":8080" }
	if cfg.Storage.Path == "" { cfg.Storage.Path = "data/auditor.db" }

	metrics.MustRegister()

	db, err := store.Open(cfg.Storage.Path)
	if err != nil { log.Fatal().Err(err).Msg("open store") }
	defer db.Close()

	ruleSet := rules.New(cfg.Rules)
	processor := ingest.NewProcessor(log, db, ruleSet, cfg.Slack.Enabled, cfg.Slack.Webhook)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { c:=make(chan os.Signal,1); signal.Notify(c,syscall.SIGINT,syscall.SIGTERM); <-c; cancel() }()

	srv := api.NewServer(api.Deps{
		Log: log, Store: db, Proc: processor,
		AuthToken: cfg.AuthToken,
	}, api.Config{Addr: cfg.Server.Addr})
	if err := srv.Run(ctx); err != nil {
		log.Error().Err(err).Msg("server stopped")
	}
}

func env(k,d string) string { if v:=os.Getenv(k); v!="" {return v}; return d }