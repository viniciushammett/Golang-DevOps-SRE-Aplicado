package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Job struct {
	Name        string `yaml:"name"`
	Namespace   string `yaml:"namespace"`
	Selector    string `yaml:"selector"`
	Schedule    string `yaml:"schedule"`    // cron, e.g. "0 3 * * *"
	DryRun      bool   `yaml:"dryRun"`
	Force       bool   `yaml:"force"`
	MaxAge      string `yaml:"maxAge"`      // parse later (e.g. "1h")
	GracePeriod string `yaml:"gracePeriod"` // e.g. "30s"
}

type Config struct {
	Jobs []Job `yaml:"jobs"`
}

func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	return &c, nil
}