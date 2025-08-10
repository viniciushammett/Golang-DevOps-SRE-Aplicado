package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"
)

// ---- Modelos de dados ----

// Entrada (config): serviços a monitorar
type Service struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// Saída (API /status): resultado da checagem
type ServiceStatus struct {
	Name       string `json:"name"`
	URL        string `json:"url"`
	Up         bool   `json:"up"`
	StatusCode int    `json:"status_code"`
	LatencyMS  int64  `json:"latency_ms"`
	CheckedAt  string `json:"checked_at"`
	Error      string `json:"error,omitempty"`
}

// ---- Flags ----

var (
	flagAddr        = flag.String("addr", ":8080", "Endereço do servidor HTTP (ex.: :8080)")
	flagTimeout     = flag.Duration("timeout", 3*time.Second, "Timeout por checagem HTTP")
	flagConcurrency = flag.Int("concurrency", 8, "Máximo de checagens simultâneas")
	flagConfig      = flag.String("config", "", "Caminho de um arquivo JSON com serviços (ex.: services.json)")
	flagServices    = flag.String("services", "", "Lista inline de serviços: Nome1=url1,Nome2=url2 (tem prioridade sobre -config)")
)

// ---- Carregamento de configuração ----

// Ordem de prioridade para obter a lista de serviços:
// 1) -services
// 2) -config (arquivo JSON)
// 3) SSD_SERVICES (env) -- mesmo formato do -services
func loadServices() ([]Service, error) {
	if strings.TrimSpace(*flagServices) != "" {
		svcs, err := parseServicesInline(*flagServices)
		if err != nil {
			return nil, fmt.Errorf("parse -services: %w", err)
		}
		return svcs, nil
	}
	if strings.TrimSpace(*flagConfig) != "" {
		return readServicesJSON(*flagConfig)
	}
	if env := strings.TrimSpace(os.Getenv("SSD_SERVICES")); env != "" {
		svcs, err := parseServicesInline(env)
		if err != nil {
			return nil, fmt.Errorf("parse SSD_SERVICES: %w", err)
		}
		return svcs, nil
	}
	return nil, errors.New("nenhuma fonte de serviços encontrada (use -services, -config ou SSD_SERVICES)")
}

// Formato inline: "Nome A=https://a.com,Nome B=https://b.com"
func parseServicesInline(s string) ([]Service, error) {
	var out []Service
	parts := strings.Split(s, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		kv := strings.SplitN(p, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("item inválido: %q (esperado Nome=url)", p)
		}
		name := strings.TrimSpace(kv[0])
		url := strings.TrimSpace(kv[1])
		if name == "" || url == "" {
			return nil, fmt.Errorf("item inválido: %q (Nome e url não podem ser vazios)", p)
		}
		out = append(out, Service{Name: name, URL: url})
	}
	if len(out) == 0 {
		return nil, errors.New("nenhum serviço válido no inline")
	}
	return out, nil
}

func readServicesJSON(path string) ([]Service, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var svcs []Service
	if err := json.Unmarshal(b, &svcs); err != nil {
		return nil, err
	}
	if len(svcs) == 0 {
		return nil, errors.New("arquivo JSON sem serviços")
	}
	return svcs, nil
}

// ---- Checagens reais HTTP ----

type checker struct {
	client      *http.Client
	concurrency int
}

func newChecker(timeout time.Duration, concurrency int) *checker {
	if concurrency <= 0 {
		concurrency = 1
	}
	return &checker{
		client: &http.Client{
			Timeout: timeout,
		},
		concurrency: concurrency,
	}
}

func (c *checker) CheckAll(ctx context.Context, list []Service) []ServiceStatus {
	out := make([]ServiceStatus, len(list))
	sem := make(chan struct{}, c.concurrency)
	var wg sync.WaitGroup

	for i, svc := range list {
		wg.Add(1)
		i, svc := i, svc // capturas
		go func() {
			defer wg.Done()
			sem <- struct{}{}         // ocupa um slot
			defer func() { <-sem }()  // libera

			out[i] = c.checkOne(ctx, svc)
		}()
	}
	wg.Wait()
	return out
}

func (c *checker) checkOne(ctx context.Context, svc Service) ServiceStatus {
	start := time.Now()

	// Contexto por requisição (herda cancelamentos do request principal)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, svc.URL, nil)
	if err != nil {
		return ServiceStatus{
			Name:       svc.Name,
			URL:        svc.URL,
			Up:         false,
			StatusCode: 0,
			LatencyMS:  time.Since(start).Milliseconds(),
			CheckedAt:  time.Now().Format(time.RFC3339),
			Error:      fmt.Sprintf("new request: %v", err),
		}
	}
	req.Header.Set("User-Agent", "service-status-dashboard/1.0")

	resp, err := c.client.Do(req)
	lat := time.Since(start).Milliseconds()

	if err != nil {
		return ServiceStatus{
			Name:       svc.Name,
			URL:        svc.URL,
			Up:         false,
			StatusCode: 0,
			LatencyMS:  lat,
			CheckedAt:  time.Now().Format(time.RFC3339),
			Error:      err.Error(),
		}
	}
	defer resp.Body.Close()

	up := resp.StatusCode >= 200 && resp.StatusCode < 400

	return ServiceStatus{
		Name:       svc.Name,
		URL:        svc.URL,
		Up:         up,
		StatusCode: resp.StatusCode,
		LatencyMS:  lat,
		CheckedAt:  time.Now().Format(time.RFC3339),
	}
}

// ---- HTTP server & handlers ----

func main() {
	flag.Parse()

	// Carrega lista de serviços (flags/env/arquivo)
	services, err := loadServices()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	// Preparar checker com timeout e concorrência
	chk := newChecker(*flagTimeout, *flagConcurrency)

	mux := http.NewServeMux()

	// Frontend estático (index.html)
	mux.Handle("/", http.FileServer(http.Dir("static")))

	// Endpoint /status: executa checagens reais a cada request (simples e efetivo)
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), *flagTimeout+1*time.Second)
		defer cancel()

		results := chk.CheckAll(ctx, services)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(results)
	})

	srv := &http.Server{
		Addr:              *flagAddr,
		Handler:           logMiddleware(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Sobe o servidor
	go func() {
		log.Printf("🌐 Dashboard em http://localhost%s  (serviços: %d, timeout: %s, conc: %d)",
			*flagAddr, len(services), flagTimeout.String(), *flagConcurrency)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("erro no servidor: %v", err)
		}
	}()

	// Encerramento (Ctrl+C)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	log.Println("⏹️  Encerrando servidor...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	log.Println("✅ Encerrado.")
}

func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s (%s)", r.Method, r.URL.Path, time.Since(start))
	})
}
