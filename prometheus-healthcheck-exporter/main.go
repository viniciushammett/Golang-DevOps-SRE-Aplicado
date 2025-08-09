package main

import (
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// Estrutura para armazenar resultados
type CheckResult struct {
	UP bool
	MS int64
}

// Mapa para guardar resultados por URL
var (
	results   = make(map[string]CheckResult)
	resultsMu sync.RWMutex
)

// Função para checar uma URL
func checkURL(target string) CheckResult {
	start := time.Now()

	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(target)
	elapsed := time.Since(start).Milliseconds()

	if err != nil {
		return CheckResult{UP: false, MS: elapsed}
	}
	defer resp.Body.Close()

	up := resp.StatusCode >= 200 && resp.StatusCode < 400
	return CheckResult{UP: up, MS: elapsed}
}

// Função para validar URL
func validateURL(raw string) error {
	parsed, err := url.ParseRequestURI(raw)
	if err != nil {
		return err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("URL deve começar com http:// ou https://")
	}
	return nil
}

// Loop para atualizar métricas periodicamente
func startChecker(urls []string, interval time.Duration) {
	go func() {
		for {
			for _, u := range urls {
				res := checkURL(u)
				resultsMu.Lock()
				results[u] = res
				resultsMu.Unlock()
			}
			time.Sleep(interval)
		}
	}()
}

// Handler /metrics para Prometheus
func metricsHandler(w http.ResponseWriter, r *http.Request) {
	resultsMu.RLock()
	defer resultsMu.RUnlock()

	for u, res := range results {
		fmt.Fprintf(w, "healthcheck_up{url=%q} %d\n", u, boolToInt(res.UP))
		fmt.Fprintf(w, "healthcheck_ms{url=%q} %d\n", u, res.MS)
	}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func main() {
	// URLs para monitorar (fixo por enquanto)
	urls := []string{
		"https://example.com",
		"https://httpbin.org/status/204",
		"https://httpbin.org/status/500",
	}

	// Validação de URLs
	for _, u := range urls {
		if err := validateURL(u); err != nil {
			fmt.Printf("URL inválida: %s (%v)\n", u, err)
			return
		}
	}

	// Inicia checador com intervalo de 30 segundos
	startChecker(urls, 30*time.Second)

	// Endpoint de métricas
	http.HandleFunc("/metrics", metricsHandler)

	fmt.Println("Exporter rodando em :8080/metrics")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println("Erro ao iniciar servidor:", err)
	}
}
