package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// ---- Tipos ----

type Release struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	HTMLURL     string    `json:"html_url"`
	PublishedAt time.Time `json:"published_at"`
}

type Out struct {
	Repo        string    `json:"repo"` // owner/repo
	Tag         string    `json:"tag"`
	Name        string    `json:"name"`
	URL         string    `json:"url"`
	PublishedAt time.Time `json:"published_at"`
	Error       string    `json:"error,omitempty"`
}

// ---- HTTP / GitHub ----

func fetchLatest(owner, repo string, timeout time.Duration) (Release, int, error) {
	var zero Release

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx,
		http.MethodGet,
		fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo),
		nil)
	if err != nil {
		return zero, 0, err
	}
	// GitHub exige um User-Agent válido
	req.Header.Set("User-Agent", "go-cli-learning/1.0")

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return zero, 0, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// ok
	case http.StatusNotFound:
		return zero, resp.StatusCode, fmt.Errorf("not found (ou sem releases)")
	case http.StatusForbidden:
		return zero, resp.StatusCode, fmt.Errorf("forbidden/rate limit (403) — use token")
	default:
		return zero, resp.StatusCode, fmt.Errorf("status inesperado: %s", resp.Status)
	}

	var rel Release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return zero, resp.StatusCode, fmt.Errorf("decode: %w", err)
	}
	return rel, resp.StatusCode, nil
}

// ---- Aux ----

func splitRepo(s string) (owner, repo string, ok bool) {
	i := strings.IndexByte(s, '/')
	if i <= 0 || i == len(s)-1 {
		return "", "", false
	}
	return s[:i], s[i+1:], true
}

func dash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// ---- main ----

func main() {
	// Mantemos -owner/-repo por compatibilidade, mas preferimos -repos
	owner := flag.String("owner", "", "Dono do repositório (ex.: grafana)")
	repo := flag.String("repo", "", "Nome do repositório (ex.: grafana)")
	repos := flag.String("repos", "", "Lista de repositórios owner/repo separados por vírgula")
	timeout := flag.Duration("timeout", 5*time.Second, "Timeout por requisição (ex.: 5s)")
	asJSON := flag.Bool("json", false, "Saída em JSON (uma linha por repo - NDJSON)")
	concurrency := flag.Int("concurrency", 5, "Número máximo de consultas em paralelo")
	flag.Parse()

	// Monta a lista de repositórios
	var list []string
	if *repos != "" {
		for _, r := range strings.Split(*repos, ",") {
			r = strings.TrimSpace(r)
			if r != "" {
				list = append(list, r)
			}
		}
	} else if *owner != "" && *repo != "" {
		list = []string{*owner + "/" + *repo}
	}

	if len(list) == 0 {
		fmt.Fprintln(os.Stderr, "uso: go run . -repos owner1/repo1,owner2/repo2 [-timeout 5s] [-json] [-concurrency 5]")
		fmt.Fprintln(os.Stderr, "     ou: go run . -owner grafana -repo grafana")
		os.Exit(2)
	}
	if *concurrency <= 0 {
		fmt.Fprintln(os.Stderr, "-concurrency deve ser >= 1")
		os.Exit(2)
	}

	type result struct {
		out   Out
		ok    bool
		error error
	}

	sem := make(chan struct{}, *concurrency) // semáforo de paralelismo
	var wg sync.WaitGroup
	results := make(chan result)

	for _, full := range list {
		full := full // captura
		owner, repo, ok := splitRepo(full)
		if !ok {
			// erro de formato já aqui
			go func() {
				results <- result{out: Out{Repo: full, Error: "formato inválido (use owner/repo)"}, ok: false, error: fmt.Errorf("formato inválido")}
			}()
			continue
		}

		wg.Add(1)
		go func(owner, repo, full string) {
			defer wg.Done()
			sem <- struct{}{}         // ocupa vaga
			defer func() { <-sem }()  // libera

			rel, _, err := fetchLatest(owner, repo, *timeout)
			res := result{
				out: Out{
					Repo:        full,
					Tag:         rel.TagName,
					Name:        rel.Name,
					URL:         rel.HTMLURL,
					PublishedAt: rel.PublishedAt,
				},
				ok: err == nil,
			}
			if err != nil {
				res.out.Error = err.Error()
				res.error = err
			}
			results <- res
		}(owner, repo, full)
	}

	// Fechar canal quando tudo terminar
	go func() {
		wg.Wait()
		close(results)
	}()

	exitCode := 0

	// Consome conforme as goroutines terminam (ordem pode variar)
	for r := range results {
		if *asJSON {
			_ = json.NewEncoder(os.Stdout).Encode(r.out) // NDJSON
		} else {
			if r.ok {
				fmt.Printf("[%s]\n", r.out.Repo)
				fmt.Printf("  Tag: %s\n", dash(r.out.Tag))
				fmt.Printf("  Nome: %s\n", dash(r.out.Name))
				fmt.Printf("  Publicado: %s\n", r.out.PublishedAt.Format(time.RFC3339))
				fmt.Printf("  URL: %s\n", r.out.URL)
			} else {
				fmt.Printf("[%s] ERRO: %s\n", r.out.Repo, r.out.Error)
			}
		}
		if !r.ok {
			exitCode = 1
		}
	}

	os.Exit(exitCode)
}
// ---- Fim do código ----