package main

import (
	"net/http"
	"time"
)

type HealthResult struct {
	URL       string    `json:"url"`
	Status    string    `json:"status"`
	Code      int       `json:"code"`
	ElapsedMS int64     `json:"elapsed_ms"`
	CheckedAt time.Time `json:"checked_at"`
}

// checkHTTP faz o healthcheck HTTP simples com timeout configurável.
func checkHTTP(url string, timeoutSec int) (*HealthResult, error) {
	client := &http.Client{Timeout: time.Duration(timeoutSec) * time.Second}
	start := time.Now()
	resp, err := client.Get(url)
	elapsed := time.Since(start)

	if err != nil {
		return &HealthResult{
			URL:       url,
			Status:    "DOWN",
			Code:      0,
			ElapsedMS: elapsed.Milliseconds(),
			CheckedAt: time.Now(),
		}, err
	}
	defer resp.Body.Close()

	status := "DOWN"
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		status = "UP"
	}

	return &HealthResult{
		URL:       url,
		Status:    status,
		Code:      resp.StatusCode,
		ElapsedMS: elapsed.Milliseconds(),
		CheckedAt: time.Now(),
	}, nil
}
