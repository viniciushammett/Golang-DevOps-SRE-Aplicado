package main

// Projeto: logwatcher
// Objetivo: Seguir logs em tempo real (tail -f) com regex, tratar rotação, múltiplos arquivos (glob),
// deduplicar alertas com cooldown, expor métricas Prometheus e (por último) enviar alertas via webhook.
//
// Passos implementados:
// 1) Tail + Regex
// 3) Rotação de logs (fsnotify + truncamento)
// 4) Múltiplos arquivos (flag -files com glob) e prefixo por fonte
// 5) Deduplicação & Cooldown (buffer de agregação)
// 6) Prometheus /metrics (counters/gauges)
// 2) Webhook Slack/Discord (flag -webhook) – implementado por último, mas já no binário.
//
// Observação de design: mantemos um watcher por diretório e uma goroutine tailer por arquivo.
// As métricas são expostas em /metrics; o servidor HTTP é opcional (levantado ao passar -metrics-addr).

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"hash/fnv"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// =============== Métricas (Passo 6) ===============

var (
	linesRead = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "logwatcher_lines_read_total",
			Help: "Total de linhas lidas por arquivo.",
		},
		[]string{"file"},
	)
	matches = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "logwatcher_matches_total",
			Help: "Total de matches por arquivo e regex.",
		},
		[]string{"file", "pattern"},
	)
	alertsSent = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "logwatcher_alerts_sent_total",
			Help: "Total de alertas enviados por arquivo e regex.",
		},
		[]string{"file", "pattern"},
	)
	lastMatchTS = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "logwatcher_last_match_timestamp_seconds",
			Help: "Timestamp (epoch seconds) do último match por arquivo e regex.",
		},
		[]string{"file", "pattern"},
	)
	activeTargets = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "logwatcher_active_targets",
			Help: "Quantidade de arquivos sendo seguidos (ativos).",
		},
	)
)

func init() {
	prometheus.MustRegister(linesRead, matches, alertsSent, lastMatchTS, activeTargets)
}

// =============== Config e Flags ===============

type config struct {
	// Entrada
	File       string        // Passo 1: único arquivo (-file)
	FilesGlob  string        // Passo 4: múltiplos arquivos via glob (-files)
	Pattern    string        // Passo 1: regex
	FromStart  bool          // Passo 1: ler do início
	PollEvery  time.Duration // Passo 1: polling entre leituras (quando não há linha nova)

	// Dedup/Cooldown (Passo 5)
	Cooldown          time.Duration // janela de cooldown
	BundleWindow      time.Duration // janela para agrupar N ocorrências em 1 alerta
	BundleMaxMessages int           // máximo de mensagens por bundle/alert

	// Métricas (Passo 6)
	MetricsAddr string // ex.: ":9100" para expor /metrics

	// Webhook (Passo 2 - por último)
	WebhookURL string // Slack ou Discord Webhook
	Channel    string // opcional (Slack custom), será incluído no payload "text"
	Title      string // prefixo do alerta
}

// =============== Tailer por arquivo (Passos 1, 3) ===============

type tailer struct {
	path      string
	dir       string
	base      string
	prefix    string
	fromStart bool
	pollEvery time.Duration
	rx        *regexp.Regexp

	mu       sync.Mutex
	f        *os.File
	reader   *bufio.Reader
	lastSize int64

	dirWatcher *fsnotify.Watcher // observamos o diretório para detectar rotação
	eventsCh   chan string       // canal de eventos "match" para Dedup/Alerting
	stop       chan struct{}
}

// newTailer: constrói o tailer para um arquivo específico.
func newTailer(path string, prefix string, fromStart bool, pollEvery time.Duration, rx *regexp.Regexp, eventsCh chan string) *tailer {
	abs, _ := filepath.Abs(path)
	return &tailer{
		path:      abs,
		dir:       filepath.Dir(abs),
		base:      filepath.Base(abs),
		prefix:    prefix,
		fromStart: fromStart,
		pollEvery: pollEvery,
		rx:        rx,
		eventsCh:  eventsCh,
		stop:      make(chan struct{}),
	}
}

func (t *tailer) openInitial() error {
	f, err := os.Open(t.path)
	if err != nil {
		return fmt.Errorf("abrindo arquivo inicial: %w", err)
	}
	t.f = f
	t.reader = bufio.NewReader(f)
	info, err := f.Stat()
	if err == nil {
		t.lastSize = info.Size()
	}
	if !t.fromStart {
		// tail -f padrão: inicia no fim
		if _, err := f.Seek(0, io.SeekEnd); err != nil {
			return fmt.Errorf("seek end: %w", err)
		}
		if info, err := f.Stat(); err == nil {
			t.lastSize = info.Size()
		}
	}
	return nil
}

func (t *tailer) reopenFromStart() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.f != nil {
		_ = t.f.Close()
	}
	f, err := os.Open(t.path)
	if err != nil {
		return err
	}
	t.f = f
	t.reader = bufio.NewReader(f)
	t.fromStart = true // nesta reabertura lemos do início
	t.lastSize = 0
	return nil
}

func (t *tailer) sameInode() (bool, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.f == nil {
		return false, nil
	}
	fi1, err := t.f.Stat()
	if err != nil {
		return false, err
	}
	fi2, err := os.Stat(t.path)
	if err != nil {
		return false, err
	}
	st1, ok1 := fi1.Sys().(*syscall.Stat_t)
	st2, ok2 := fi2.Sys().(*syscall.Stat_t)
	if !ok1 || !ok2 {
		// fallback onde não há Stat_t (Windows, por ex.)
		return strings.EqualFold(fi1.Name(), fi2.Name()), nil
	}
	return st1.Ino == st2.Ino && st1.Dev == st2.Dev, nil
}

func (t *tailer) reopenAfterRotation() error {
	// Espera até o novo arquivo base aparecer
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(t.path); err == nil {
			return t.reopenFromStart()
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("novo arquivo base não apareceu: %s", t.path)
}

func (t *tailer) handleFsEvent(ev fsnotify.Event) {
	ename := filepath.Base(ev.Name)
	if ename != t.base {
		return
	}
	if ev.Op&(fsnotify.Rename|fsnotify.Remove) != 0 {
		// rotação típica: app.log -> app.log.1 e novo app.log
		fmt.Printf("[%s] 🔁 rotação detectada: %s (%s)\n", t.prefix, ev.Name, opString(ev.Op))
		if err := t.reopenAfterRotation(); err != nil {
			fmt.Printf("[%s] falha ao reabrir após rotação: %v\n", t.prefix, err)
		}
		return
	}
	if ev.Op&(fsnotify.Create) != 0 {
		fmt.Printf("[%s] ➕ novo arquivo base: %s\n", t.prefix, ev.Name)
		if err := t.reopenFromStart(); err != nil {
			fmt.Printf("[%s] falha ao reabrir novo arquivo: %v\n", t.prefix, err)
		}
		return
	}
	if ev.Op&(fsnotify.Write) != 0 {
		// se inode mudou silenciosamente, reabrir
		if same, err := t.sameInode(); err == nil && !same {
			fmt.Printf("[%s] 🔄 inode mudou; reabrindo\n", t.prefix)
			if err := t.reopenFromStart(); err != nil {
				fmt.Printf("[%s] falha ao reabrir após inode change: %v\n", t.prefix, err)
			}
		}
	}
}

func opString(op fsnotify.Op) string {
	var parts []string
	if op&fsnotify.Create != 0 {
		parts = append(parts, "CREATE")
	}
	if op&fsnotify.Write != 0 {
		parts = append(parts, "WRITE")
	}
	if op&fsnotify.Remove != 0 {
		parts = append(parts, "REMOVE")
	}
	if op&fsnotify.Rename != 0 {
		parts = append(parts, "RENAME")
	}
	if op&fsnotify.Chmod != 0 {
		parts = append(parts, "CHMOD")
	}
	return strings.Join(parts, "|")
}

func (t *tailer) follow(ctx context.Context) error {
	if err := t.openInitial(); err != nil {
		return err
	}

	// watcher no diretório do arquivo
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("fsnotify: %w", err)
	}
	t.dirWatcher = w
	if err := w.Add(t.dir); err != nil {
		return fmt.Errorf("watcher add dir: %w", err)
	}

	fmt.Printf("[%s] 📄 seguindo: %s (from-start=%v, poll=%s)\n", t.prefix, t.path, t.fromStart, t.pollEvery)

	go func() {
		for {
			select {
			case ev := <-w.Events:
				t.handleFsEvent(ev)
			case err := <-w.Errors:
				fmt.Printf("[%s] watcher erro: %v\n", t.prefix, err)
			case <-t.stop:
				return
			}
		}
	}()

	// loop principal de leitura
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			// tenta ler uma linha
			line, err := t.reader.ReadString('\n')
			if err == nil {
				linesRead.WithLabelValues(t.path).Inc()
				if t.rx != nil {
					if t.rx.MatchString(line) {
						fmt.Printf("[%s][MATCH] %s", t.prefix, line)
						matches.WithLabelValues(t.path, t.rx.String()).Inc()
						lastMatchTS.WithLabelValues(t.path, t.rx.String()).Set(float64(time.Now().Unix()))
						// envia o evento para dedup/alert
						select {
						case t.eventsCh <- fmt.Sprintf("%s%s", t.prefixPrefix(), line):
						default:
							// se o canal estiver cheio, descartamos para não travar o tail
						}
					}
				} else {
					fmt.Printf("[%s] %s", t.prefix, line)
				}

				// atualiza lastSize
				if info, e := t.f.Stat(); e == nil {
					t.lastSize = info.Size()
				}
				continue
			}

			// sem nova linha: checar truncamento
			if info, e := t.f.Stat(); e == nil {
				cur := info.Size()
				if cur < t.lastSize {
					fmt.Printf("[%s] 🔁 truncamento; reabrindo do início\n", t.prefix)
					if err := t.reopenFromStart(); err != nil {
						fmt.Printf("[%s] falha ao reabrir após truncamento: %v\n", t.prefix, err)
					}
				} else {
					t.lastSize = cur
				}
			}

			// espera um pouco
			time.Sleep(t.pollEvery)
		}
	}
}

func (t *tailer) prefixPrefix() string {
	if t.prefix == "" {
		return ""
	}
	return fmt.Sprintf("[%s] ", t.prefix)
}

func (t *tailer) close() {
	close(t.stop)
	if t.dirWatcher != nil {
		_ = t.dirWatcher.Close()
	}
	if t.f != nil {
		_ = t.f.Close()
	}
}

// =============== Deduplicação & Cooldown (Passo 5) ===============

type dedupKey struct {
	FilePrefix string
	Pattern    string
	// poderíamos incluir outras chaves (ex.: hostname) se necessário
}

type bundler struct {
	cfg      config
	rxString string

	mu      sync.Mutex
	lastHit map[dedupKey]time.Time     // última vez que enviamos alerta para esta chave (cooldown)
	buffer  map[dedupKey][]string      // mensagens acumuladas durante janela
	timer   map[dedupKey]*time.Timer   // timers por chave para flush
	outFn   func(key dedupKey, lines []string) // destino do bundle (ex.: webhook)
}

func newBundler(cfg config, rx *regexp.Regexp, outFn func(key dedupKey, lines []string)) *bundler {
	s := ""
	if rx != nil {
		s = rx.String()
	}
	return &bundler{
		cfg:      cfg,
		rxString: s,
		lastHit:  map[dedupKey]time.Time{},
		buffer:   map[dedupKey][]string{},
		timer:    map[dedupKey]*time.Timer{},
		outFn:    outFn,
	}
}

func (b *bundler) push(filePrefix string, line string) {
	key := dedupKey{FilePrefix: filePrefix, Pattern: b.rxString}

	now := time.Now()

	b.mu.Lock()
	defer b.mu.Unlock()

	// cooldown: se enviamos há pouco tempo, só acumula no buffer até a janela "bundleWindow"
	if last, ok := b.lastHit[key]; ok {
		if now.Sub(last) < b.cfg.Cooldown {
			// apenas adiciona no buffer para o próximo flush
			b.buffer[key] = appendClip(b.buffer[key], line, b.cfg.BundleMaxMessages)
			return
		}
	}

	// primeira ocorrência ou cooldown passado → inicia buffer e agenda flush
	b.buffer[key] = appendClip(b.buffer[key], line, b.cfg.BundleMaxMessages)
	if _, ok := b.timer[key]; !ok {
		// agenda flush para o fim da janela
		b.timer[key] = time.AfterFunc(b.cfg.BundleWindow, func() {
			b.flush(key)
		})
	}
}

// flush envia o bundle e atualiza o cooldown
func (b *bundler) flush(key dedupKey) {
	b.mu.Lock()
	lines := b.buffer[key]
	delete(b.buffer, key)
	t := b.timer[key]
	delete(b.timer, key)
	b.mu.Unlock()

	if t != nil {
		t.Stop()
	}
	if len(lines) == 0 {
		return
	}
	// registra o envio agora
	b.mu.Lock()
	b.lastHit[key] = time.Now()
	b.mu.Unlock()

	b.outFn(key, lines)
}

func appendClip(slice []string, s string, max int) []string {
	slice = append(slice, strings.TrimRight(s, "\n"))
	if max > 0 && len(slice) > max {
		// mantemos só os últimos "max"
		return slice[len(slice)-max:]
	}
	return slice
}

// =============== Webhook (Passo 2 – por último) ===============

// Suporte simples a Slack e Discord: mandamos JSON com "text" (Slack) e "content" (Discord).
// A maioria dos webhooks ignora campos desconhecidos; então enviamos ambos.
type webhookPayload struct {
	Text    string `json:"text,omitempty"`    // Slack
	Content string `json:"content,omitempty"` // Discord
}

func sendWebhook(ctx context.Context, url string, title string, channel string, filePrefix string, pattern string, lines []string) error {
	if url == "" {
		return nil
	}
	var buf bytes.Buffer
	if title != "" {
		buf.WriteString(title)
		buf.WriteString(" ")
	}
	if channel != "" {
		buf.WriteString("#")
		buf.WriteString(channel)
		buf.WriteString(" ")
	}
	if filePrefix != "" {
		buf.WriteString("[")
		buf.WriteString(filePrefix)
		buf.WriteString("] ")
	}
	if pattern != "" {
		buf.WriteString("(pattern: ")
		buf.WriteString(pattern)
		buf.WriteString(") ")
	}
	buf.WriteString(fmt.Sprintf("x%d\n", len(lines)))
	for _, ln := range lines {
		buf.WriteString("• ")
		buf.WriteString(ln)
		buf.WriteByte('\n')
	}

	payload := webhookPayload{
		Text:    buf.String(),
		Content: buf.String(),
	}

	b, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "logwatcher/1.0")
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	// consideramos 2xx sucesso
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook status não-2xx: %s", resp.Status)
	}
	return nil
}

// =============== Utilidades ===============

func mustCompileRegex(pat string) *regexp.Regexp {
	if pat == "" {
		return nil
	}
	rx, err := regexp.Compile(pat)
	if err != nil {
		fmt.Fprintf(os.Stderr, "regex inválida: %v\n", err)
		os.Exit(2)
	}
	return rx
}

func hashShort(s string) string {
	// usado para gerar prefixos estáveis quando -files aponta para muitos caminhos
	h := fnv.New32a()
	_, _ = h.Write([]byte(s))
	return fmt.Sprintf("%08x", h.Sum32())
}

func sha1Short(s string) string {
	sum := sha1.Sum([]byte(s))
	return hex.EncodeToString(sum[:])[:8]
}

// =============== Servidor de Métricas (Passo 6) ===============

func startMetricsServer(addr string, readyCh chan struct{}) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	srv := &http.Server{Addr: addr, Handler: mux}
	go func() {
		close(readyCh)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "metrics server error: %v\n", err)
		}
	}()
	return srv
}

// =============== main ===============

func main() {
	var cfg config

	// Entrada
	flag.StringVar(&cfg.File, "file", "", "Caminho de um arquivo único (ex.: /var/log/nginx/error.log)")
	flag.StringVar(&cfg.FilesGlob, "files", "", "Glob de arquivos (ex.: \"/var/log/nginx/*.log,/var/log/app/*.log\")")
	flag.StringVar(&cfg.Pattern, "pattern", "", "Regex para filtrar linhas (ex.: '(?i)error|critical')")
	flag.BoolVar(&cfg.FromStart, "from-start", false, "Ler desde o início (padrão: segui do fim)")
	flag.DurationVar(&cfg.PollEvery, "poll", 300*time.Millisecond, "Intervalo de polling quando não há novas linhas")

	// Dedup/Cooldown
	flag.DurationVar(&cfg.Cooldown, "cooldown", 30*time.Second, "Janela de cooldown por chave (arquivo+regex) para evitar spam")
	flag.DurationVar(&cfg.BundleWindow, "bundle-window", 5*time.Second, "Janela para agrupar ocorrências em um único alerta")
	flag.IntVar(&cfg.BundleMaxMessages, "bundle-max", 20, "Máximo de linhas por alerta (bundle)")

	// Métricas
	flag.StringVar(&cfg.MetricsAddr, "metrics-addr", "", "Endereço para expor /metrics (ex.: :9100). Vazio desativa.")

	// Webhook (por último)
	flag.StringVar(&cfg.WebhookURL, "webhook", "", "URL do webhook (Slack/Discord). Vazio = sem envio.")
	flag.StringVar(&cfg.Channel, "channel", "", "Nome do canal (opcional, Slack).")
	flag.StringVar(&cfg.Title, "title", "Logwatcher", "Título/Prefixo do alerta.")

	flag.Parse()

	if cfg.File == "" && cfg.FilesGlob == "" {
		fmt.Fprintln(os.Stderr, "uso: go run . -file /caminho/do.log [-pattern REGEX] [opções...]")
		fmt.Fprintln(os.Stderr, " ou : go run . -files \"/var/log/nginx/*.log,/var/log/app/*.log\" [-pattern REGEX] [opções...]")
		os.Exit(2)
	}

	fmt.Printf("🖥️  OS=%s ARCH=%s | regex=%q | metrics=%q\n", runtime.GOOS, runtime.GOARCH, cfg.Pattern, cfg.MetricsAddr)

	rx := mustCompileRegex(cfg.Pattern)

	// Servidor de métricas (opcional)
	var metricsSrv *http.Server
	if cfg.MetricsAddr != "" {
		ready := make(chan struct{})
		metricsSrv = startMetricsServer(cfg.MetricsAddr, ready)
		<-ready
		fmt.Printf("📈 /metrics em %s\n", cfg.MetricsAddr)
	}

	// Resolver lista de arquivos-alvo (Passo 4)
	targets := make([]string, 0, 8)
	if cfg.File != "" {
		targets = append(targets, cfg.File)
	}
	if cfg.FilesGlob != "" {
		for _, g := range strings.Split(cfg.FilesGlob, ",") {
			g = strings.TrimSpace(g)
			if g == "" {
				continue
			}
			matches, err := filepath.Glob(g)
			if err != nil {
				fmt.Fprintf(os.Stderr, "glob inválido %q: %v\n", g, err)
				continue
			}
			targets = append(targets, matches...)
		}
	}
	if len(targets) == 0 {
		fmt.Fprintln(os.Stderr, "nenhum arquivo alvo encontrado")
		os.Exit(2)
	}

	activeTargets.Set(float64(len(targets)))

	// Canal de eventos de match → bundler (dedup/cooldown)
	eventsCh := make(chan string, 1024)

	// Função de envio de alertas (Webhook) utilizada pelo bundler
	sendFn := func(key dedupKey, lines []string) {
		// Incrementa métrica de alertas
		alertsSent.WithLabelValues(key.FilePrefix, cfg.Pattern).Inc()

		// Envia webhook (se configurado)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := sendWebhook(ctx, cfg.WebhookURL, cfg.Title, cfg.Channel, key.FilePrefix, key.Pattern, lines); err != nil {
			fmt.Fprintf(os.Stderr, "webhook erro: %v\n", err)
		} else {
			fmt.Printf("[%s] 🔔 alerta enviado (bundle=%d)\n", key.FilePrefix, len(lines))
		}
	}

	// Um bundler global (poderíamos ter por regex, mas 1 já atende)
	bndl := newBundler(cfg, rx, sendFn)

	// ctx para encerrar com Ctrl+C
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Inicia goroutine consumidora do canal de eventos (dedup/cooldown)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-eventsCh:
				// derivar o prefixo (entre colchetes) para identificar a chave no bundler
				filePrefix := extractPrefix(msg) // ex.: "[nginx-error]" ⇒ "nginx-error"
				if filePrefix == "" {
					filePrefix = "log"
				}
				bndl.push(filePrefix, msg)
			}
		}
	}()

	// Lança um tailer por arquivo-alvo
	var wg sync.WaitGroup
	tailers := make([]*tailer, 0, len(targets))

	for _, p := range targets {
		abs, _ := filepath.Abs(p)

		// Prefixo amigável por fonte: nome do arquivo + hash curto do path
		base := filepath.Base(abs)
		prefix := fmt.Sprintf("%s#%s", base, sha1Short(abs))

		t := newTailer(abs, prefix, cfg.FromStart, cfg.PollEvery, rx, eventsCh)
		tailers = append(tailers, t)

		wg.Add(1)
		go func(tl *tailer) {
			defer wg.Done()
			if err := tl.follow(ctx); err != nil {
				fmt.Fprintf(os.Stderr, "[%s] erro no tail: %v\n", tl.prefix, err)
			}
		}(t)
	}

	// Espera Ctrl+C
	wg.Wait()

	// Encerrar tudo limpo
	for _, t := range tailers {
		t.close()
	}
	if metricsSrv != nil {
		_ = metricsSrv.Shutdown(context.Background())
	}

	fmt.Println("✅ logwatcher finalizado.")
}

// extractPrefix pega o conteúdo entre os dois primeiros colchetes “[<prefix>] ...”
func extractPrefix(s string) string {
	// opção simples e barata
	l := strings.IndexByte(s, '[')
	if l == -1 {
		return ""
	}
	r := strings.IndexByte(s[l+1:], ']')
	if r == -1 {
		return ""
	}
	return s[l+1 : l+1+r]
}

// Fim do codigo