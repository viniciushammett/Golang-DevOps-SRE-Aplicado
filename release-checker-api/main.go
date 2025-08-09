package main

import (
	"enconde/json"
	"fmt"
	"net/http"
	"os"
	"flag"
)

// Estrutura para mapear a resposta JSON da API do GitHub

type Release struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

func main() {
	// Flags para entrada do usuário
	owner := flag.String("owner", "", "Nome do proprietário do repositório")
	repo := flag.String("repo", "", "Nome do repositório")
	flag.Parse()

	// Validação básica
	if *owner == "" || *repo == "" {
		fmt.Println("Uso: go run main.go -owner=<proprietário> -repo=<repositório>")
		os.Exit(1)
	}

	// Construção da URL da API do GitHub
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", *owner, *repo)

	// Requisição HTTP para a API do GitHub
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Erro ao acessar API:", err)
		os.Exit(1)
	}

	defer resp.Body.Close()

	// Decodificação da resposta JSON
	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		fmt.Printf("Erro ao decodificar JSON:", err)
		os.Exit(1)
	}

	// Exibição da última versão
	fmt.Printf("Última release de %s/%s:\n", *owner, *repo)
	fmt.Printf("Versão: %s\n", release.TagName)
	fmt.Printf("URL: %s\n", release.HTMLURL)
}

