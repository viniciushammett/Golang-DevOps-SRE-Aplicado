package notify

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"
)

type payload struct {
	Text string `json:"text"`
}

func SendSlack(webhook string, text string) error {
	if webhook == "" { return nil }
	b, _ := json.Marshal(payload{Text: text})
	client := &http.Client{ Timeout: 10 * time.Second }
	resp, err := client.Post(webhook, "application/json", bytes.NewReader(b))
	if err != nil { return err }
	_ = resp.Body.Close()
	return nil
}