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

//
// ===================== Modelos de dados =====================
//

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

// Envelope servido pelo /status (cacheado)
type StatusPayload struct {
	Services    []ServiceStatus `json:"services"`
	LastUpdated string          `json:"last_updated"` // ISO8601
	IntervalSec int             `json:"interval_sec"`
}

//
// ===================== Flags =====================
//

var (
	flagAddr        = flag.String("addr", ":8080", "Endereço do servidor HTTP (ex.: :8080)")
	flagTimeout     = flag.Duration("timeout", 3*time.Second, "Timeout por checagem HTTP")
	flagConcurrency = flag.Int("concurrency", 8, "Máximo de checagens simultâneas")
	flagInterval    = flag.Duration("interval", 10*time.Second, "Intervalo entre rodadas de checagem")
	flagConfig      = flag.String("config", "", "Arquivo JSON com serviços (ex.: services.json)")
	flagServices    = flag.String("services", "", "Lista inline: Nome1=url1,Nome2=url2 (maior prioridade)")
)

//
// ===================== Config loader =====================
//

// Prioridade:
// 1) -services
// 2) -config (arquivo JSON)
// 3) SSD_SERVICES (env) — mesmo formato do -services
func loadServices() ([]Service, error) {
	if s := strings.TrimSpace(*flagServices); s != "" {
		return parseServicesInline(s)
	}
	if c := strings.TrimSpace(*flagConfig); c != "" {
		return readServicesJSON(c)
	}
	if env := strings.TrimSpace(os.Getenv("SSD_SERVICES")); env != "" {
		return parseServicesInline(env)
	}
	return nil, errors.New("nenhuma fonte de serviços encontrada (use -services, -config ou SSD_SERVICES)")
}

func parseServicesInline(s string) ([]Service, error) {
	var out []Service
	for _, item := range strings.Split(s, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		kv := strings.SplitN(item, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("item inválido: %q (esperado Nome=url)", item)
		}
		name := strings.TrimSpace(kv[0])
		url := strings.TrimSpace(kv[1])
		if name == "" || url == "" {
			return nil, fmt.Errorf("item inválido: %q (Nome e url não podem ser vazios)", item)
		}
		out = append(out, Service{Name: name, URL: url})
	}
	if len(out) == 0 {
		return nil, errors.New("nenhum serviço válido")
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

//
// ===================== Checker =====================
//

type checker struct {
	client      *http.Client
	concurrency int
}

func newChecker(timeout time.Duration, concurrency int) *checker {
	if concurrency <= 0 {
		concurrency = 1
	}
	return &checker{
		client: &http.Client{Timeout: timeout},
		concurrency: concurrency,
	}
}

func (c *checker) CheckAll(ctx context.Context, list []Service) []ServiceStatus {
	out := make([]ServiceStatus, len(list))
	sem := make(chan struct{}, c.concurrency)
	var wg sync.WaitGroup

	for i, svc := range list {
		wg.Add(1)
		i, svc := i, svc
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			out[i] = c.checkOne(ctx, svc)
		}()
	}
	wg.Wait()
	return out
}

func (c *checker) checkOne(ctx context.Context, svc Service) ServiceStatus {
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, svc.URL, nil)
	if err != nil {
		return ServiceStatus{
			Name: svc.Name, URL: svc.URL, Up: false, StatusCode: 0,
			LatencyMS: time.Since(start).Milliseconds(),
			CheckedAt: time.Now().Format(time.RFC3339),
			Error:     fmt.Sprintf("new request: %v", err),
		}
	}
	req.Header.Set("User-Agent", "service-status-dashboard/1.0")

	resp, err := c.client.Do(req)
	lat := time.Since(start).Milliseconds()
	if err != nil {
		return ServiceStatus{
			Name: svc.Name, URL: svc.URL, Up: false, StatusCode: 0,
			LatencyMS: lat,
			CheckedAt: time.Now().Format(time.RFC3339),
			Error:     err.Error(),
		}
	}
	defer resp.Body.Close()

	up := resp.StatusCode >= 200 && resp.StatusCode < 400
	return ServiceStatus{
		Name: svc.Name, URL: svc.URL, Up: up, StatusCode: resp.StatusCode,
		LatencyMS: lat, CheckedAt: time.Now().Format(time.RFC3339),
	}
}

//
// ===================== Cache + Scheduler =====================
//

type statusStore struct {
	mu          sync.RWMutex
	payload     StatusPayload
	hasData     bool
}

func (s *statusStore) Set(p StatusPayload) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.payload = p
	s.hasData = true
}

func (s *statusStore) Get() (StatusPayload, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.payload, s.hasData
}

// runScheduler executa checagens periódicas e grava no cache.
// - roda 1x imediatamente (para preencher o cache na inicialização)
// - depois repete a cada interval
func runScheduler(ctx context.Context, c *checker, services []Service, interval time.Duration, store *statusStore) {
	runOnce := func() {
		// contexto com timeout total = timeout do checker + uma folga
		timeout := c.client.Timeout
		if timeout == 0 {
			timeout = 3 * time.Second
		}
		ctxCheck, cancel := context.WithTimeout(ctx, timeout+1*time.Second)
		defer cancel()

		res := c.CheckAll(ctxCheck, services)
		store.Set(StatusPayload{
			Services:    res,
			LastUpdated: time.Now().Format(time.RFC3339),
			IntervalSec: int(interval.Seconds()),
		})
	}

	// primeira rodada já
	runOnce()

	t := time.NewTicker(interval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			runOnce()
		}
	}
}

//
// ===================== HTTP server =====================
//

func main() {
	flag.Parse()

	services, err := loadServices()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	chk := newChecker(*flagTimeout, *flagConcurrency)

	// cache em memória
	var store statusStore

	// contexto para encerramento gracioso
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// inicia o scheduler em background
	go runScheduler(ctx, chk, services, *flagInterval, &store)

	// mux HTTP
	mux := http.NewServeMux()

	// frontend
	mux.Handle("/", http.FileServer(http.Dir("static")))

	// status: serve SOMENTE o cache (rápido)
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		p, ok := store.Get()
		if !ok {
			// ainda não carregou a primeira rodada
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": "status ainda não disponível",
			})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(p)
	})

	// healthz simples
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		// OK se já temos pelo menos uma rodada no cache
		if _, ok := store.Get(); ok {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("warming up"))
	})

	srv := &http.Server{
		Addr:              *flagAddr,
		Handler:           logMiddleware(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	// sobe servidor
	go func() {
		log.Printf("🌐 Dashboard em http://localhost%s | serviços=%d | interval=%s | timeout=%s | conc=%d",
			*flagAddr, len(services), flagInterval.String(), flagTimeout.String(), *flagConcurrency)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("erro no servidor: %v", err)
		}
	}()

	// espera Ctrl+C
	<-ctx.Done()

	log.Println("⏹️  Encerrando servidor...")
	shCtx, shCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shCancel()
	_ = srv.Shutdown(shCtx)
	log.Println("✅ Encerrado.")
}

func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s (%s)", r.Method, r.URL.Path, time.Since(start))
	})
}