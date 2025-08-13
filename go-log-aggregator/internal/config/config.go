package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Filesrc struct {
	Name         string        `yaml:"name"`
	Path         string        `yaml:"path"`
	PollInterval time.Duration `yaml:"pollInterval"` // ex: 250ms, 1s
}
type HTTPSrc struct {
	Name     string        `yaml:"name"`
	URL      string        `yaml:"url"`
	Interval time.Duration `yaml:"interval"` // ex: 2s, 5s
}
type StdinCfg struct {
	Enabled bool `yaml:"enabled"`
}

type SourceSet struct {
	Files []Filesrc `yaml:"files"`
	HTTP  []HTTPSrc `yaml:"http"`
	Stdin StdinCfg  `yaml:"stdin"`
}

type Config struct {
	BufferSize int       `yaml:"bufferSize"` // linhas
	Sources    SourceSet `yaml:"sources"`
}

func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path); if err != nil { return nil, err }
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil { return nil, err }
	if c.BufferSize == 0 { c.BufferSize = 5000 }
	return &c, nil
}