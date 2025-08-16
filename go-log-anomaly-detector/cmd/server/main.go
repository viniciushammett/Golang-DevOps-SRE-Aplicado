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
	"github.com/viniciushammett/go-log-anomaly-detector/internal/tracing"
	"github.com/viniciushammett/go-log-anomaly-detector/internal/ml"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct{ Addr string `yaml:"addr"` } `yaml:"server"`
	Storage struct{ Path string `yaml:"path"` } `yaml:"storage"`
	AuthToken string `yaml:"authToken"`
	RulesFile string `yaml:"rulesFile"`
	Slack struct { Enabled bool `yaml:"enabled"`; Webhook string `yaml:"webhook"` } `yaml:"slack"`
	Tracing struct {
		Enabled bool `yaml:"enabled"`
		ServiceName string `yaml:"serviceName"`
		OTLPEndpoint string `yaml:"otlpEndpoint"`
		SampleRatio float64 `yaml:"sampleRatio"`
	} `yaml:"tracing"`
	ML struct {
		Enabled bool `yaml:"enabled"`
		ModelPath string `yaml:"modelPath"`
		ZScoreK float64 `yaml:"zScoreK"`
		Bucket string `yaml:"bucket"`
	} `yaml:"ml"`
}

func main() {
	log := logger.New(env("LOG_LEVEL","info"))
	cfgPath := env("CONFIG_PATH","configs/config.yaml")

	var cfg Config
	if b, err := os.ReadFile(cfgPath); err == nil { _ = yaml.Unmarshal(b, &cfg) }
	if cfg.Server.Addr == "" { cfg.Server.Addr = ":8080" }
	if cfg.Storage.Path == "" { cfg.Storage.Path = "data/log-anomaly.db" }

	metrics.MustRegister()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Tracing
	closer, err := tracing.Init(ctx, tracing.Config{
		Enabled: cfg.Tracing.Enabled,
		ServiceName: first(cfg.Tracing.ServiceName, "go-log-anomaly-detector"),
		OTLPEndpoint: first(cfg.Tracing.OTLPEndpoint, "localhost:4317"),
		SampleRatio: ifzero(cfg.Tracing.SampleRatio, 1.0),
	})
	if err != nil { log.Error().Err(err).Msg("tracing init failed") }
	defer func(){ _ = closer(context.Background()) }()

	// Store
	db, err := store.Open(cfg.Storage.Path); if err != nil { log.Fatal().Err(err).Msg("open store") }
	defer db.Close()

	// Rules
	ruleSet, err := rules.LoadFromFile(cfg.RulesFile); if err != nil { log.Fatal().Err(err).Msg("load rules") }

	// Notifier
	notifier := notify.NewSlack(cfg.Slack.Enabled, cfg.Slack.Webhook)

	// ML (opcional)
	var mlDet *ml.Detector
	if cfg.ML.Enabled {
		mlDet, err = ml.NewDetector(cfg.ML.ModelPath, ifzero(cfg.ML.ZScoreK, 3.0), first(cfg.ML.Bucket, "1m"))
		if err != nil { log.Error().Err(err).Msg("ml init failed") }
	}

	// Detector
	det := detector.New(log, db, ruleSet, notifier, mlDet)

	// Ingest
	ing := ingest.New(det, db)

	// API
	srv := api.NewServer(api.Deps{
		Log: log, Store: db, Ingest: ing, Detector: det, AuthToken: cfg.AuthToken,
	}, api.Config{Addr: cfg.Server.Addr})
	if err := srv.Run(ctx); err != nil { log.Error().Err(err).Msg("server stopped") }
}

func env(k,d string) string { if v:=os.Getenv(k); v!=""{return v}; return d }
func ifzero(f,def float64) float64 { if f==0 {return def}; return f }
func first(s,def string) string { if s=="" {return def}; return s }
