// go-diskmonitor é uma ferramenta simples para monitorar uso de disco em sistemas Unix.

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
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

// getDiskUsage coleta métricas do filesystem contendo "path".
// Nota: em Unix, Statfs funciona para qualquer caminho no FS (dir ou arquivo).
func getDiskUsage(path string) (total, used, freeUser uint64, percentUsed float64, err error) {
	var st syscall.Statfs_t
	if err = syscall.Statfs(path, &st); err != nil {
		return 0, 0, 0, 0, err
	}

	// Tamanhos em bytes: blocks * tamanho do bloco
	blockSize := uint64(st.Bsize)
	total = st.Blocks * blockSize

	// Livre "para o usuário" (não root) costuma ser Bavail.
	freeUser = st.Bavail * blockSize

	// Usado costuma considerar blocos reservados: used = total - Bfree*blockSize.
	// Isso alinha com ferramentas como 'df' (coluna Used).
	freeTotal := st.Bfree * blockSize
	used = total - freeTotal

	if total == 0 {
		return total, used, freeUser, 0, nil
	}
	percentUsed = (float64(used) / float64(total)) * 100
	return total, used, freeUser, percentUsed, nil
}

func main() {
	// Permite usar -path em vez de arg posicional (mais amigável).
	pathFlag := flag.String("path", "/", "Caminho (diretório ou arquivo) para analisar o uso de disco")
	human := flag.Bool("human", true, "Imprimir tamanhos em formato legível (IEC)")
	flag.Parse()

	path := *pathFlag
	// Se o usuário passou um argumento posicional, priorize-o.
	if flag.NArg() >= 1 {
		path = flag.Arg(0)
	}

	// Normaliza caminho (só pra exibir bonito).
	abs, _ := filepath.Abs(path)

	// Validação básica de existência/permissão.
	if fi, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Erro: caminho não existe: %s\n", abs)
			os.Exit(1)
		}
		// Outros erros (permissão, I/O, etc.)
		fmt.Fprintf(os.Stderr, "Erro ao acessar %s: %v\n", abs, err)
		os.Exit(1)
	} else {
		// Não é obrigatório ser diretório; apenas informativo.
		_ = fi
	}

	total, used, freeUser, pct, err := getDiskUsage(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao coletar uso de disco em %s: %v\n", abs, err)
		os.Exit(1)
	}

	fmt.Printf("Uso de disco em: %s\n", abs)
	if *human {
		fmt.Printf("Total: %s\n", humanizeBytesIEC(total))
		fmt.Printf("Usado: %s (%.2f%%)\n", humanizeBytesIEC(used), pct)
		fmt.Printf("Livre (usuário): %s\n", humanizeBytesIEC(freeUser))
	} else {
		// Em bytes “crus”
		fmt.Printf("Total: %d\n", total)
		fmt.Printf("Usado: %d (%.2f%%)\n", used, pct)
		fmt.Printf("Livre (usuário): %d\n", freeUser)
	}
}
