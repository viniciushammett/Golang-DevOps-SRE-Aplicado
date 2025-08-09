package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"
)

// Release mapeia só os campos que nos interessam da API do GitHub.
type Release struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	HTMLURL     string    `json:"html_url"`
	PublishedAt time.Time `json:"published_at"`
}

// Resposta JSON do nosso CLI (quando -json for usado).
type Out struct {
	Owner       string    `json:"owner"`
	Repo        string    `json:"repo"`
	Tag         string    `json:"tag"`
	Name        string    `json:"name"`
	URL         string    `json:"url"`
	PublishedAt time.Time `json:"published_at"`
	Error       string    `json:"error,omitempty"`
}

func main() {
	// ----- Flags -----
	owner := flag.String("owner", "", "Dono do repositório (ex.: grafana)")
	repo := flag.String("repo", "", "Nome do repositório (ex.: grafana)")
	timeout := flag.Duration("timeout", 5*time.Second, "Timeout da requisição (ex.: 5s, 2s)")
	asJSON := flag.Bool("json", false, "Imprime saída em JSON")
	flag.Parse()

	// ----- Validação de uso -----
	if *owner == "" || *repo == "" {
		fmt.Fprintln(os.Stderr, "uso: go run . -owner <owner> -repo <repo> [-timeout 5s] [-json]")
		os.Exit(2) // uso inválido
	}

	// ----- Faz a chamada com timeout e User-Agent -----
	rel, status, err := fetchLatest(*owner, *repo, *timeout)

	// ----- Saída JSON opcional -----
	if *asJSON {
		out := Out{
			Owner:       *owner,
			Repo:        *repo,
			Tag:         rel.TagName,
			Name:        rel.Name,
			URL:         rel.HTMLURL,
			PublishedAt: rel.PublishedAt,
		}
		if err != nil {
			out.Error = err.Error()
		}
		_ = json.NewEncoder(os.Stdout).Encode(out)
		// exit codes coerentes pra automação:
		if err != nil {
			if status == 0 {
				os.Exit(1)
			}
			// 404 e 403 também são 1 (falha)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// ----- Saída humana -----
	if err != nil {
		fmt.Fprintf(os.Stderr, "erro: %v\n", err)
		if status == 403 {
			fmt.Fprintln(os.Stderr, "dica: use um token do GitHub para aumentar o rate limit (flag futura -token ou GITHUB_TOKEN).")
		}
		os.Exit(1)
	}

	fmt.Printf("Repo: %s/%s\n", *owner, *repo)
	fmt.Printf("Tag: %s\n", dash(rel.TagName))
	fmt.Printf("Nome: %s\n", dash(rel.Name))
	fmt.Printf("Publicado em: %s\n", rel.PublishedAt.Format(time.RFC3339))
	fmt.Printf("URL: %s\n", rel.HTMLURL)
	os.Exit(0)
}

// fetchLatest faz GET em /releases/latest com timeout e trata status comuns.
func fetchLatest(owner, repo string, timeout time.Duration) (Release, int, error) {
	var zero Release

	// Contexto com timeout para não travar o CLI se a rede/host estiver lento.
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx,
		http.MethodGet,
		fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo),
		nil)
	if err != nil {
		return zero, 0, err
	}

	// GitHub exige User-Agent válido; sem isso pode dar 403.
	req.Header.Set("User-Agent", "go-cli-learning/1.0")

	// Cliente com timeout adicional (fallback).
	client := &http.Client{Timeout: timeout}

	resp, err := client.Do(req)
	if err != nil {
		return zero, 0, err
	}
	defer resp.Body.Close()

	// Trata status comuns antes de tentar decodificar JSON.
	switch resp.StatusCode {
	case http.StatusOK:
		// segue o fluxo
	case http.StatusNotFound:
		return zero, resp.StatusCode, fmt.Errorf("repositório sem release ou não encontrado (%s/%s) [404]", owner, repo)
	case http.StatusForbidden:
		// geralmente rate limit; podemos inspecionar headers se quiser evoluir
		return zero, resp.StatusCode, fmt.Errorf("acesso negado/rate limit [403]")
	default:
		return zero, resp.StatusCode, fmt.Errorf("status inesperado da API: %s", resp.Status)
	}

	var rel Release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return zero, resp.StatusCode, fmt.Errorf("falha ao decodificar JSON: %w", err)
	}
	return rel, resp.StatusCode, nil
}

func dash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// Fim do código.
