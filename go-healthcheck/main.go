package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	urlpkg "net/url"
	"os"
	"strings"
	"sync"
	"time"
)

func main() {
	// Flags
	urlsFlag := flag.String("urls", "https://example.com", "Lista de URLs separadas por vírgula")
	retries := flag.Int("retries", 0, "Número de tentativas extras em caso de falha")
	asJSON := flag.Bool("json", false, "Imprime saída em JSON (uma linha por URL)")
	concurrency := flag.Int("concurrency", 5, "Número máximo de checagens em paralelo")
	flag.Parse()

	// Parse simples da lista de URLs
	raw := strings.Split(*urlsFlag, ",")
	var urls []string
	for _, s := range raw {
		u := strings.TrimSpace(s)
		if u != "" {
			urls = append(urls, u)
		}
	}
	if len(urls) == 0 {
		fmt.Fprintln(os.Stderr, "uso: go run . -urls https://a.com,https://b.com")
		os.Exit(2)
	}

	// Validação de cada URL
	for _, u := range urls {
		pu, err := urlpkg.ParseRequestURI(u)
		if err != nil || (pu.Scheme != "http" && pu.Scheme != "https") {
			fmt.Fprintln(os.Stderr, "URL inválida:", u, "(use http(s)://)")
			os.Exit(2)
		}
	}

	// HTTP client com timeout fixo (pode virar flag depois)
	client := http.Client{Timeout: 3 * time.Second}

	// Uma tentativa de checagem
	checkOnce := func(target string) (status int, ms int64, up bool, err error) {
		start := time.Now()
		resp, err := client.Get(target)
		ms = time.Since(start).Milliseconds()
		if err != nil {
			return 0, ms, false, err
		}
		defer resp.Body.Close()

		up = resp.StatusCode >= 200 && resp.StatusCode < 400
		return resp.StatusCode, ms, up, nil
	}

	// Estrutura de resultado (para -json)
	type Result struct {
		URL    string `json:"url"`
		Status int    `json:"status"`
		MS     int64  `json:"ms"`
		UP     bool   `json:"up"`
		Error  string `json:"error,omitempty"`
	}

	results := make(chan Result)              // canal para coletar resultados
	sem := make(chan struct{}, *concurrency)  // semáforo de concorrência
	var wg sync.WaitGroup

	// Dispara uma goroutine por URL
	for _, u := range urls {
		u := u // captura
		wg.Add(1)

		go func() {
			defer wg.Done()

			// ocupar vaga do semáforo
			sem <- struct{}{}
			defer func() { <-sem }() // liberar vaga quando terminar

			var status int
			var ms int64
			var up bool
			var err error

			for attempt := 0; attempt <= *retries; attempt++ {
				status, ms, up, err = checkOnce(u)
				if err == nil && up {
					break
				}
				if attempt < *retries {
					time.Sleep(time.Duration(200*(attempt+1)) * time.Millisecond)
				}
			}

			r := Result{URL: u, Status: status, MS: ms, UP: up}
			if err != nil && !up {
				r.Error = err.Error()
			}
			results <- r
		}()
	}

	// Fecha o canal quando todas terminarem
	go func() {
		wg.Wait()
		close(results)
	}()

	exitCode := 0
	// Consome resultados conforme ficam prontos (ordem pode variar)
	for r := range results {
		if *asJSON {
			_ = json.NewEncoder(os.Stdout).Encode(r)
		} else {
			if r.UP {
				fmt.Printf("[%s] UP %d em %dms\n", r.URL, r.Status, r.MS)
			} else {
				if r.Error != "" {
					fmt.Printf("[%s] DOWN (erro) em %dms: %s\n", r.URL, r.MS, r.Error)
				} else {
					fmt.Printf("[%s] DOWN %d em %dms\n", r.URL, r.Status, r.MS)
				}
			}
		}
		if !r.UP {
			exitCode = 1
		}
	}

	os.Exit(exitCode)
}
