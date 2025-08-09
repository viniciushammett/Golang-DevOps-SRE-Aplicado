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
	url := flag.String("url", "https://example.com", "URL para verificar")
	retries := flag.Int("retries", 0, "Número de tentativas extras em caso de falha")
	flag.Parse()

	// -------------------------------------
	// Validação da URL
	// -------------------------------------
	// Se a URL não for fornecida, exibe mensagem de uso e sai com erro
	// -------------------------------------
	
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

	// Função que faz UMA tentativa de checagem.
	checkOnce := func(target string) (status int, ms int64, up bool, err error) {
		start := time.Now()
		resp, err := client.Get(target)
		ms = time.Since(start).Milliseconds()

		if err != nil {
			return 0, ms, false, err // erro de rede/timeout = DOWN
		}
		defer resp.Body.Close()

		up = resp.StatusCode >= 200 && resp.StatusCode < 400
		return resp.StatusCode, ms, up, nil
	}

	var status int
	var ms int64
	var up bool

	// Tentamos (retries + 1) vezes no total.
	for attempt := 0; attempt <= *retries; attempt++ {
		var err error
		status, ms, up, err = checkOnce(*url)

		if err == nil && up {
			// sucesso: 2xx/3xx sem erro
			fmt.Printf("UP %d em %dms (tentativa %d)\n", status, ms, attempt+1)
			os.Exit(0)
		}

		// Se falhou e ainda temos tentativas sobrando, aplica backoff e tenta de novo.
		if attempt < *retries {
			// Backoff linear: 200ms * (número da tentativa, começando em 1)
			sleep := time.Duration(200*(attempt+1)) * time.Millisecond
			fmt.Printf("Falhou (tentativa %d). Aguardando %v para tentar novamente...\n", attempt+1, sleep)
			time.Sleep(sleep)
			continue
		}

		// Chegou aqui? Acabaram as tentativas → DOWN.
		if err != nil {
			fmt.Printf("DOWN (erro) em %dms: %v (tentativas: %d)\n", ms, err, attempt+1)
		} else {
			fmt.Printf("DOWN %d em %dms (tentativas: %d)\n", status, ms, attempt+1)
		}
		os.Exit(1)
	}
}
