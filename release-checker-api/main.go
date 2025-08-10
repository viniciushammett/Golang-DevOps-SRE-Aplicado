package main

import (
	"context"
	"encoding/json"
	"encoding/xml"
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

	// Comparação de versão
	Current       string `json:"current,omitempty"`
	CurrentNorm   string `json:"current_norm,omitempty"`
	LatestNorm    string `json:"latest_norm,omitempty"`
	Outdated      *bool  `json:"outdated,omitempty"`
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

// ----- SemVer (simples/robusto) -----
type semver struct {
	Major int
	Minor int
	Patch int
	Pre   string
	Raw   string
}

func normalizeVer(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if s[0] == 'v' || s[0] == 'V' {
		s = s[1:]
	}
	if i := strings.IndexByte(s, '+'); i >= 0 {
		s = s[:i]
	}
	return s
}

func parseSemver(s string) (semver, error) {
	raw := s
	s = normalizeVer(s)
	pre := ""
	if i := strings.IndexByte(s, '-'); i >= 0 {
		pre = s[:i+0]
		pre = s[i+1:]
		s = s[:i]
	}
	parts := strings.Split(s, ".")
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
	if a.Pre < b.Pre {
		return -1
	}
	return 1
}

// ---- HTTP / GitHub ----

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

// ---- current-map ----

func parseCurrentMap(s string) map[string]string {
	m := map[string]string{}
	if strings.TrimSpace(s) == "" {
		return m
	}
	entries := strings.Split(s, ",")
	for _, e := range entries {
		e = strings.TrimSpace(e)
		if e == "" {
			continue
		}
		kv := strings.SplitN(e, "=", 2)
		if len(kv) != 2 {
			continue
		}
		repo := strings.TrimSpace(kv[0])
		ver := strings.TrimSpace(kv[1])
		if repo != "" && ver != "" {
			m[repo] = ver
		}
	}
	return m
}

// ---- JUnit XML ----

type junitTestSuite struct {
	XMLName    xml.Name       `xml:"testsuite"`
	Name       string         `xml:"name,attr"`
	Tests      int            `xml:"tests,attr"`
	Failures   int            `xml:"failures,attr"`
	Time       string         `xml:"time,attr,omitempty"`
	TestCases  []junitCase    `xml:"testcase"`
	Properties *junitProps    `xml:"properties,omitempty"`
}

type junitProps struct {
	Properties []junitProp `xml:"property"`
}
type junitProp struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

type junitCase struct {
	Name      string        `xml:"name,attr"`
	Classname string        `xml:"classname,attr,omitempty"`
	Time      string        `xml:"time,attr,omitempty"`
	Failure   *junitFailure `xml:"failure,omitempty"`
	SystemOut *cdataWrap    `xml:"system-out,omitempty"`
}
type junitFailure struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Body    string `xml:",chardata"`
}
type cdataWrap struct {
	Body string `xml:",cdata"`
}

func writeJUnit(path string, suite junitTestSuite) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := xml.NewEncoder(f)
	enc.Indent("", "  ")
	_, _ = f.WriteString(xml.Header)
	return enc.Encode(suite)
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
	current := flag.String("current", "", "Versão instalada/local (ex.: v1.2.3). Usada como padrão.")
	outdatedExit := flag.Int("outdated-exitcode", 3, "Exit code quando desatualizado (default 3)")

	currentMapFlag := flag.String("current-map", "", "Versões por repo: owner/repo=vX.Y.Z,... (tem precedência sobre -current)")
	junitPath := flag.String("junit", "", "Grava relatório JUnit XML no caminho informado")
	summaryJSONPath := flag.String("summary-json", "", "Grava o resumo final em JSON no caminho informado")
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

	// Parse do current-map
	curMap := parseCurrentMap(*currentMapFlag)

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
			// rate info opcional
			if *showRate {
				r.out.Rate = &rate
			}
			if err != nil {
				r.out.Error = err.Error()
				results <- r
				return
			}

			// current por repo (map) tem precedência, depois -current global
			cur := curMap[full]
			if cur == "" {
				cur = *current
			}
			if cur != "" {
				curSV, errCur := parseSemver(cur)
				latSV, errLat := parseSemver(rel.TagName)

				r.out.Current = cur
				if errCur == nil {
					r.out.CurrentNorm = normalizeVer(cur)
				}
				if errLat == nil {
					r.out.LatestNorm = normalizeVer(rel.TagName)
				}

				if errCur != nil || errLat != nil {
					r.out.CompareReason = "semver inválido (current ou latest)"
				} else {
					switch cmp := compareSemver(curSV, latSV); {
					case cmp < 0:
						od := true
						r.out.Outdated = &od
						r.out.CompareReason = "current < latest"
					case cmp == 0:
						od := false
						r.out.Outdated = &od
						r.out.CompareReason = "current == latest"
					default:
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
	// Coletamos para também conseguir escrever JUnit ao final
	var collected []result

	for r := range results {
		collected = append(collected, r)

		if *asJSON {
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
				fmt.Printf("[%s] FAIL  err=%s  ms=%d\n", r.out.Repo, r.out.Error, r.out.DurationMS)
			}
			if r.out.Rate != nil {
				fmt.Printf("   rate: limit=%d remaining=%d reset=%s\n",
					r.out.Rate.Limit, r.out.Rate.Remaining, humanReset(r.out.Rate.ResetUnix))
			}
		}

		if r.ok {
			okCount++
		} else {
			failCount++
			failed = append(failed, r.out.Repo)
			exitCode = 1
		}
		if r.ok && r.out.Outdated != nil && *r.out.Outdated {
			outdatedCount++
			outdatedRepos = append(outdatedRepos, r.out.Repo)
		}
	}

	// Resumo final
	total := okCount + failCount
	if *summary {
		if *asJSON {
			sum := struct {
				Ok       int      `json:"ok"`
				Fail     int      `json:"fail"`
				Outdated int      `json:"outdated"`
				Total    int      `json:"total"`
				Failed   []string `json:"failed_repos,omitempty"`
				Oldies   []string `json:"outdated_repos,omitempty"`
			}{
				Ok: okCount, Fail: failCount, Outdated: outdatedCount, Total: total,
				Failed: failed, Oldies: outdatedRepos,
			}
			_ = json.NewEncoder(os.Stdout).Encode(sum)

			// PASSO 7: gravar summary JSON em arquivo (opcional)
			if *summaryJSONPath != "" {
				_ = writeJSONFile(*summaryJSONPath, sum)
			}
		} else {
			fmt.Printf("\nResumo: OK=%d FAIL=%d OUTDATED=%d TOTAL=%d\n", okCount, failCount, outdatedCount, total)
			if failCount > 0 {
				fmt.Printf("Falharam: %s\n", strings.Join(failed, ", "))
			}
			if outdatedCount > 0 {
				fmt.Printf("Desatualizados: %s\n", strings.Join(outdatedRepos, ", "))
			}
		}
	}

	// PASSO 7: JUnit XML (opcional)
	if *junitPath != "" {
		suite := junitTestSuite{
			Name:     "github-latest-release",
			Tests:    total,
			Failures: 0, // vamos somar abaixo
			Properties: &junitProps{Properties: []junitProp{
				{Name: "generated_at", Value: time.Now().Format(time.RFC3339)},
			}},
		}
		for _, r := range collected {
			tc := junitCase{
				Name:      r.out.Repo,
				Classname: "release.check",
				Time:      fmt.Sprintf("%.3f", float64(r.out.DurationMS)/1000.0),
				SystemOut: &cdataWrap{Body: fmt.Sprintf("tag=%s name=%s url=%s", dash(r.out.Tag), dash(r.out.Name), r.out.URL)},
			}
			// Consideramos "falha" no JUnit tanto erro “hard” quanto “outdated”
			if !r.ok {
				suite.Failures++
				tc.Failure = &junitFailure{
					Message: "API error",
					Type:    "api",
					Body:    r.out.Error,
				}
			} else if r.out.Outdated != nil && *r.out.Outdated {
				suite.Failures++
				tc.Failure = &junitFailure{
					Message: "Outdated",
					Type:    "version",
					Body:    fmt.Sprintf("current=%s latest=%s reason=%s", dash(r.out.Current), dash(r.out.Tag), dash(r.out.CompareReason)),
				}
			}
			suite.TestCases = append(suite.TestCases, tc)
		}
		if err := writeJUnit(*junitPath, suite); err != nil {
			fmt.Fprintf(os.Stderr, "erro ao escrever JUnit: %v\n", err)
			// não altera exit code principal
		}
	}

	// Exit code para desatualizado (se não houve falhas hard)
	if exitCode == 0 && outdatedCount > 0 {
		exitCode = *outdatedExit
	}
	os.Exit(exitCode)
}

// ---- helpers de arquivo ----

func writeJSONFile(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
