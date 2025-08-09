package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
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

type RateInfo struct {
	Limit     int   `json:"limit,omitempty"`
	Remaining int   `json:"remaining,omitempty"`
	ResetUnix int64 `json:"reset_unix,omitempty"`
}

type Out struct {
	Repo         string    `json:"repo"` // owner/repo
	Tag          string    `json:"tag"`
	Name         string    `json:"name"`
	URL          string    `json:"url"`
	PublishedAt  time.Time `json:"published_at"`
	DurationMS   int64     `json:"duration_ms"`
	Error        string    `json:"error,omitempty"`
	Rate         *RateInfo `json:"rate,omitempty"`

	// Passo 6 — comparação de versão
	Current       string `json:"current,omitempty"`
	CurrentNorm   string `json:"current_norm,omitempty"`
	LatestNorm    string `json:"latest_norm,omitempty"`
	Outdated      *bool  `json:"outdated,omitempty"` // ponteiro pra omitir quando vazio
	CompareReason string `json:"compare_reason,omitempty"`
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

func isRetryable(status int, err error) bool {
	if status >= 500 && status <= 599 {
		return true
	}
	if err != nil {
		if ne, ok := err.(net.Error); ok {
			if ne.Timeout() || ne.Temporary() {
				return true
			}
		}
		if strings.Contains(strings.ToLower(err.Error()), "timeout") {
			return true
		}
	}
	return false
}

// parseRate lê X-RateLimit-* dos headers
func parseRate(h http.Header) RateInfo {
	ri := RateInfo{}
	if v := h.Get("X-RateLimit-Limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			ri.Limit = n
		}
	}
	if v := h.Get("X-RateLimit-Remaining"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			ri.Remaining = n
		}
	}
	if v := h.Get("X-RateLimit-Reset"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			ri.ResetUnix = n
		}
	}
	return ri
}

func humanReset(ts int64) string {
	if ts == 0 {
		return "-"
	}
	return time.Unix(ts, 0).Format(time.RFC3339)
}

// ----- SemVer (simplificado e robusto o suficiente) -----
// Normaliza: remove prefixo "v"/"V", corta build metadata (+) e separa pre-release (-)
type semver struct {
	Major int
	Minor int
	Patch int
	Pre   string // pré-release inteiro (ex.: "rc.1"). Sem parsing fino; só ordenação básica.
	Raw   string // para debug
}

func normalizeVer(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if s[0] == 'v' || s[0] == 'V' {
		s = s[1:]
	}
	// remove build metadata
	if i := strings.IndexByte(s, '+'); i >= 0 {
		s = s[:i]
	}
	return s
}

func parseSemver(s string) (semver, error) {
	raw := s
	s = normalizeVer(s)
	// separa pre-release
	pre := ""
	if i := strings.IndexByte(s, '-'); i >= 0 {
		pre = s[i+1:]
		s = s[:i]
	}
	parts := strings.Split(s, ".")
	// Padding para até 3 partes
	for len(parts) < 3 {
		parts = append(parts, "0")
	}
	if len(parts) > 3 {
		parts = parts[:3]
	}
	atoi := func(x string) (int, error) {
		if x == "" {
			return 0, nil
		}
		return strconv.Atoi(x)
	}
	maj, err := atoi(parts[0])
	if err != nil {
		return semver{Raw: raw}, fmt.Errorf("semver major inválido: %q", parts[0])
	}
	min, err := atoi(parts[1])
	if err != nil {
		return semver{Raw: raw}, fmt.Errorf("semver minor inválido: %q", parts[1])
	}
	pat, err := atoi(parts[2])
	if err != nil {
		return semver{Raw: raw}, fmt.Errorf("semver patch inválido: %q", parts[2])
	}
	return semver{Major: maj, Minor: min, Patch: pat, Pre: pre, Raw: raw}, nil
}

// compareSemver: -1 se a<b, 0 se a==b, 1 se a>b
// Regras: major/minor/patch numérico; sem pre-release > com pre-release; se ambos pre, compara lexicograficamente.
func compareSemver(a, b semver) int {
	if a.Major != b.Major {
		if a.Major < b.Major {
			return -1
		}
		return 1
	}
	if a.Minor != b.Minor {
		if a.Minor < b.Minor {
			return -1
		}
		return 1
	}
	if a.Patch != b.Patch {
		if a.Patch < b.Patch {
			return -1
		}
		return 1
	}
	if a.Pre == "" && b.Pre != "" {
		return 1
	}
	if a.Pre != "" && b.Pre == "" {
		return -1
	}
	if a.Pre == b.Pre {
		return 0
	}
	// Fallback simples: ordem lexicográfica do pre-release
	if a.Pre < b.Pre {
		return -1
	}
	return 1
}

// ---- HTTP / GitHub ----

// fetchLatest executa UMA tentativa, com timeout e (opcional) token.
// Retorna release, status HTTP, duração em ms, rate info e erro.
func fetchLatest(owner, repo string, timeout time.Duration, token string) (Release, int, int64, RateInfo, error) {
	var zero Release
	var zr RateInfo

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return zero, 0, 0, zr, err
	}
	req.Header.Set("User-Agent", "go-cli-learning/1.0")
	if token != "" {
		req.Header.Set("Authorization", "token "+token)
	}

	client := &http.Client{Timeout: timeout}

	start := time.Now()
	resp, err := client.Do(req)
	elapsed := time.Since(start).Milliseconds()
	if err != nil {
		return zero, 0, elapsed, zr, err
	}
	defer resp.Body.Close()

	rate := parseRate(resp.Header)

	switch resp.StatusCode {
	case http.StatusOK:
		// segue
	case http.StatusNotFound:
		return zero, resp.StatusCode, elapsed, rate, fmt.Errorf("not found (ou sem releases)")
	case http.StatusForbidden:
		return zero, resp.StatusCode, elapsed, rate, fmt.Errorf("forbidden/rate limit (403) — remaining=%d reset=%s", rate.Remaining, humanReset(rate.ResetUnix))
	default:
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return zero, resp.StatusCode, elapsed, rate, fmt.Errorf("status inesperado: %s", resp.Status)
		}
	}

	var rel Release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return zero, resp.StatusCode, elapsed, rate, fmt.Errorf("decode: %w", err)
	}
	return rel, resp.StatusCode, elapsed, rate, nil
}

// fetchWithRetries aplica backoff exponencial: base * 2^attempt
func fetchWithRetries(owner, repo string, timeout time.Duration, token string, retries int, backoffBase time.Duration) (Release, int, int64, RateInfo, error) {
	var lastRel Release
	var lastStatus int
	var lastDur int64
	var lastRate RateInfo
	var lastErr error

	for attempt := 0; attempt <= retries; attempt++ {
		rel, status, dur, rate, err := fetchLatest(owner, repo, timeout, token)
		lastRel, lastStatus, lastDur, lastRate, lastErr = rel, status, dur, rate, err

		if err == nil {
			return rel, status, dur, rate, nil
		}
		if !isRetryable(status, err) || attempt == retries {
			break
		}
		sleep := backoffBase << attempt
		time.Sleep(sleep)
	}
	return lastRel, lastStatus, lastDur, lastRate, lastErr
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
	showRate := flag.Bool("ratelimit", false, "Exibe informações de rate limit (X-RateLimit-*)")

	// Comparação de versão
	current := flag.String("current", "", "Versão instalada/local (ex.: v1.2.3). Aplica a todos os repositórios.")
	outdatedExit := flag.Int("outdated-exitcode", 3, "Exit code para quando estiver desatualizado (default 3)")
	flag.Parse()

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
		fmt.Fprintln(os.Stderr, "uso: go run . -repos owner1/repo1,owner2/repo2 [flags...]")
		fmt.Fprintln(os.Stderr, "     ou: go run . -owner grafana -repo grafana [flags...]")
		os.Exit(2)
	}
	if *concurrency <= 0 {
		fmt.Fprintln(os.Stderr, "-concurrency deve ser >= 1")
		os.Exit(2)
	}

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
	outdatedCount := 0
	var failed []string
	var outdatedRepos []string

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

			rel, _, dur, rate, err := fetchWithRetries(owner, repo, *timeout, token, *retries, *backoff)

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
			if *showRate {
				r.out.Rate = &rate
			}
			if err != nil {
				r.out.Error = err.Error()
				results <- r
				return
			}

			// Comparação de versão (se -current foi fornecido)
			if *current != "" {
				curSV, errCur := parseSemver(*current)
				latSV, errLat := parseSemver(rel.TagName)

				r.out.Current = *current
				if errCur == nil {
					r.out.CurrentNorm = normalizeVer(*current)
				}
				if errLat == nil {
					r.out.LatestNorm = normalizeVer(rel.TagName)
				}

				if errCur != nil || errLat != nil {
					// Não conseguimos comparar formalmente
					r.out.CompareReason = "semver inválido (current ou latest)"
				} else {
					cmp := compareSemver(curSV, latSV)
					switch {
					case cmp < 0:
						// current < latest => desatualizado
						od := true
						r.out.Outdated = &od
						r.out.CompareReason = "current < latest"
					case cmp == 0:
						od := false
						r.out.Outdated = &od
						r.out.CompareReason = "current == latest"
					case cmp > 0:
						od := false
						r.out.Outdated = &od
						r.out.CompareReason = "current > latest"
					}
				}
			}

			results <- r
		}(owner, repo, full)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	exitCode := 0

	for r := range results {
		// contador de sucesso/falha
		if r.ok {
			okCount++
		} else {
			failCount++
			failed = append(failed, r.out.Repo)
			exitCode = 1
		}

		// saída
		if asJSON := flag.Lookup("json").Value.String(); asJSON == "true" {
			_ = json.NewEncoder(os.Stdout).Encode(r.out)
		} else {
			if r.ok {
				fmt.Printf("[%s] OK  tag=%s  name=%s  ms=%d  url=%s\n",
					r.out.Repo, dash(r.out.Tag), dash(r.out.Name), r.out.DurationMS, r.out.URL)
				if r.out.Current != "" {
					odStr := "-"
					if r.out.Outdated != nil {
						if *r.out.Outdated {
							odStr = "OUTDATED"
						} else {
							odStr = "up-to-date"
						}
					}
					fmt.Printf("   compare: current=%s  latest=%s  result=%s  reason=%s\n",
						dash(r.out.Current), dash(r.out.Tag), odStr, dash(r.out.CompareReason))
				}
			} else {
				fmt.Printf("[%s] FAIL  err=%s  ms=%d\n",
					r.out.Repo, r.out.Error, r.out.DurationMS)
			}
			if r.out.Rate != nil {
				fmt.Printf("   rate: limit=%d remaining=%d reset=%s\n",
					r.out.Rate.Limit, r.out.Rate.Remaining, humanReset(r.out.Rate.ResetUnix))
			}
		}

		// marca outdated
		if r.ok && r.out.Outdated != nil && *r.out.Outdated {
			outdatedCount++
			outdatedRepos = append(outdatedRepos, r.out.Repo)
		}
	}

	// resumo
	if flag.Lookup("summary").Value.String() == "true" {
		if flag.Lookup("json").Value.String() == "true" {
			sum := struct {
				Ok       int      `json:"ok"`
				Fail     int      `json:"fail"`
				Outdated int      `json:"outdated"`
				Total    int      `json:"total"`
				Failed   []string `json:"failed_repos,omitempty"`
				Oldies   []string `json:"outdated_repos,omitempty"`
			}{
				Ok: okCount, Fail: failCount, Outdated: outdatedCount, Total: okCount + failCount,
				Failed: failed, Oldies: outdatedRepos,
			}
			_ = json.NewEncoder(os.Stdout).Encode(sum)
		} else {
			fmt.Printf("\nResumo: OK=%d FAIL=%d OUTDATED=%d TOTAL=%d\n",
				okCount, failCount, outdatedCount, okCount+failCount)
			if failCount > 0 {
				fmt.Printf("Falharam: %s\n", strings.Join(failed, ", "))
			}
			if outdatedCount > 0 {
				fmt.Printf("Desatualizados: %s\n", strings.Join(outdatedRepos, ", "))
			}
		}
	}

	// exit code para desatualizado (se não houve falhas “hard”)
	if exitCode == 0 && outdatedCount > 0 {
		exitCode = *outdatedExit
	}
	os.Exit(exitCode)
}
