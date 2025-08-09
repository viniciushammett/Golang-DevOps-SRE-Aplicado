package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"
)

func main() {
	url := flag.String("url", "http://localhost:8080/health", "URL to check health status")
	flag.Parse()

	client := &http.Client{Timeout: 3 * time.Second}

	start := time.Now()
	resp, err := client.Get(*url)
	ms := time.Since(start).Milliseconds()
	if err != nil {
		fmt.Printf("DOWN (erro) em %dms: %v\n", ms, err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		fmt.Printf("UP %d em %dms\n", resp.StatusCode, ms)
		os.Exit(0)
	} else {
		fmt.Printf("DOWN %d em %dms\n", resp.StatusCode, ms)
		os.Exit(1)
	}
}