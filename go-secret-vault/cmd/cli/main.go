package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var base = env("GSV_ADDR", "http://localhost:8080")

func main() {
	if len(os.Args) < 2 { usage(); return }
	switch os.Args[1] {
	case "login": login()
	case "put": put()
	case "ls": ls()
	case "get": get()
	case "del": del()
	case "export-k8s": exportK8s()
	default: usage()
	}
}

func usage() {
	fmt.Println(`gsv CLI
Usage:
  gsv login -u USER -p PASS
  gsv put NAME -v VALUE [--ttl 1h]
  gsv ls
  gsv get ID
  gsv del ID
  gsv export-k8s ID [--namespace default] [--key VALUE]
`)
}

func tokenPath() string { home, _ := os.UserHomeDir(); return filepath.Join(home, ".gsv", "token") }
func readToken() string { b, _ := os.ReadFile(tokenPath()); return strings.TrimSpace(string(b)) }
func saveToken(tok string) { home, _ := os.UserHomeDir(); os.MkdirAll(filepath.Join(home, ".gsv"), 0o700); _ = os.WriteFile(tokenPath(), []byte(tok), 0o600) }

func login() {
	u, p := flag("-u"), flag("-p")
	if u == "" || p == "" { fmt.Println("login requires -u and -p"); return }
	resp, err := http.Post(base+"/login", "application/json", strings.NewReader(fmt.Sprintf(`{"username":"%s","password":"%s"}`, esc(u), esc(p))))
	check(err)
	defer resp.Body.Close()
	if resp.StatusCode != 200 { fmt.Println("login failed:", resp.Status); return }
	var out struct{ Token string `json:"token"` }
	json.NewDecoder(resp.Body).Decode(&out)
	saveToken(out.Token)
	fmt.Println("ok")
}

func put() {
	name := os.Args[2]
	val := flag("-v")
	ttl := flag("--ttl")
	body := fmt.Sprintf(`{"name":"%s","value":"%s","ttl":"%s"}`, esc(name), esc(val), esc(ttl))
	req("POST", "/secrets/", body)
}

func ls() { req("GET", "/secrets/", "") }

func get() { id := os.Args[2]; req("GET", "/secrets/"+url.PathEscape(id), "") }

func del() { id := os.Args[2]; r := reqRaw("DELETE", "/secrets/"+url.PathEscape(id), ""); if r.StatusCode == 204 { fmt.Println("deleted") } else { fmt.Println("error:", r.Status) } }

func exportK8s() {
	id := os.Args[2]
	ns := flag("--namespace"); if ns == "" { ns = "default" }
	key := flag("--key")
	resp := reqRaw("POST", "/secrets/"+url.PathEscape(id)+"/export/k8s", fmt.Sprintf(`{"namespace":"%s","key":"%s"}`, esc(ns), esc(key)))
	defer resp.Body.Close()
	b, _ := ioReadAll(resp.Body)
	fmt.Print(string(b))
}

// helpers

func req(method, path, body string) {
	r := reqRaw(method, path, body)
	defer r.Body.Close()
	b, _ := ioReadAll(r.Body)
	fmt.Println(string(b))
}

func reqRaw(method, path, body string) *http.Response {
	tok := readToken()
	req, _ := http.NewRequest(method, base+path, strings.NewReader(body))
	if tok != "" { req.Header.Set("Authorization", "Bearer "+tok) }
	if body != "" { req.Header.Set("Content-Type", "application/json") }
	client := &http.Client{Timeout: 15 * time.Second}
	r, err := client.Do(req)
	check(err)
	return r
}

func flag(name string) string { for i := 0; i < len(os.Args)-1; i++ { if os.Args[i] == name { return os.Args[i+1] } }; return "" }
func env(k, d string) string { if v := os.Getenv(k); v != "" { return v }; return d }
func esc(s string) string { return strings.ReplaceAll(s, "\"", "\\\"") }
func check(err error) { if err != nil { panic(err) } }

// tiny io util to avoid extra imports
func ioReadAll(r io.Reader) ([]byte, error) { var b strings.Builder; _, err := io.Copy(&b, r); return []byte(b.String()), err }

// imports we used implicitly
import "io"