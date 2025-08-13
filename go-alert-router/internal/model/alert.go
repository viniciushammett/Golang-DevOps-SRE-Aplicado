package model

import (
	"crypto/sha1"
	"encoding/hex"
	"sort"
	"strings"
	"time"
)

type Alert struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	StartsAt    time.Time         `json:"startsAt"`
	EndsAt      time.Time         `json:"endsAt"`
	GeneratorURL string           `json:"generatorURL"`
	Fingerprint string            `json:"fingerprint"` // calculado se vazio
}

func (a *Alert) EnsureFingerprint() {
	if a.Fingerprint != "" { return }
	parts := make([]string, 0, len(a.Labels))
	for k, v := range a.Labels { parts = append(parts, k+"="+v) }
	sort.Strings(parts)
	h := sha1.Sum([]byte(strings.Join(parts, ",")))
	a.Fingerprint = hex.EncodeToString(h[:])
}

type WebhookPayload struct {
	Receiver string  `json:"receiver"`
	Status   string  `json:"status"`
	Alerts   []Alert `json:"alerts"`
	ExternalURL string `json:"externalURL"`
}