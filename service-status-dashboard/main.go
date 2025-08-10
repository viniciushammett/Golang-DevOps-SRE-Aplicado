package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"
)

// ServiceStatus representa um serviço monitorado (mock por enquanto).
type ServiceStatus struct {
	Name       string  `json:"name"`        // nome amigável
	URL        string  `json:"url"`         // endpoint monitorado
	Up         bool    `json:"up"`          // está no ar?
	StatusCode int     `json:"status_code"` // último status HTTP
	LatencyMS  int64   `json:"latency_ms"`  // latência da última checagem
	CheckedAt  string  `json:"checked_at"`  // timestamp ISO8601
	Note       string  `json:"note"`        // extra (ex.: “mockado no passo 1”)
	Score      float64 `json:"score"`       // campo livre p/ heurísticas futuras
}

func main() {
	mux := http.NewServeMux()

	// 1) Arquivos estáticos (HTML, CSS, JS) servidos da pasta ./static
	//    Ex.: GET / -> index.html
	mux.Handle("/", http.FileServer(http.Dir("static")))

	// 2) Endpoint /status — por enquanto, dados MOCK (fixos) pra testar o frontend.
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		now := time.Now().Format(time.RFC3339)
		mock := []ServiceStatus{
			{
				Name:       "Example",
				URL:        "https://example.com",
				Up:         true,
				StatusCode: 200,
				LatencyMS:  123,
				CheckedAt:  now,
				Note:       "mockado no passo 1",
				Score:      0.99,
			},
			{
				Name:       "HttpBin (204)",
				URL:        "https://httpbin.org/status/204",
				Up:         true,
				StatusCode: 204,
				LatencyMS:  87,
				CheckedAt:  now,
				Note:       "mockado no passo 1",
				Score:      0.95,
			},
			{
				Name:       "HttpBin (500)",
				URL:        "https://httpbin.org/status/500",
				Up:         false,
				StatusCode: 500,
				LatencyMS:  65,
				CheckedAt:  now,
				Note:       "mockado no passo 1",
				Score:      0.10,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mock)
	})

	// 3) Servidor com timeouts (boas práticas)
	srv := &http.Server{
		Addr:              ":8080",
		Handler:           logMiddleware(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	// 4) Inicialização assíncrona + desligamento gracioso (Ctrl+C)
	go func() {
		log.Println("🌐 Service Status Dashboard rodando em http://localhost:8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("erro no servidor: %v", err)
		}
	}()

	// Espera Ctrl+C
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	log.Println("⏹️  Encerrando servidor...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	log.Println("✅ Encerrado.")
}

// logMiddleware: loga método, caminho e duração de cada request (útil p/ dev).
func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s (%s)", r.Method, r.URL.Path, time.Since(start))
	})
}
