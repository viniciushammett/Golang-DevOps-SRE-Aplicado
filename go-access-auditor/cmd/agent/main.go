package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"
)

type Event struct {
	When     time.Time         `json:"when"`
	User     string            `json:"user"`
	Host     string            `json:"host"`
	Source   string            `json:"source"`   // e.g. "ssh", "bash", "kubectl"
	Command  string            `json:"command"`  // linha de comando
	Meta     map[string]string `json:"meta"`     // chave/valor (tty, ip, pod, ns...)
}

func main() {
	api := flag.String("api", "http://localhost:8080", "collector base URL")
	token := flag.String("token", "", "bearer token (optional)")
	user := flag.String("user", "", "user")
	host := flag.String("host", "", "host")
	src  := flag.String("source", "bash", "source (bash|ssh|kubectl|psql|mysql)")
	cmd  := flag.String("cmd", "", "command (if empty, read stdin lines)")
	flag.Parse()

	if *user == "" { *user = os.Getenv("USER") }
	if *host == "" { h,_ := os.Hostname(); *host = h }

	send := func(line string) {
		ev := Event{
			When: time.Now(), User: *user, Host: *host,
			Source: *src, Command: line, Meta: map[string]string{},
		}
		buf, _ := json.Marshal(ev)
		req, _ := http.NewRequest("POST", *api+"/v1/events", bytes.NewReader(buf))
		req.Header.Set("Content-Type","application/json")
		if *token != "" { req.Header.Set("Authorization","Bearer "+*token) }
		resp, err := http.DefaultClient.Do(req)
		if err != nil { fmt.Fprintln(os.Stderr, "send error:", err); return }
		resp.Body.Close()
	}

	if *cmd != "" {
		send(*cmd); return
	}
	// modo stdin
	st, _ := os.Stdin.Stat()
	if (st.Mode() & os.ModeCharDevice) != 0 {
		fmt.Fprintln(os.Stderr, "Usage: echo 'rm -rf /' | agent -source bash  OR  agent -cmd 'kubectl get pods'")
		return
	}
	var b bytes.Buffer
	_, _ = b.ReadFrom(os.Stdin)
	for _, line := range bytes.Split(bytes.TrimSpace(b.Bytes()), []byte{'\n'}) {
		if len(line) > 0 { send(string(line)) }
	}
}
// This code is a simple command-line agent that sends events to a collector API.
// It captures user commands executed in a terminal and sends them as JSON events.
// The agent can be used with different sources like bash, ssh, kubectl, etc.
// It supports reading commands from standard input or directly from command line arguments.
// The events include metadata such as the user, host, source, command, and a timestamp.
// The agent can be configured with a base API URL and an optional bearer token for authentication.