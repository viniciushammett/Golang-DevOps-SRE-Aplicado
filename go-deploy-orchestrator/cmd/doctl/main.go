package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"
)

type deployReq struct {
	App       string            `json:"app"`
	Namespace string            `json:"namespace"`
	Image     string            `json:"image"`
	Strategy  string            `json:"strategy"` // canary|bluegreen
	Params    map[string]string `json:"params"`   // ex: canaryStep=20, maxError=0.02, maxP95=0.5
	RequireApproval bool        `json:"requireApproval"`
}

func main() {
	api := flag.String("api", "http://localhost:8080", "API base")
	app := flag.String("app", "", "app name (Deployment)")
	ns := flag.String("ns", "default", "namespace")
	img := flag.String("image", "", "image")
	strategy := flag.String("strategy", "canary", "canary|bluegreen")
	params := flag.String("params", "", "k=v,k=v")
	require := flag.Bool("approve", false, "require manual approval")
	flag.Parse()

	if *app == "" || *img == "" { fmt.Println("app and image required"); os.Exit(1) }

	p := map[string]string{}
	if *params != "" {
		for _, pair := range split(*params, ",") {
			kv := split(pair, "=")
			if len(kv)==2 { p[kv[0]] = kv[1] }
		}
	}

	body, _ := json.Marshal(deployReq{
		App: *app, Namespace: *ns, Image: *img, Strategy: *strategy, Params: p, RequireApproval: *require,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, *api + "/deploys", bytesReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil { panic(err) }
	defer resp.Body.Close()
	ioCopy(os.Stdout, resp.Body)
}

func split(s, sep string) []string { return strings.Split(s, sep) }

import (
	"bytes"
	"io"
	"strings"
)
func bytesReader(b []byte) *bytes.Reader { return bytes.NewReader(b) }
func ioCopy(dst io.Writer, src io.Reader) { _, _ = io.Copy(dst, src) }