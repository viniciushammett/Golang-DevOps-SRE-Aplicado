package notify

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/viniciushammett/go-alert-router/internal/logger"
)

type Slack struct {
	log *logger.Logger
}

func NewSlack(log *logger.Logger) *Slack { return &Slack{log: log} }

type slackMsg struct {
	Text string `json:"text"`
}

func (s *Slack) Send(webhook, text string) error {
	b, _ := json.Marshal(slackMsg{Text: text})
	client := &http.Client{ Timeout: 10 * time.Second }
	resp, err := client.Post(webhook, "application/json", bytes.NewReader(b))
	if err != nil { return err }
	_ = resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return errHTTP(resp.Status)
	}
	return nil
}

type httpErr string
func (e httpErr) Error() string { return string(e) }
func errHTTP(st string) error { return httpErr("slack http status " + st) }