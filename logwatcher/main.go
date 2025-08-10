package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
)

type tailer struct {
	path      string
	dir       string
	base      string
	fromStart bool
	pollEvery time.Duration
	rx        *regexp.Regexp

	f      *os.File
	reader *bufio.Reader

	lastSize int64
}

func newTailer(path string, fromStart bool, pollEvery time.Duration, rx *regexp.Regexp) *tailer {
	abs, _ := filepath.Abs(path)
	return &tailer{
		path:      abs,
		dir:       filepath.Dir(abs),
		base:      filepath.Base(abs),
		fromStart: fromStart,
		pollEvery: pollEvery,
		rx:        rx,
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
		// tail -f: ir pro final
		if _, err := f.Seek(0, os.SEEK_END); err != nil {
			return fmt.Errorf("seek end: %w", err)
		}
		// atualiza tamanho após seek
		if info, err := f.Stat(); err == nil {
			t.lastSize = info.Size()
		}
	}
	return nil
}

func (t *tailer) reopenFromStart() error {
	if t.f != nil {
		_ = t.f.Close()
	}
	f, err := os.Open(t.path)
	if err != nil {
		return err
	}
	t.f = f
	t.reader = bufio.NewReader(f)
	t.fromStart = true // esta reabertura começa do início
	t.lastSize = 0
	return nil
}

func (t *tailer) follow(ctx context.Context) error {
	if err := t.openInitial(); err != nil {
		return err
	}
	fmt.Printf("📄 seguindo: %s (from-start=%v, poll=%s)\n", t.path, t.fromStart, t.pollEvery)

	// watcher no diretório
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("fsnotify: %w", err)
	}
	defer watcher.Close()

	if err := watcher.Add(t.dir); err != nil {
		return fmt.Errorf("watcher add dir: %w", err)
	}
	fmt.Printf("👀 observando diretório: %s\n", t.dir)

	// sinais de SO
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	// loop principal
	for {
		select {
		case <-ctx.Done():
			return nil

		case s := <-sigCh:
			_ = s
			return nil

		default:
			// 1) tentar ler uma linha
			line, err := t.reader.ReadString('\n')
			if err == nil {
				// temos linha inteira
				if t.rx != nil {
					if t.rx.MatchString(line) {
						fmt.Printf("[MATCH] %s", line)
					}
				} else {
					fmt.Print(line)
				}
				// atualiza lastSize
				if info, e := t.f.Stat(); e == nil {
					t.lastSize = info.Size()
				}
				continue
			}

			// 2) sem nova linha: checar truncamento
			if info, e := t.f.Stat(); e == nil {
				cur := info.Size()
				if cur < t.lastSize {
					// arquivo truncado → reabrir do início
					log.Printf("🔁 truncamento detectado; reabrindo %s do início\n", t.path)
					if err := t.reopenFromStart(); err != nil {
						log.Printf("erro reabrindo após truncamento: %v", err)
					}
				} else {
					t.lastSize = cur
				}
			}

			// 3) checar eventos do watcher (não bloqueante)
			hasEvent := true
			for hasEvent {
				select {
				case ev := <-watcher.Events:
					t.handleFsEvent(ev)
				default:
					hasEvent = false
				}
			}

			// 4) aguardar um pouco (poll)
			time.Sleep(t.pollEvery)
		}
	}
}

func (t *tailer) handleFsEvent(ev fsnotify.Event) {
	// Queremos reagir quando:
	// - o arquivo observado (base) for Renamed/Removed (logrotate típico)
	// - for criado um novo arquivo com o mesmo nome (Create/Write no base)
	ename := filepath.Base(ev.Name)

	// alguns sistemas disparam WRITE/CHMOD o tempo todo — filtrar pelo base
	if ename != t.base {
		return
	}

	if ev.Op&(fsnotify.Rename|fsnotify.Remove) != 0 {
		// arquivo foi movido/renomeado/removido → tentar reabrir o NOVO base quando existir
		log.Printf("ℹ️  evento de rotação: %s (%s)\n", ev.Name, opString(ev.Op))
		waitForNew := time.NewTimer(500 * time.Millisecond)
		defer waitForNew.Stop()
		<-waitForNew.C
		if err := t.reopenAfterRotation(); err != nil {
			log.Printf("falha ao reabrir após rotação: %v", err)
		}
		return
	}

	if ev.Op&(fsnotify.Create) != 0 {
		// um novo arquivo com mesmo nome foi criado → reabrir imediatamente
		log.Printf("➕ novo arquivo base criado: %s\n", ev.Name)
		if err := t.reopenFromStart(); err != nil {
			log.Printf("falha ao reabrir novo arquivo: %v", err)
		}
		return
	}

	//  se WRITE no base e nosso file descriptor não é o mesmo inode, reabrir.
	if ev.Op&(fsnotify.Write) != 0 {
		// checar inode mudou
		same, err := t.sameInode()
		if err == nil && !same {
			log.Printf("🔄 inode mudou em WRITE; reabrindo %s\n", t.path)
			if err := t.reopenFromStart(); err != nil {
				log.Printf("falha ao reabrir após inode change: %v", err)
			}
		}
	}
}

func (t *tailer) reopenAfterRotation() error {
	// espera até o novo base aparecer
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(t.path); err == nil {
			return t.reopenFromStart()
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("novo arquivo base não apareceu em %s", t.path)
}

func (t *tailer) sameInode() (bool, error) {
	// compara inode do fd atual vs inode do path atual
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
		// em sistemas que não expõem Stat_t (ex.: Windows), caímos num fallback
		return strings.EqualFold(fi1.Name(), fi2.Name()), nil
	}
	return st1.Ino == st2.Ino && st1.Dev == st2.Dev, nil
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

func main() {
	filePath := flag.String("file", "", "Caminho do arquivo de log (ex.: /var/log/nginx/error.log)")
	pat := flag.String("pattern", "", "Regex para filtrar linhas (ex.: '(?i)error|critical')")
	fromStart := flag.Bool("from-start", false, "Ler desde o início (padrão: seguir a partir do fim)")
	pollEvery := flag.Duration("poll", 300*time.Millisecond, "Intervalo de polling quando não há novas linhas")
	flag.Parse()

	if *filePath == "" {
		fmt.Fprintln(os.Stderr, "uso: go run . -file /caminho/do/log [-pattern REGEX] [-from-start] [-poll 300ms]")
		os.Exit(2)
	}

	var rx *regexp.Regexp
	if *pat != "" {
		var err error
		rx, err = regexp.Compile(*pat)
		if err != nil {
			fmt.Fprintf(os.Stderr, "regex inválida: %v\n", err)
			os.Exit(2)
		}
	}

	fmt.Printf("🖥️  OS: %s | Arch: %s\n", runtime.GOOS, runtime.GOARCH)

	t := newTailer(*filePath, *fromStart, *pollEvery, rx)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := t.follow(ctx); err != nil {
		log.Fatalf("erro no tailer: %v", err)
	}
}