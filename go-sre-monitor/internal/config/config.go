package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Service struct {
	Name           string        `yaml:"name"`
	URL            string        `yaml:"url"`
	Method         string        `yaml:"method"`          // GET/HEAD
	ExpectedStatus int           `yaml:"expectedStatus"`  // ex: 200
	Timeout        time.Duration `yaml:"timeout"`         // ex: 3s
	SLOLatency     time.Duration `yaml:"sloLatency"`      // ex: 500ms
	Headers        map[string]string `yaml:"headers"`     // opcional
}

type Config struct {
	Services []Service `yaml:"services"`
}

func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("unmarshal yaml: %w", err)
	}

	// Defaults mínimos
	for i := range c.Services {
		if c.Services[i].Method == "" {
			c.Services[i].Method = "GET"
		}
		if c.Services[i].Timeout == 0 {
			c.Services[i].Timeout = 3 * time.Second
		}
		if c.Services[i].ExpectedStatus == 0 {
			c.Services[i].ExpectedStatus = 200
		}
	}
	return &c, nil
}