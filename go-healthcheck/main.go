package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	urlFlag := flag.String("url", "Endereço ou Site", "URL para healthcheck")
	timeoutFlag := flag.Int("timeout", 3, "timeout em segundos")
	outFlag := flag.String("out", "health.json", "arquivo de saída JSON")
	interactive := flag.Bool("interactive", true, "habilita perguntas interativas no terminal")
	metricsMode := flag.Bool("metrics", false, "ativa modo servidor de métricas para Prometheus")
	intervalFlag := flag.Int("interval", 15, "intervalo entre checks (segundos) no modo métricas")
	listenFlag := flag.String("listen", ":8080", "endereço/porta para expor /metrics")

	flag.Parse()

	url := *urlFlag
	timeout := *timeoutFlag
	out := *outFlag

	// --- Modo servidor de métricas (Prometheus) ---
	if *metricsMode {
		fmt.Println("Iniciando em modo servidor de métricas (Prometheus)...")
		startMetricsServer(url, timeout, time.Duration(*intervalFlag)*time.Second, *listenFlag)
		return
	}

	if *interactive {
		reader := bufio.NewReader(os.Stdin)

		fmt.Printf("URL para healthcheck [%s]: ", url)
		if input, _ := reader.ReadString('\n'); s(input) != "" {
			url = s(input)
		}

		fmt.Printf("Timeout em segundos [%d]: ", timeout)
		if input, _ := reader.ReadString('\n'); s(input) != "" {
			if v, err := strconv.Atoi(s(input)); err == nil && v > 0 {
				timeout = v
			} else {
				fmt.Println("Timeout inválido, mantendo padrão.")
			}
		}

		fmt.Printf("Arquivo de saída JSON [%s]: ", out)
		if input, _ := reader.ReadString('\n'); s(input) != "" {
			out = s(input)
		}
	}

	// Garantir protocolo
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	res, err := checkHTTP(url, timeout)
	if err != nil {
		fmt.Printf("Falha ao checar %s: %v\n", url, err)
	} else if res.Status == "UP" {
		fmt.Printf("%s saudável! (status %d, %dms)\n", res.URL, res.Code, res.ElapsedMS)
	} else {
		fmt.Printf("%s com problemas (status %d, %dms)\n", res.URL, res.Code, res.ElapsedMS)
	}

	// --- Append de JSON sem sobrescrever ---
	saveJSONAppend(out, res)
}

// Remove espaços e quebra de linha
func s(v string) string {
	return strings.TrimSpace(v)
}

// -----------------
// Função que faz append no JSON
// -----------------
func saveJSONAppend(filename string, newItem *HealthResult) {
	var list []HealthResult

	// Se arquivo já existe, carregar conteúdo
	if data, err := os.ReadFile(filename); err == nil {
		_ = json.Unmarshal(data, &list)
	}

	// Adicionar novo item
	list = append(list, *newItem)

	// Regravar tudo formatado
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		fmt.Println("erro gerando JSON:", err)
		return
	}

	if err := os.WriteFile(filename, data, 0o644); err != nil {
		fmt.Println("erro salvando JSON:", err)
		return
	}

	fmt.Println("Resultado adicionado em", filename)
}
