package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"syscall"
	"time"
)

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

	abs, _ := filepath.Abs(*filePath)
	f, err := os.Open(*filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "erro ao abrir arquivo: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	// Por padrão, começamos do fim (tail -f). Com -from-start, do começo.
	if !*fromStart {
		if _, err := f.Seek(0, os.SEEK_END); err != nil {
			fmt.Fprintf(os.Stderr, "erro ao posicionar no fim: %v\n", err)
			os.Exit(1)
		}
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

	fmt.Printf("📄 seguindo: %s  (from-start=%v, poll=%s)\n", abs, *fromStart, pollEvery.String())
	if rx != nil {
		fmt.Printf("🔎 filtro regex: %q\n", rx.String())
	}
	fmt.Println("pressione Ctrl+C para sair…")

	// Contexto para terminar com Ctrl+C
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	reader := bufio.NewReader(f)

	for {
		select {
		case <-ctx.Done():
			fmt.Println("\nEncerrando…")
			return
		default:
			// Tenta ler uma linha
			line, err := reader.ReadString('\n')
			if err != nil {
				// Sem nova linha disponível: aguarda e tenta de novo (polling).
				time.Sleep(*pollEvery)
				continue
			}

			// Se tiver regex, só imprime se casar; senão, imprime tudo
			if rx != nil {
				if rx.MatchString(line) {
					fmt.Printf("[MATCH] %s", line)
				}
			} else {
				fmt.Print(line)
			}
		}
	}
}
