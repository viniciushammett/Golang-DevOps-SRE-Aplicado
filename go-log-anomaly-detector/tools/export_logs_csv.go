package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/viniciushammett/go-log-anomaly-detector/internal/store"
)

func main() {
	var (
		dbPath  = flag.String("db", "data/log-anomaly.db", "caminho do BoltDB")
		outPath = flag.String("out", "logs.csv", "arquivo CSV de saída")
	)
	flag.Parse()

	st, err := store.Open(*dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "erro ao abrir db: %v\n", err)
		os.Exit(1)
	}
	defer st.Close()

	f, err := os.Create(*outPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "erro ao criar arquivo: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	// Cabeçalho
	if err := w.Write([]string{"ts", "source", "msg"}); err != nil {
		fmt.Fprintf(os.Stderr, "erro ao escrever cabeçalho: %v\n", err)
		os.Exit(1)
	}

	n := 0
	err = st.IterateLogs(func(lr store.LogRecord) bool {
		ts := lr.TS.UTC().Format(time.RFC3339)
		row := []string{ts, lr.Source, lr.Msg}
		if err := w.Write(row); err != nil {
			fmt.Fprintf(os.Stderr, "erro ao escrever linha: %v\n", err)
			return false
		}
		n++
		return true
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "erro ao iterar logs: %v\n", err)
		os.Exit(1)
	}
	w.Flush()
	if err := w.Error(); err != nil {
		fmt.Fprintf(os.Stderr, "erro ao finalizar csv: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Exportados %d logs para %s\n", n, *outPath)
}