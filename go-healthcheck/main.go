package main

import (
	"flag"
	"fmt"
	"net/http"
	urlpkg "net/url"
	"os"
	"time"
)

func main() {
	url := flag.String("url", "http://localhost:8080/health", "URL to check health status")
	flag.Parse()

	// ---- Validação simples de entrada ----
	if *url == "" {
		fmt.Fprintln(os.Stderr, "uso: go run . -url http://localhost:8080/health")
		os.Exit(2)
	}

	u, err := urlpkg.ParseRequestURI(*url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "URL inválida: Exemplo: http://localhost:8080/health")
		os.Exit(2)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		fmt.Fprintf(os.Stderr, "URL inválida: deve começar com http:// ou https://")
		os.Exit(2)
	}

	// -------------------------------------------------

	client := &http.Client{Timeout: 3 * time.Second}

	start := time.Now()
	resp, err := client.Get(*url)
	ms := time.Since(start).Milliseconds()
	
	if err != nil {
		fmt.Printf("DOWN (erro) em %dms: %v\n", ms, err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		fmt.Printf("UP %d em %dms\n", resp.StatusCode, ms)
		os.Exit(0)
	} else {
		fmt.Printf("DOWN %d em %dms\n", resp.StatusCode, ms)
		os.Exit(1)
	}
}