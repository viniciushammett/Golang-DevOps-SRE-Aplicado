package rules

import (
	"os"
	"regexp"
	"time"
	"gopkg.in/yaml.v3"
)

type RuleYAML struct {
	Name      string `yaml:"name"`
	Pattern   string `yaml:"pattern"`     // regex
	Window    string `yaml:"window"`      // ex: "1m"
	MinCount  int    `yaml:"minCount"`    // >= para disparar
	MaxRate   int    `yaml:"maxRate"`     // opções alternativas
	Severity  string `yaml:"severity"`    // info|warn|critical
}

type Rule struct {
	Name     string
	RE       *regexp.Regexp
	Window   time.Duration
	MinCount int
	MaxRate  int
	Severity string
}

type Set struct {
	Items []Rule
}

func LoadFromFile(path string) (*Set, error) {
	if path == "" { path = "configs/rules.example.yaml" }
	b, err := os.ReadFile(path); if err != nil { return nil, err }
	var raw []RuleYAML
	if err := yaml.Unmarshal(b, &raw); err != nil { return nil, err }
	out := &Set{}
	for _, r := range raw {
		d, _ := time.ParseDuration(r.Window)
		if d == 0 { d = time.Minute }
		re, err := regexp.Compile(r.Pattern); if err != nil { continue }
		out.Items = append(out.Items, Rule{
			Name: r.Name, RE: re, Window: d, MinCount: r.MinCount, MaxRate: r.MaxRate,
			Severity: r.Severity,
		})
	}
	return out, nil
}