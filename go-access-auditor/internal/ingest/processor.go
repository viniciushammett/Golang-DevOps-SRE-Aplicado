package ingest

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/viniciushammett/go-access-auditor/internal/logger"
	"github.com/viniciushammett/go-access-auditor/internal/metrics"
	"github.com/viniciushammett/go-access-auditor/internal/notify"
	"github.com/viniciushammett/go-access-auditor/internal/rules"
	"github.com/viniciushammett/go-access-auditor/internal/store"
)

type Processor struct {
	log   *logger.Logger
	db    *store.Store
	rules *rules.Set
	slackEnabled bool
	slackWebhook string
}

func NewProcessor(log *logger.Logger, db *store.Store, rs *rules.Set, slackEnabled bool, slackWebhook string) *Processor {
	return &Processor{log: log, db: db, rules: rs, slackEnabled: slackEnabled, slackWebhook: slackWebhook}
}

type Incoming struct {
	When    time.Time         `json:"when"`
	User    string            `json:"user"`
	Host    string            `json:"host"`
	Source  string            `json:"source"`
	Command string            `json:"command"`
	Meta    map[string]string `json:"meta"`
}

func (p *Processor) Handle(ev Incoming) (store.Event, error) {
	if ev.When.IsZero() { ev.When = time.Now() }
	id := randID()
	match, rule := p.rules.Match(ev.Command)

	rec := store.Event{
		ID: id, When: ev.When, User: ev.User, Host: ev.Host, Source: ev.Source,
		Command: ev.Command, Meta: ev.Meta, FlagSensitive: match, RuleMatched: rule,
	}
	if err := p.db.Put(rec); err != nil { return rec, err }

	metrics.EventsIngested.WithLabelValues(ev.Source).Inc()
	if match {
		metrics.SensitiveMatches.WithLabelValues(rule).Inc()
		if p.slackEnabled {
			_ = notify.SendSlack(p.slackWebhook,
				fmt.Sprintf(":rotating_light: *Sensitive command* by *%s@%s* via *%s*\n```%s```\n_rule: %s_",
					ev.User, ev.Host, ev.Source, ev.Command, rule))
		}
	}
	return rec, nil
}

func randID() string {
	b := make([]byte, 6); _, _ = rand.Read(b)
	return hex.EncodeToString(b)
}