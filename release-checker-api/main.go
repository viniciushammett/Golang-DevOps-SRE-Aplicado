package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net"
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
	DurationMS  int64     `json:"duration_ms"`
	Error       string    `json:"error,omitempty"`
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

// Verifica se deve retry (apenas erros transitórios)
func isRetryable(status int, err error) bool {
	// 5xx -> geralmente transitório
	if status >= 500 && status <= 599 {
		return true
	}
	// Erros de rede/timeout
	if err != nil {
		// context deadline/cancel
		if ne, ok := err.(net.Error); ok {
			if ne.Timeout() || ne.Temporary() {
				return true
			}
		}
		// generic timeout string (quando vem de context)
		if strings.Contains(strings.ToLower(err.Error()), "timeout") {
			return true
		}
	}
	return false
}

// ---- HTTP / GitHub ----

// fetchLatest executa UMA tentativa, com timeout e (opcional) token.
// Retorna release, status HTTP, duração em ms e erro.
func fetchLatest(owner, repo string, timeout time.Duration, token string) (Release, int, int64, error) {
	var zero Release

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return zero, 0, 0, err
	}
	req.Header.Set("User-Agent", "go-cli-learning/1.0")
	if token != "" {
		// GitHub REST v3 aceita "token <TOKEN>"
		req.Header.Set("Authorization", "token "+token)
	}

	client := &http.Client{Timeout: timeout}

	start := time.Now()
	resp, err := client.Do(req)
	elapsed := time.Since(start).Milliseconds()
	if err != nil {
		return zero, 0, elapsed, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// segue
	case http.StatusNotFound:
		return zero, resp.StatusCode, elapsed, fmt.Errorf("not found (ou sem releases)")
	case http.StatusForbidden:
		// Geralmente rate limit sem token suficiente
		return zero, resp.StatusCode, elapsed, fmt.Errorf("forbidden/rate limit (403) — forneça -token ou GITHUB_TOKEN")
	default:
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return zero, resp.StatusCode, elapsed, fmt.Errorf("status inesperado: %s", resp.Status)
		}
	}

	var rel Release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return zero, resp.StatusCode, elapsed, fmt.Errorf("decode: %w", err)
	}
	return rel, resp.StatusCode, elapsed, nil
}

// fetchWithRetries aplica backoff exponencial: base * 2^tentativa (1,2,4,...)
func fetchWithRetries(owner, repo string, timeout time.Duration, token string, retries int, backoffBase time.Duration) (Release, int, int64, error) {
	var lastRel Release
	var lastStatus int
	var lastDur int64
	var lastErr error

	for attempt := 0; attempt <= retries; attempt++ {
		rel, status, dur, err := fetchLatest(owner, repo, timeout, token)
		lastRel, lastStatus, lastDur, lastErr = rel, status, dur, err

		if err == nil {
			return rel, status, dur, nil
		}
		if !isRetryable(status, err) || attempt == retries {
			break
		}
		// Backoff exponencial: base * 2^attempt (attempt inicia em 0)
		sleep := backoffBase << attempt
		time.Sleep(sleep)
	}
	return lastRel, lastStatus, lastDur, lastErr
}

// ---- main ----

func main() {
	repos := flag.String("repos", "", "Lista de repositórios owner/repo separados por vírgula")
	owner := flag.String("owner", "", "Dono do repositório (ex.: grafana)")
	repo := flag.String("repo", "", "Nome do repositório (ex.: grafana)")
	timeout := flag.Duration("timeout", 5*time.Second, "Timeout por requisição (ex.: 5s)")
	concurrency := flag.Int("concurrency", 5, "Número máximo de consultas em paralelo")
	asJSON := flag.Bool("json", false, "Saída em JSON (uma linha por repo - NDJSON)")
	retries := flag.Int("retries", 0, "Tentativas extras para erros transitórios (5xx/timeout)")
	backoff := flag.Duration("backoff", 300*time.Millisecond, "Backoff base para retries (ex.: 300ms)")
	tokenFlag := flag.String("token", "", "GitHub token (opcional). Se vazio, usa env GITHUB_TOKEN.")
	summary := flag.Bool("summary", true, "Imprime resumo final (OK/FAIL) ao terminar")
	flag.Parse()

	// Lista de repositórios
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
		fmt.Fprintln(os.Stderr, "uso: go run . -repos owner1/repo1,owner2/repo2 [-timeout 5s] [-json] [-concurrency 5] [-retries 2] [-backoff 300ms] [-token <...>] [-summary]")
		fmt.Fprintln(os.Stderr, "     ou: go run . -owner grafana -repo grafana")
		os.Exit(2)
	}
	if *concurrency <= 0 {
		fmt.Fprintln(os.Stderr, "-concurrency deve ser >= 1")
		os.Exit(2)
	}

	// Token: flag > env
	token := *tokenFlag
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}

	type result struct {
		out Out
		ok  bool
	}

	sem := make(chan struct{}, *concurrency)
	var wg sync.WaitGroup
	results := make(chan result)

	okCount := 0
	failCount := 0
	var failed []string

	for _, full := range list {
		full := full
		owner, repo, ok := splitRepo(full)
		if !ok {
			wg.Add(1)
			go func() {
				defer wg.Done()
				results <- result{
					out: Out{Repo: full, Error: "formato inválido (use owner/repo)"},
					ok:  false,
				}
			}()
			continue
		}

		wg.Add(1)
		go func(owner, repo, full string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			rel, _, dur, err := fetchWithRetries(owner, repo, *timeout, token, *retries, *backoff)
			r := result{
				out: Out{
					Repo:        full,
					Tag:         rel.TagName,
					Name:        rel.Name,
					URL:         rel.HTMLURL,
					PublishedAt: rel.PublishedAt,
					DurationMS:  dur,
				},
				ok: err == nil,
			}
			if err != nil {
				r.out.Error = err.Error()
			}
			results <- r
		}(owner, repo, full)
	}

	// Fechar canal quando tudo acabar
	go func() {
		wg.Wait()
		close(results)
	}()

	exitCode := 0

	for r := range results {
		if *asJSON {
			_ = json.NewEncoder(os.Stdout).Encode(r.out) // NDJSON
		} else {
			if r.ok {
				fmt.Printf("[%s] OK  tag=%s  name=%s  ms=%d  url=%s\n",
					r.out.Repo, dash(r.out.Tag), dash(r.out.Name), r.out.DurationMS, r.out.URL)
			} else {
				fmt.Printf("[%s] FAIL  err=%s  ms=%d\n",
					r.out.Repo, r.out.Error, r.out.DurationMS)
			}
		}

		if r.ok {
			okCount++
		} else {
			failCount++
			failed = append(failed, r.out.Repo)
			exitCode = 1
		}
	}

	if *summary {
		if *asJSON {
			sum := struct {
				Ok    int      `json:"ok"`
				Fail  int      `json:"fail"`
				Total int      `json:"total"`
				Repos []string `json:"failed_repos,omitempty"`
			}{
				Ok: okCount, Fail: failCount, Total: okCount + failCount,
				Repos: failed,
			}
			_ = json.NewEncoder(os.Stdout).Encode(sum)
		} else {
			fmt.Printf("\nResumo: OK=%d FAIL=%d TOTAL=%d\n", okCount, failCount, okCount+failCount)
			if failCount > 0 {
				fmt.Printf("Falharam: %s\n", strings.Join(failed, ", "))
			}
		}
	}

	os.Exit(exitCode)
}
