package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	urlpkg "net/url"
	"os"
	"time"
)

func main() {
	url := flag.String("url", "https://example.com", "URL para verificar")
	retries := flag.Int("retries", 0, "Número de tentativas extras em caso de falha")
	asJSON := flag.Bool("json", false, "Imprime saída em JSON")
	flag.Parse()

	// ---- Validação simples (Passo 4) ----
	if *url == "" {
		fmt.Fprintln(os.Stderr, "uso: go run . -url https://example.com")
		os.Exit(2)
	}
	u, err := urlpkg.ParseRequestURI(*url)
	if err != nil {
		fmt.Fprintln(os.Stderr, "URL inválida. Exemplo: https://example.com")
		os.Exit(2)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		fmt.Fprintln(os.Stderr, "URL deve começar com http:// ou https://")
		os.Exit(2)
	}
	// -------------------------------------

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

	// Estrutura do JSON (usada só quando -json for true)
	type Result struct {
		URL    string `json:"url"`
		Status int    `json:"status"`
		MS     int64  `json:"ms"`
		UP     bool   `json:"up"`
		Error  string `json:"error,omitempty"`
	}

	var status int
	var ms int64
	var up bool

	for attempt := 0; attempt <= *retries; attempt++ {
		var err error
		status, ms, up, err = checkOnce(*url)

		// SUCESSO
		if err == nil && up {
			if *asJSON {
				_ = json.NewEncoder(os.Stdout).Encode(Result{
					URL: *url, Status: status, MS: ms, UP: true,
				})
			} else {
				fmt.Printf("UP %d em %dms (tentativa %d)\n", status, ms, attempt+1)
			}
			os.Exit(0)
		}

		// Ainda tem tentativa? faz backoff e continua
		if attempt < *retries {
			sleep := time.Duration(200*(attempt+1)) * time.Millisecond
			if !*asJSON {
				fmt.Printf("Falhou (tentativa %d). Aguardando %v para tentar novamente...\n", attempt+1, sleep)
			}
			time.Sleep(sleep)
			continue
		}

		// FALHA FINAL
		if *asJSON {
			msg := ""
			if err != nil {
				msg = err.Error()
			}
			_ = json.NewEncoder(os.Stdout).Encode(Result{
				URL: *url, Status: status, MS: ms, UP: false, Error: msg,
			})
		} else {
			if err != nil {
				fmt.Printf("DOWN (erro) em %dms: %v (tentativas: %d)\n", ms, err, attempt+1)
			} else {
				fmt.Printf("DOWN %d em %dms (tentativas: %d)\n", status, ms, attempt+1)
			}
		}
		os.Exit(1)
	}
}
