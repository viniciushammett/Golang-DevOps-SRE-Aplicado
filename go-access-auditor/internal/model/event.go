package model

import "time"

type Event struct {
	ID       string            `json:"id"`
	When     time.Time         `json:"when"`
	User     string            `json:"user"`
	Host     string            `json:"host"`
	Source   string            `json:"source"`
	Command  string            `json:"command"`
	Meta     map[string]string `json:"meta"`
	// Derivados:
	FlagSensitive bool   `json:"flagSensitive"`
	RuleMatched   string `json:"ruleMatched,omitempty"`
}