package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// ----- Tipos e estado -----

type CheckResult struct {
	UP         bool
	MS         int64
	StatusCode int
	Err        string
}

type MetricsStore struct {
	mu      sync.RWMutex
	results map[string]CheckResult // chave = URL
}

func NewMetricsStore() *MetricsStore {
	return &MetricsStore{results: make(map[string]CheckResult)}
}

func (s *MetricsStore) Set(u string, r CheckResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.results[u] = r
}

func (s *MetricsStore) Snapshot() map[string]CheckResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	// cópia para leitura segura
	cp := make(map[string]CheckResult, len(s.results))
	for k, v := range s.results {
		cp[k] = v
	}
	return cp
}

// ----- Checagem -----

func checkURL(ctx context.Context, target string, timeout time.Duration) CheckResult {
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		elapsed := time.Since(start).Milliseconds()
		return CheckResult{UP: false, MS: elapsed, StatusCode: 0, Err: err.Error()}
	}

	client := http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	elapsed := time.Since(start).Milliseconds()

	if err != nil {
		return CheckResult{UP: false, MS: elapsed, StatusCode: 0, Err: err.Error()}
	}
	defer resp.Body.Close()

	up := resp.StatusCode >= 200 && resp.StatusCode < 400
	return CheckResult{UP: up, MS: elapsed, StatusCode: resp.StatusCode}
}

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

// ----- Loop de coleta (concorrente com limite) -----

func startCollector(urls []string, store *MetricsStore, interval, timeout time.Duration, concurrency int) {
	sem := make(chan struct{}, concurrency)

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		// rodada imediata na partida
		runRound(urls, store, timeout, sem)

		for range ticker.C {
			runRound(urls, store, timeout, sem)
		}
	}()
}

func runRound(urls []string, store *MetricsStore, timeout time.Duration, sem chan struct{}) {
	var wg sync.WaitGroup
	for _, u := range urls {
		u := u
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}          // ocupa vaga
			defer func() { <-sem }()   // libera vaga

			ctx := context.Background()
			res := checkURL(ctx, u, timeout)
			store.Set(u, res)
		}()
	}
	wg.Wait()
}

// ----- Handlers HTTP -----

func metricsHandler(store *MetricsStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Cabeçalhos padrão Prometheus (opcional)
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

		snap := store.Snapshot()

		// Help/Type (boa prática)
		fmt.Fprintln(w, "# HELP healthcheck_up 1 se UP, 0 se DOWN")
		fmt.Fprintln(w, "# TYPE healthcheck_up gauge")
		for u, res := range snap {
			fmt.Fprintf(w, "healthcheck_up{url=%q} %d\n", u, btoi(res.UP))
		}

		fmt.Fprintln(w, "# HELP healthcheck_ms Tempo da checagem em milissegundos")
		fmt.Fprintln(w, "# TYPE healthcheck_ms gauge")
		for u, res := range snap {
			fmt.Fprintf(w, "healthcheck_ms{url=%q} %d\n", u, res.MS)
		}

		fmt.Fprintln(w, "# HELP healthcheck_status_code Último status HTTP observado")
		fmt.Fprintln(w, "# TYPE healthcheck_status_code gauge")
		for u, res := range snap {
			fmt.Fprintf(w, "healthcheck_status_code{url=%q} %d\n", u, res.StatusCode)
		}
	}
}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

// ----- main -----

func main() {
	// Flags
	urlsFlag := flag.String("urls", "https://example.com,https://httpbin.org/status/204,https://httpbin.org/status/500",
		"Lista de URLs separadas por vírgula")
	intervalFlag := flag.String("interval", "30s", "Intervalo de checagem (ex.: 15s, 1m)")
	timeoutFlag := flag.String("timeout", "3s", "Timeout por request (ex.: 3s, 1s)")
	portFlag := flag.String("port", ":8080", "Porta/host do servidor HTTP (ex.: :8080 ou 0.0.0.0:8080)")
	concurrencyFlag := flag.Int("concurrency", 5, "Número máximo de checagens simultâneas por rodada")
	flag.Parse()

	// Parse de durations
	interval, err := time.ParseDuration(*intervalFlag)
	if err != nil || interval <= 0 {
		fmt.Fprintln(os.Stderr, "interval inválido (ex.: 30s, 1m)")
		os.Exit(2)
	}
	timeout, err := time.ParseDuration(*timeoutFlag)
	if err != nil || timeout <= 0 {
		fmt.Fprintln(os.Stderr, "timeout inválido (ex.: 3s, 1s)")
		os.Exit(2)
	}
	if *concurrencyFlag <= 0 {
		fmt.Fprintln(os.Stderr, "concurrency deve ser >= 1")
		os.Exit(2)
	}

	// Parse/validação das URLs
	raw := strings.Split(*urlsFlag, ",")
	var urls []string
	for _, s := range raw {
		u := strings.TrimSpace(s)
		if u == "" {
			continue
		}
		if err := validateURL(u); err != nil {
			fmt.Fprintf(os.Stderr, "URL inválida: %s (%v)\n", u, err)
			os.Exit(2)
		}
		urls = append(urls, u)
	}
	if len(urls) == 0 {
		fmt.Fprintln(os.Stderr, "nenhuma URL válida fornecida; use -urls https://a.com,https://b.com")
		os.Exit(2)
	}

	// Estado e coletor
	store := NewMetricsStore()
	startCollector(urls, store, interval, timeout, *concurrencyFlag)

	// HTTP server
	http.HandleFunc("/metrics", metricsHandler(store))
	http.HandleFunc("/healthz", healthzHandler)

	fmt.Printf("Exporter ouvindo em %s (interval=%s timeout=%s concurrency=%d)\n",
		*portFlag, interval, timeout, *concurrencyFlag)
	fmt.Printf("URLs: %s\n", strings.Join(urls, ", "))

	if err := http.ListenAndServe(*portFlag, nil); err != nil {
		fmt.Fprintln(os.Stderr, "erro ao iniciar servidor:", err)
		os.Exit(1)
	}
}
// ----- Fim do código -----