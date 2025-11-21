// go-diskmonitor - Monitor de disco com suporte a Linux e Windows,
// análise de hotspots (logs/temp/cache), limpeza opcional e saída em formato Prometheus.

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// humanizeBytesIEC converte bytes para uma string legível (KiB, MiB, GiB, TiB).
func humanizeBytesIEC(b uint64) string {
	const (
		KiB = 1024
		MiB = 1024 * KiB
		GiB = 1024 * MiB
		TiB = 1024 * GiB
	)
	switch {
	case b >= TiB:
		return fmt.Sprintf("%.2f TiB", float64(b)/float64(TiB))
	case b >= GiB:
		return fmt.Sprintf("%.2f GiB", float64(b)/float64(GiB))
	case b >= MiB:
		return fmt.Sprintf("%.2f MiB", float64(b)/float64(MiB))
	case b >= KiB:
		return fmt.Sprintf("%.2f KiB", float64(b)/float64(KiB))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

// 👉 IMPORTANTE: NÃO declarar getDiskUsage aqui.
// Ela é implementada em disk_unix.go e disk_windows.go com build tags.

// getDiskUsage(path string) (total, used, freeUser uint64, percentUsed float64, err error)

type DirUsage struct {
	Path string
	Size uint64
}

// defaultScanDirs retorna os diretórios mais comuns de encher disco,
// variando conforme o sistema operacional.
func defaultScanDirs() []string {
	if runtime.GOOS == "windows" {
		winDir := os.Getenv("SystemRoot")
		if winDir == "" {
			winDir = `C:\Windows`
		}
		temp := os.TempDir()

		return []string{
			temp,
			filepath.Join(winDir, "Temp"),
			filepath.Join(winDir, "Logs"),
			filepath.Join(winDir, "SoftwareDistribution", "Download"),
		}
	}

	// Unix-like (Linux, etc.)
	return []string{
		"/var/log",
		"/var/tmp",
		"/tmp",
		"/var/cache",
	}
}

// tempDirs retorna os diretórios considerados seguros para limpeza automática.
func tempDirs() []string {
	if runtime.GOOS == "windows" {
		winDir := os.Getenv("SystemRoot")
		if winDir == "" {
			winDir = `C:\Windows`
		}
		return []string{
			os.TempDir(),
			filepath.Join(winDir, "Temp"),
		}
	}

	return []string{
		"/tmp",
		"/var/tmp",
	}
}

// getDirSize calcula o tamanho total (recursivo) de um diretório.
func getDirSize(path string) (uint64, error) {
	var total uint64

	err := filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			// Se der erro em algum arquivo, só pula.
			return nil
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		total += uint64(info.Size())
		return nil
	})

	return total, err
}

// findHotspots analisa os diretórios "críticos" e retorna os que possuem dados.
func findHotspots(dirs []string) []DirUsage {
	var result []DirUsage
	for _, d := range dirs {
		if _, err := os.Stat(d); err != nil {
			continue
		}
		size, err := getDirSize(d)
		if err != nil {
			continue
		}
		if size == 0 {
			continue
		}
		result = append(result, DirUsage{Path: d, Size: size})
	}
	return result
}

// cleanupTempDirs remove TODO o conteúdo dos diretórios de temporário.
func cleanupTempDirs(dirs []string) error {
	for _, d := range dirs {
		entries, err := os.ReadDir(d)
		if err != nil {
			continue
		}
		for _, e := range entries {
			target := filepath.Join(d, e.Name())
			if err := os.RemoveAll(target); err != nil {
				continue
			}
		}
	}
	return nil
}

// writePrometheusFile escreve métricas em formato Prometheus textfile collector.
func writePrometheusFile(path, mount string, pct float64, freeUser uint64) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Fprintln(f, "# HELP disk_usage_percent Disk usage percentage")
	fmt.Fprintln(f, "# TYPE disk_usage_percent gauge")
	fmt.Fprintf(f, "disk_usage_percent{mount=%q} %.2f\n", mount, pct)

	fmt.Fprintln(f, "# HELP disk_free_user_bytes Disk free space available to non-root users in bytes")
	fmt.Fprintln(f, "# TYPE disk_free_user_bytes gauge")
	fmt.Fprintf(f, "disk_free_user_bytes{mount=%q} %d\n", mount, freeUser)

	return nil
}

func askYesNo(prompt string) bool {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("%s [s/n]: ", prompt)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))
		switch input {
		case "s", "sim", "y", "yes":
			return true
		case "n", "nao", "não", "no":
			return false
		default:
			fmt.Println("Responda com 's' ou 'n'.")
		}
	}
}

func main() {
	pathFlag := flag.String("path", "/", "Caminho (diretório ou arquivo) para analisar o uso de disco")
	human := flag.Bool("human", true, "Imprimir tamanhos em formato legível (IEC)")
	threshold := flag.Float64("threshold", 80.0, "Percentual de uso a partir do qual dispara alerta/ação")
	autoClean := flag.Bool("auto-clean", false, "Limpeza automática de diretórios temporários quando threshold for atingido (sem perguntar)")
	promFile := flag.String("prom-file", "", "Arquivo de saída em formato Prometheus (ex: /var/lib/node_exporter/diskmonitor.prom)")

	flag.Parse()

	path := *pathFlag
	if flag.NArg() >= 1 {
		path = flag.Arg(0)
	}

	abs, _ := filepath.Abs(path)

	if fi, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Erro: caminho não existe: %s\n", abs)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Erro ao acessar %s: %v\n", abs, err)
		os.Exit(1)
	} else {
		_ = fi
	}

	total, used, freeUser, pct, err := getDiskUsage(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao coletar uso de disco em %s: %v\n", abs, err)
		os.Exit(1)
	}

	fmt.Printf(" Sistema operacional detectado: %s\n", runtime.GOOS)
	fmt.Printf(" Uso de disco em: %s\n", abs)

	if *human {
		fmt.Printf("  Total: %s\n", humanizeBytesIEC(total))
		fmt.Printf("  Usado: %s (%.2f%%)\n", humanizeBytesIEC(used), pct)
		fmt.Printf("  Livre (usuário): %s\n", humanizeBytesIEC(freeUser))
	} else {
		fmt.Printf("  Total: %d bytes\n", total)
		fmt.Printf("  Usado: %d bytes (%.2f%%)\n", used, pct)
		fmt.Printf("  Livre (usuário): %d bytes\n", freeUser)
	}

	if *promFile != "" {
		if err := writePrometheusFile(*promFile, abs, pct, freeUser); err != nil {
			fmt.Fprintf(os.Stderr, "  Falha ao escrever arquivo Prometheus: %v\n", err)
		} else {
			fmt.Printf(" Métricas escritas em formato Prometheus em: %s\n", *promFile)
		}
	}

	if pct < *threshold {
		fmt.Printf(" Uso de disco abaixo do threshold (%.2f%% < %.2f%%). Nenhuma ação necessária.\n", pct, *threshold)
		return
	}

	fmt.Printf("  ATENÇÃO: uso de disco acima do threshold (%.2f%% >= %.2f%%)\n", pct, *threshold)
	fmt.Println(" Analisando diretórios mais comuns que enchem o disco (logs, temp, cache)...")

	hotDirs := findHotspots(defaultScanDirs())
	if len(hotDirs) == 0 {
		fmt.Println("Nenhum hotspot significativo encontrado nos diretórios padrão.")
	} else {
		fmt.Println(" Hotspots encontrados:")
		for _, d := range hotDirs {
			fmt.Printf("  - %s -> %s\n", d.Path, humanizeBytesIEC(d.Size))
		}
	}

	doClean := *autoClean
	if !*autoClean {
		doClean = askYesNo("Deseja que eu faça a limpeza AUTOMÁTICA dos diretórios temporários?")
	}

	if doClean {
		fmt.Println("🧹 Iniciando limpeza de diretórios temporários...")
		if err := cleanupTempDirs(tempDirs()); err != nil {
			fmt.Fprintf(os.Stderr, "Erro durante a limpeza: %v\n", err)
		} else {
			fmt.Println(" Limpeza de temporários concluída.")
		}

		total, used, freeUser, pct, err = getDiskUsage(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao recalcular uso de disco após limpeza: %v\n", err)
			os.Exit(1)
		}

		fmt.Println(" Uso de disco após limpeza:")
		if *human {
			fmt.Printf("  Total: %s\n", humanizeBytesIEC(total))
			fmt.Printf("  Usado: %s (%.2f%%)\n", humanizeBytesIEC(used), pct)
			fmt.Printf("  Livre (usuário): %s\n", humanizeBytesIEC(freeUser))
		} else {
			fmt.Printf("  Total: %d bytes\n", total)
			fmt.Printf("  Usado: %d bytes (%.2f%%)\n", used, pct)
			fmt.Printf("  Livre (usuário): %d bytes\n", freeUser)
		}

		if *promFile != "" {
			if err := writePrometheusFile(*promFile, abs, pct, freeUser); err != nil {
				fmt.Fprintf(os.Stderr, "⚠️  Falha ao escrever arquivo Prometheus após limpeza: %v\n", err)
			} else {
				fmt.Printf(" Métricas ATUALIZADAS em formato Prometheus em: %s\n", *promFile)
			}
		}
	} else {
		fmt.Println(" Limpeza automática NÃO será executada.")
		fmt.Println("Você pode limpar manualmente os diretórios listados acima e depois rodar o go-diskmonitor novamente.")
	}
}
