package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Storage struct {
	Path string `yaml:"path"` // e.g. data/alert-router.db
}

type SlackRoute struct {
	Webhook string `yaml:"webhook"`
	Channel string `yaml:"channel"` // opcional (exibicao)
}

type EmailConfig struct {
	SMTPHost string `yaml:"smtpHost"`
	SMTPPort int    `yaml:"smtpPort"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	From     string `yaml:"from"`
}

type Matcher struct {
	Label string `yaml:"label"` // e.g. severity
	Regex string `yaml:"regex"` // e.g. ^(critical|high)$
}

type Route struct {
	Name           string        `yaml:"name"`
	Matchers       []Matcher     `yaml:"matchers"`
	DedupeWindow   time.Duration `yaml:"dedupeWindow"`   // e.g. 2m
	GroupWindow    time.Duration `yaml:"groupWindow"`    // e.g. 30s (agrupa antes de enviar)
	RateLimitPerMin int          `yaml:"rateLimitPerMin"`// e.g. 60
	Slack          *SlackRoute   `yaml:"slack,omitempty"`
	EmailTo        []string      `yaml:"emailTo,omitempty"`
}

type SilencesBootstrap struct {
	// opcional: silences iniciais
}

type Config struct {
	HTTPAuthToken string   `yaml:"httpAuthToken"` // opcional para proteger endpoints de admin
	Storage       Storage  `yaml:"storage"`
	Email         EmailConfig `yaml:"email"`
	Routes        []Route  `yaml:"routes"`
}

func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path); if err != nil { return nil, err }
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil { return nil, err }
	if c.Storage.Path == "" { c.Storage.Path = "data/alert-router.db" }
	return &c, nil
}