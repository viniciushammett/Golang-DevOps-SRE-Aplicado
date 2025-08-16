package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Slack struct {
	enabled bool
	webhook string
}

func NewSlack(enabled bool, webhook string) *Slack { return &Slack{enabled, webhook} }

func (s *Slack) Send(text string) error {
	if !s.enabled || s.webhook == "" { return nil }
	body, _ := json.Marshal(map[string]string{"text": text})
	cl := &http.Client{Timeout: 10 * time.Second}
	resp, err := cl.Post(s.webhook, "application/json", bytes.NewReader(body))
	if err != nil { return err }
	_ = resp.Body.Close()
	return nil
}

func Format(rule, kind string, count int, window string, sample string, severity string) string {
	return fmt.Sprintf(":mag: *Anomalia* `%s` (%s) — count=%d window=%s sev=%s\n```%s```", rule, kind, count, window, severity, sample)
}