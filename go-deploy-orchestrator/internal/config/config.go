package config

import (
	"os"
	"time"
	"gopkg.in/yaml.v3"
)

type Kube struct {
	// se vazio, tenta in-cluster; senão, usa kubeconfig
	Kubeconfig string `yaml:"kubeconfig"`
	Context    string `yaml:"context"`
}

type PromCfg struct {
	URL     string        `yaml:"url"`     // http://prometheus:9090
	Timeout time.Duration `yaml:"timeout"` // ex: 10s
	Queries struct {
		ErrorRate string `yaml:"errorRate"` // promQL para erro (retorna fração 0..1)
		P95       string `yaml:"p95"`       // promQL para latência p95 em segundos
	} `yaml:"queries"`
	Thresholds struct {
		MaxError float64 `yaml:"maxError"`  // ex: 0.02
		MaxP95   float64 `yaml:"maxP95"`    // ex: 0.5s
	} `yaml:"thresholds"`
	Window string `yaml:"window"` // ex: 5m
}

type Storage struct {
	Path string `yaml:"path"` // BoltDB
}

type Defaults struct {
	CanaryStepPercent int `yaml:"canaryStepPercent"` // ex: 20
	CanaryPauseSec    int `yaml:"canaryPauseSec"`    // ex: 60
}

type Config struct {
	Server struct {
		HTTPAddr string `yaml:"httpAddr"`
	} `yaml:"server"`
	Kube       Kube      `yaml:"kube"`
	Prometheus PromCfg   `yaml:"prometheus"`
	Storage    Storage   `yaml:"storage"`
	Defaults   Defaults  `yaml:"defaults"`
	AuthToken  string    `yaml:"authToken"` // opcional p/ rotas admin
}

func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path); if err != nil { return nil, err }
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil { return nil, err }
	if c.Server.HTTPAddr == "" { c.Server.HTTPAddr = ":8080" }
	if c.Storage.Path == "" { c.Storage.Path = "data/deploy-orchestrator.db" }
	if c.Prometheus.Timeout == 0 { c.Prometheus.Timeout = 10 * time.Second }
	if c.Defaults.CanaryStepPercent == 0 { c.Defaults.CanaryStepPercent = 20 }
	if c.Defaults.CanaryPauseSec == 0 { c.Defaults.CanaryPauseSec = 60 }
	return &c, nil
}